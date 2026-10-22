// Package httpserver builds the HTTP handler used by both the local server and Vercel.
package httpserver

import (
	"fmt"
	"net"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/internal/access"
	"github.com/fluentfox/api/internal/auth"
	"github.com/fluentfox/api/internal/classes"
	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/content"
	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/internal/profile"
	"github.com/fluentfox/api/internal/progress"
	"github.com/fluentfox/api/internal/services"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/alert"
	"github.com/fluentfox/api/pkg/database"
	"github.com/fluentfox/api/pkg/emailcheck"
	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/ipintel"
	"github.com/fluentfox/api/pkg/mailer"
	"github.com/fluentfox/api/pkg/middleware"
	"github.com/fluentfox/api/pkg/storage"
	"github.com/fluentfox/api/pkg/token"
	"github.com/fluentfox/api/pkg/validator"
	"github.com/fluentfox/api/pkg/youtube"
)

func NewHandler(cfg *config.Config, log *zap.Logger) (http.Handler, func(), error) {
	db, err := database.NewDB(cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get database connection: %w", err)
	}
	cleanup := func() {
		if err := sqlDB.Close(); err != nil {
			log.Error("close database connection", zap.Error(err))
		}
	}

	tokenMaker := token.NewMaker(
		cfg.JWTAccessSecret,
		cfg.JWTAccessExpiryMinutes, cfg.JWTRefreshExpiryDays,
	)
	var mailTransport mailer.Mailer = mailer.NewResendMailer(cfg.ResendAPIKey, cfg.EmailFromAddress, common.EMAIL_SEND_TIMEOUT)
	mailService := services.NewMailService(mailTransport, cfg.AppURL, log)
	// Everything built from here on logs through the alerting logger. mailService
	// keeps the plain one, so its own send failures never raise alerts.
	log = withServerErrorAlerts(log, cfg, mailTransport)
	userRepo := users.NewRepository(db)

	emailChecker, err := emailcheck.New(net.DefaultResolver, common.EMAIL_MX_LOOKUP_TIMEOUT, cfg.EmailMXCheckEnabled)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("load email domain lists: %w", err)
	}

	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("invalid TRUSTED_PROXIES: %w", err)
	}
	r.Use(gin.CustomRecovery(recoverPanic(log)))
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.Logger(log))
	r.Use(middleware.RequestInfoCapture(cfg.VercelDeployment))
	r.Use(otelgin.Middleware("fluentfox-api"))

	humaConfig := huma.DefaultConfig("FluentFox API", "1.0.0")
	humaConfig.CreateHooks = nil
	api := humagin.New(r, humaConfig)
	humautil.InitHumaErrors()
	humautil.RegisterBearerScheme(api)
	api.UseMiddleware(humautil.AuthMiddleware(api, tokenMaker, cfg.AllowedOrigins...))

	staleReleaser := auth.NewStaleAccountReleaser(userRepo, log)
	var locator auth.IPLocator
	if cfg.VPNAPIKey != "" {
		locator = ipintel.NewVPNAPI(cfg.VPNAPIKey, common.IP_LOOKUP_TIMEOUT)
	}
	loginChallenges := auth.NewLoginChallengeService(userRepo, mailService, cfg.JWTAccessSecret, log)
	loginService := auth.NewLogin(userRepo, tokenMaker, locator, loginChallenges, log)
	googleOAuth, err := auth.NewGoogleOAuthService(cfg, userRepo, loginService, log)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("configure Google OAuth: %w", err)
	}
	xOAuth, err := auth.NewXOAuthService(cfg, userRepo, loginService, log)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("configure X OAuth: %w", err)
	}
	authHandler := auth.NewHandler(
		auth.NewAuthService(userRepo, mailService, emailChecker, staleReleaser, loginService, log),
		auth.NewTokenVerificationService(userRepo, log),
		auth.NewResendVerificationService(userRepo, mailService, log),
		loginService,
		auth.NewChangeEmailService(userRepo, mailService, emailChecker, staleReleaser, log),
		auth.NewPasswordResetService(userRepo, mailService, cfg.JWTAccessSecret, log),
		log, validator.New(), cfg.JWTRefreshExpiryDays,
	)
	auth.RegisterRoutes(api, authHandler)
	refreshMaxAge := cfg.JWTRefreshExpiryDays * 24 * 60 * 60
	auth.RegisterGoogleOAuthRoutes(r, googleOAuth, tokenMaker, refreshMaxAge, log)
	auth.RegisterXOAuthRoutes(r, xOAuth, tokenMaker, refreshMaxAge, log)

	registerProfileRoutes(api, cfg, userRepo, log)

	accessRepo := access.NewRepository(db)
	permissionChecker := access.NewChecker(accessRepo, log)
	registerAccessRoutes(api, accessRepo, permissionChecker, userRepo, mailService, log)
	registerClassesRoutes(api, cfg, classes.NewRepository(db), permissionChecker,
		access.NewEditableChannels(permissionChecker, accessRepo), log)

	contentRepo := content.NewRepository(db)
	content.RegisterRoutes(api, content.NewHandler(
		content.NewKanaContentService(contentRepo, cfg.StrokesBaseURL, log),
		content.NewKanjiContentService(contentRepo, cfg.StrokesBaseURL, log),
		log,
	))

	learnRepo := learn.NewRepository(db)
	learnCatalog := learn.NewCatalog(contentRepo)
	learnQuestions := learn.NewQuestionBuilder(nil)
	learn.RegisterRoutes(api, learn.NewHandler(
		learn.NewLearningSettingsService(learnRepo, log),
		learn.NewSummaryService(learnRepo, learnCatalog, log),
		learn.NewStartLessonService(learnRepo, learnCatalog, learnQuestions, log),
		learn.NewStartReviewService(learnRepo, learnCatalog, learnQuestions, log),
		learn.NewStartPracticeService(learnRepo, learnCatalog, learnQuestions, nil, log),
		learn.NewStartTestService(learnRepo, learnCatalog, learnQuestions, nil, log),
		learn.NewSubmitAnswersService(learnRepo, learnCatalog, log),
		learn.NewCompleteSessionService(learnRepo, learn.NewTestScoreService(learnRepo, learnCatalog, nil, log), log),
		log,
	))
	learn.RegisterPathRoutes(api, learn.NewPathHandler(
		learn.NewUnitProgressService(learnRepo, learnCatalog, log),
		learn.NewKanjiNoteService(learnRepo, log),
		learn.NewTestHistoryService(learnRepo, log),
		log,
	))
	progress.RegisterRoutes(api, progress.NewHandler(
		progress.NewOverviewService(learnRepo, learnCatalog, log),
		progress.NewForecastService(learnRepo, log),
		progress.NewCalendarService(learnRepo, log),
		progress.NewWeakItemsService(learnRepo, learnCatalog, log),
		progress.NewWeakGroupsService(learnRepo, learnCatalog, log),
		progress.NewConfusionService(learnRepo, learnCatalog, log),
		progress.NewItemHistoryService(learnRepo, learnCatalog, contentRepo, log),
		progress.NewDashboardService(learnRepo, learnCatalog, log),
		log,
	))

	r.GET("/health", func(c *gin.Context) {
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "error",
				"error":  "database unreachable",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "env": cfg.AppEnv})
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return r, cleanup, nil
}

// Tees an alert core onto the logger so each logged 500 also emails
// ALERT_EMAIL_TO. The alert core reports its own failures to the plain logger,
// so a failed alert cannot trigger another one.
func withServerErrorAlerts(log *zap.Logger, cfg *config.Config, m mailer.Mailer) *zap.Logger {
	if cfg.AlertEmailTo == "" {
		return log
	}
	alerts := alert.NewCore(
		services.NewServerErrorAlertMailer(m, cfg.AlertEmailTo, cfg.AppURL, cfg.AppEnv),
		alert.Config{
			Messages: []string{humautil.InternalErrorLogMessage},
			Interval: common.SERVER_ERROR_ALERT_INTERVAL,
			Timeout:  common.EMAIL_SEND_TIMEOUT,
		},
		log,
	)
	return log.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, alerts)
	}))
}

// Logs a panic like any other 500 so it carries the request_id and raises an alert.
// The Logger middleware runs after this one, so c.Request already holds the request logger.
func recoverPanic(log *zap.Logger) gin.RecoveryFunc {
	return func(c *gin.Context, recovered any) {
		err, ok := recovered.(error)
		if !ok {
			err = fmt.Errorf("%v", recovered)
		}
		middleware.LoggerFromContext(c.Request.Context(), log).Error(humautil.InternalErrorLogMessage,
			zap.Error(fmt.Errorf("panic: %w", err)),
			zap.String("path", c.Request.URL.Path),
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func registerProfileRoutes(api huma.API, cfg *config.Config, userRepo *users.Repository, log *zap.Logger) {
	// A nil interface (not a nil *R2Store) is what lets the services see
	// "uploads off" when R2 is not configured.
	var store profile.ObjectStore
	if cfg.R2Enabled() {
		store = storage.NewR2Store(cfg.R2AccountID, cfg.R2AccessKey, cfg.R2SecretKey, cfg.R2BucketName, cfg.R2PublicURL)
	}
	avatars := profile.NewAvatarResolver(cfg.R2PublicURL)
	profileService := profile.NewProfileService(userRepo, avatars, log)
	profile.RegisterRoutes(api,
		profile.NewProfileHandler(
			profileService,
			profile.NewUpdateProfileService(userRepo, profileService, avatars, store, log),
			profile.NewAvatarService(userRepo, store, avatars, log),
			avatars,
			log,
		),
		profile.NewSessionHandler(
			profile.NewSessionListService(userRepo, log),
			profile.NewSessionRevokeService(userRepo, log),
			profile.NewSessionLocationService(userRepo, log),
			log,
		),
	)
}

func registerAccessRoutes(
	api huma.API, repo *access.Repository, checker *access.Checker, userRepo *users.Repository, mail *services.MailService, log *zap.Logger,
) {
	access.RegisterRoutes(api,
		access.NewPermissionHandler(
			access.NewMyPermissionsService(repo, log),
			access.NewListPermissionsService(checker, repo, log),
			access.NewGrantPermissionService(checker, repo, userRepo, log),
			access.NewRevokePermissionService(checker, repo, log),
			log,
		),
		access.NewInviteHandler(access.InviteServices{
			Create:  access.NewCreateInviteService(checker, repo, mail, log),
			List:    access.NewListInvitesService(checker, repo, log),
			Resend:  access.NewResendInviteService(checker, repo, mail, log),
			Revoke:  access.NewRevokeInviteService(checker, repo, log),
			Preview: access.NewPreviewInviteService(repo, log),
			Accept:  access.NewAcceptInviteService(repo, userRepo, log),
		}, log),
	)
}

func registerClassesRoutes(
	api huma.API, cfg *config.Config, repo *classes.Repository, perms classes.PermissionChecker, grants classes.GrantLister, log *zap.Logger,
) {
	// Nil interfaces (not nil pointers) let the services answer 503 when R2
	// or the YouTube key is not configured.
	var store classes.ObjectStore
	if cfg.R2Enabled() {
		store = storage.NewR2Store(cfg.R2AccountID, cfg.R2AccessKey, cfg.R2SecretKey, cfg.R2BucketName, cfg.R2PublicURL)
	}
	var ytChannels classes.YouTubeChannels
	var ytPlaylists classes.YouTubePlaylists
	if cfg.YouTubeAPIKey != "" {
		client := youtube.NewClient(cfg.YouTubeAPIKey, common.YOUTUBE_REQUEST_TIMEOUT)
		ytChannels, ytPlaylists = client, client
	}
	courseReader := classes.NewCourseAdminGetService(repo, perms, log)
	classes.RegisterRoutes(api,
		classes.NewCatalogHandler(classes.NewListCoursesService(repo, log), classes.NewCourseDetailService(repo, log), log),
		classes.NewChannelAdminHandler(&classes.ChannelAdminServices{
			List:       classes.NewChannelListService(repo, grants, log),
			Read:       classes.NewChannelReadService(repo, perms, log),
			Create:     classes.NewChannelCreateService(repo, perms, log),
			Update:     classes.NewChannelUpdateService(repo, perms, log),
			Media:      classes.NewChannelMediaService(repo, perms, store, cfg.R2PublicURL, log),
			IntroVideo: classes.NewIntroVideoService(repo, perms, log),
			Reset:      classes.NewResetFieldService(repo, perms, store, cfg.R2PublicURL, log),
			Visibility: classes.NewChannelVisibilityService(repo, perms, log),
			Sync:       classes.NewChannelSyncService(repo, ytChannels, perms, log),
			History:    classes.NewChannelHistoryService(repo, perms, log),
		}, log),
		classes.NewCourseAdminHandler(&classes.CourseAdminServices{
			List:       classes.NewCourseAdminListService(repo, perms, log),
			Get:        courseReader,
			Create:     classes.NewCourseCreateService(repo, courseReader, perms, log),
			Update:     classes.NewCourseUpdateService(repo, courseReader, perms, log),
			Delete:     classes.NewCourseDeleteService(repo, perms, log),
			Visibility: classes.NewCourseVisibilityService(repo, courseReader, perms, log),
			Sync:       classes.NewPlaylistSyncService(repo, ytPlaylists, courseReader, perms, log),
		}, log),
	)
}
