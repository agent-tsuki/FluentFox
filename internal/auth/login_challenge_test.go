package auth

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/ipintel"
	"github.com/fluentfox/api/pkg/middleware"
)

const testCodeSecret = "code-secret"

var sixDigits = regexp.MustCompile(`^\d{6}$`)

type fakeLocator struct {
	result ipintel.Result
	err    error
	calls  int
}

func (f *fakeLocator) Lookup(context.Context, netip.Addr) (ipintel.Result, error) {
	f.calls++
	return f.result, f.err
}

type fakeChallenger struct {
	started  bool
	startErr error

	consumeUser uuid.UUID
	consumeLoc  users.SessionLocation
	consumeErr  error

	trusted      bool
	trustedErr   error
	trustedCalls int

	remembered  bool
	rememberErr error
}

func (f *fakeChallenger) IsTrustedNetwork(context.Context, uuid.UUID) (bool, error) {
	f.trustedCalls++
	return f.trusted, f.trustedErr
}

func (f *fakeChallenger) RememberNetwork(context.Context, uuid.UUID) error {
	f.remembered = true
	return f.rememberErr
}

func (f *fakeChallenger) Start(context.Context, *users.User, *users.SessionLocation) (LoginChallengeResponse, error) {
	f.started = true
	return LoginChallengeResponse{ChallengeID: uuid.New(), Reason: loginChallengeReasonVPN}, f.startErr
}

func (f *fakeChallenger) Consume(context.Context, uuid.UUID, string) (uuid.UUID, users.SessionLocation, error) {
	return f.consumeUser, f.consumeLoc, f.consumeErr
}

type fakeLoginCodeMailer struct {
	err   error
	to    string
	code  string
	where string
}

func (f *fakeLoginCodeMailer) SendLoginCode(_ context.Context, to, _, code, where string) error {
	f.to, f.code, f.where = to, code, where
	return f.err
}

func requestFrom(ip string) context.Context {
	return middleware.ContextWithRequestInfo(context.Background(), middleware.RequestInfo{IP: netip.MustParseAddr(ip)})
}

func TestLoginService_LoginLocationAndVPN(t *testing.T) {
	user := users.User{ID: uuid.New(), Email: "aiko@example.com", IsActive: true}
	tokyo := ipintel.Result{CountryCode: "JP", City: "Tokyo", Latitude: 35.689487, Longitude: 139.691711}
	vpn := ipintel.Result{CountryCode: "NL", City: "Amsterdam", IsVPN: true}

	tests := []struct {
		name          string
		ip            string
		locator       *fakeLocator
		challenger    *fakeChallenger
		wantChallenge bool
		wantCity      string
		wantLookups   int
		wantTrusted   int
		wantVPN       bool
	}{
		{name: "VPN login from an unknown network gets an email code and no session", ip: "203.0.113.7", locator: &fakeLocator{result: vpn}, challenger: &fakeChallenger{}, wantChallenge: true, wantLookups: 1, wantTrusted: 1},
		{name: "VPN login from a trusted network skips the code and keeps the VPN flag", ip: "203.0.113.7", locator: &fakeLocator{result: vpn}, challenger: &fakeChallenger{trusted: true}, wantCity: "Amsterdam", wantLookups: 1, wantTrusted: 1, wantVPN: true},
		{name: "VPN login falls back to a code when the trusted lookup fails", ip: "203.0.113.7", locator: &fakeLocator{result: vpn}, challenger: &fakeChallenger{trustedErr: errors.New("db down")}, wantChallenge: true, wantLookups: 1, wantTrusted: 1},
		{name: "normal login stores the IP location", ip: "203.0.113.7", locator: &fakeLocator{result: tokyo}, challenger: &fakeChallenger{}, wantCity: "Tokyo", wantLookups: 1},
		{name: "failed lookup still logs in, without a location", ip: "203.0.113.7", locator: &fakeLocator{err: errors.New("timeout")}, challenger: &fakeChallenger{}, wantLookups: 1},
		{name: "private address is not looked up", ip: "192.168.1.5", locator: &fakeLocator{result: vpn}, challenger: &fakeChallenger{}, wantLookups: 0},
		{name: "VPN without a challenger configured still logs in", ip: "203.0.113.7", locator: &fakeLocator{result: vpn}, wantCity: "Amsterdam", wantLookups: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUserRepo{byEmail: map[string]users.User{user.Email: withPassword(t, user)}}
			var challenger LoginChallenger
			if tt.challenger != nil {
				challenger = tt.challenger
			}
			svc := NewLogin(repo, newTestMaker(), tt.locator, challenger, nopLogger())

			resp, err := svc.Login(requestFrom(tt.ip), LoginRequest{Email: user.Email, Password: loginTestPassword})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.locator.calls != tt.wantLookups {
				t.Errorf("lookups = %d, want %d", tt.locator.calls, tt.wantLookups)
			}
			if tt.challenger != nil {
				if tt.challenger.trustedCalls != tt.wantTrusted {
					t.Errorf("trusted lookups = %d, want %d", tt.challenger.trustedCalls, tt.wantTrusted)
				}
				if tt.challenger.started != tt.wantChallenge {
					t.Errorf("challenge started = %v, want %v", tt.challenger.started, tt.wantChallenge)
				}
			}
			if tt.wantChallenge {
				if resp.LoginChallenge == nil || resp.AccessToken != "" || repo.createdSessions != 0 {
					t.Fatalf("want a challenge and no session, got %+v (sessions %d)", resp, repo.createdSessions)
				}
				return
			}
			assertIssuedTokens(t, repo, resp)
			if got := repo.createdSession.Location.City; got != tt.wantCity {
				t.Errorf("session city = %q, want %q", got, tt.wantCity)
			}
			if got := repo.createdSession.Location.IsVPN; tt.wantVPN && (got == nil || !*got) {
				t.Errorf("session IsVPN = %v, want true", got)
			}
		})
	}
}

func TestLoginService_RoundsIPCoordinates(t *testing.T) {
	user := users.User{ID: uuid.New(), Email: "aiko@example.com", IsActive: true}
	repo := &fakeUserRepo{byEmail: map[string]users.User{user.Email: withPassword(t, user)}}
	locator := &fakeLocator{result: ipintel.Result{City: "Tokyo", Latitude: 35.689487, Longitude: 139.691711}}

	if _, err := NewLogin(repo, newTestMaker(), locator, nil, nopLogger()).
		Login(requestFrom("203.0.113.7"), LoginRequest{Email: user.Email, Password: loginTestPassword}); err != nil {
		t.Fatalf("login: %v", err)
	}

	loc := repo.createdSession.Location
	if loc.Latitude == nil || *loc.Latitude != 35.689 || *loc.Longitude != 139.692 || loc.Source != users.LocationSourceIP {
		t.Errorf("location = %+v, want rounded IP coordinates", loc)
	}
}

func TestLoginService_VerifyLoginCode(t *testing.T) {
	userID := uuid.New()
	active := users.User{ID: userID, IsActive: true}
	stored := users.SessionLocation{City: "Amsterdam", Source: users.LocationSourceIP}

	tests := []struct {
		name       string
		repo       *fakeUserRepo
		challenger *fakeChallenger
		wantErr    error

		wantRemembered bool
	}{
		{name: "wrong code is rejected and the network is not remembered", repo: &fakeUserRepo{}, challenger: &fakeChallenger{consumeErr: exceptions.ErrLoginCodeInvalid}, wantErr: exceptions.ErrLoginCodeInvalid},
		{name: "deleted user is rejected", repo: &fakeUserRepo{byIDErr: fmt.Errorf("x: %w", gorm.ErrRecordNotFound)}, challenger: &fakeChallenger{consumeUser: userID}, wantErr: exceptions.ErrLoginCodeInvalid},
		{name: "account suspended after the code was sent is rejected", repo: &fakeUserRepo{byID: users.User{ID: userID}}, challenger: &fakeChallenger{consumeUser: userID}, wantErr: exceptions.ErrAccountSuspended},
		{name: "correct code opens a session at the stored location and remembers the network", repo: &fakeUserRepo{byID: active}, challenger: &fakeChallenger{consumeUser: userID, consumeLoc: stored}, wantRemembered: true},
		{name: "failure to remember the network still logs in", repo: &fakeUserRepo{byID: active}, challenger: &fakeChallenger{consumeUser: userID, consumeLoc: stored, rememberErr: errors.New("db down")}, wantRemembered: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewLogin(tt.repo, newTestMaker(), nil, tt.challenger, nopLogger())
			resp, err := svc.VerifyLoginCode(context.Background(), VerifyLoginCodeRequest{ChallengeID: uuid.New(), Code: "123456"})
			if tt.challenger.remembered != tt.wantRemembered {
				t.Errorf("network remembered = %v, want %v", tt.challenger.remembered, tt.wantRemembered)
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertIssuedTokens(t, tt.repo, resp)
			if tt.repo.createdSession.Location.City != stored.City {
				t.Errorf("session location = %+v, want %+v", tt.repo.createdSession.Location, stored)
			}
		})
	}
}

func TestLoginChallengeService_Start(t *testing.T) {
	user := users.User{ID: uuid.New(), Email: "aiko@example.com"}
	recent := time.Now().UTC().Add(-10 * time.Second)

	tests := []struct {
		name    string
		repo    *fakeUserRepo
		mailErr error
		wantErr error
	}{
		{name: "sends a code within the limits", repo: &fakeUserRepo{}},
		{name: "a second code inside the cooldown is refused", repo: &fakeUserRepo{challengeStats: users.LoginChallengeStats{Count: 1, Latest: &recent}}, wantErr: exceptions.ErrLoginCodeCooldown},
		{name: "the daily limit is enforced", repo: &fakeUserRepo{challengeStats: users.LoginChallengeStats{Count: common.LOGIN_CODE_DAILY_LIMIT}}, wantErr: exceptions.ErrLoginCodeDailyLimit},
		{name: "a failed email send is reported", repo: &fakeUserRepo{}, mailErr: errors.New("smtp down"), wantErr: exceptions.ErrLoginCodeEmailSendFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mailer := &fakeLoginCodeMailer{err: tt.mailErr}
			svc := NewLoginChallengeService(tt.repo, mailer, testCodeSecret, nopLogger())

			resp, err := svc.Start(context.Background(), &user, &users.SessionLocation{City: "Amsterdam", CountryCode: "NL"})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mailer.to != user.Email || !sixDigits.MatchString(mailer.code) || mailer.where != "Amsterdam, NL" {
				t.Errorf("mail = %+v, want a 6-digit code to %s from Amsterdam, NL", mailer, user.Email)
			}
			if c := tt.repo.createdChallenge; c == nil || c.ID != resp.ChallengeID || c.CodeHash == mailer.code {
				t.Fatalf("challenge = %+v, want the hashed code stored under the returned id", c)
			}
			if resp.EmailHint != "a****@example.com" {
				t.Errorf("email hint = %q", resp.EmailHint)
			}
		})
	}
}

func TestLoginChallengeService_IsTrustedNetwork(t *testing.T) {
	userID := uuid.New()
	repoErr := errors.New("db down")

	tests := []struct {
		name       string
		ctx        context.Context
		repo       *fakeUserRepo
		want       bool
		wantErr    error
		wantPrefix string
		wantCalls  int
	}{
		{name: "IPv4 address is checked by its /24 and trusted when a row exists", ctx: requestFrom("203.0.113.7"), repo: &fakeUserRepo{trustedFound: true}, want: true, wantPrefix: "203.0.113.0/24", wantCalls: 1},
		{name: "IPv4 address with no stored row is not trusted", ctx: requestFrom("203.0.113.7"), repo: &fakeUserRepo{}, wantPrefix: "203.0.113.0/24", wantCalls: 1},
		{name: "IPv6 address is checked by its /48", ctx: requestFrom("2001:db8:abcd:1234::1"), repo: &fakeUserRepo{trustedFound: true}, want: true, wantPrefix: "2001:db8:abcd::/48", wantCalls: 1},
		{name: "request without client info is not trusted and skips the repo", ctx: context.Background(), repo: &fakeUserRepo{trustedFound: true}},
		{name: "repo error is wrapped and reported as untrusted", ctx: requestFrom("203.0.113.7"), repo: &fakeUserRepo{trustedFound: true, trustedErr: repoErr}, wantErr: repoErr, wantPrefix: "203.0.113.0/24", wantCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewLoginChallengeService(tt.repo, nil, testCodeSecret, nopLogger())
			before := time.Now().UTC()

			got, err := svc.IsTrustedNetwork(tt.ctx, userID)
			after := time.Now().UTC()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("trusted = %v, want %v", got, tt.want)
			}
			if tt.repo.hasTrustedCalls != tt.wantCalls {
				t.Fatalf("repo calls = %d, want %d", tt.repo.hasTrustedCalls, tt.wantCalls)
			}
			if tt.wantCalls == 0 {
				return
			}
			if tt.repo.trustedUserID != userID || tt.repo.trustedPrefix != tt.wantPrefix {
				t.Errorf("repo got user %s prefix %q, want %s %q", tt.repo.trustedUserID, tt.repo.trustedPrefix, userID, tt.wantPrefix)
			}
			if since := tt.repo.trustedSince; since.Before(before.Add(-common.LOGIN_TRUSTED_NETWORK_TTL)) || since.After(after.Add(-common.LOGIN_TRUSTED_NETWORK_TTL)) {
				t.Errorf("since = %v, want now minus the trusted network TTL", since)
			}
		})
	}
}

func TestLoginChallengeService_RememberNetwork(t *testing.T) {
	userID := uuid.New()
	pruneErr := errors.New("prune failed")
	upsertErr := errors.New("upsert failed")

	tests := []struct {
		name        string
		ctx         context.Context
		repo        *fakeUserRepo
		wantErr     error
		wantPrefix  string
		wantPruned  bool
		wantUpserts int
		wantCommit  bool
	}{
		{name: "prunes expired rows then stores the masked IPv4 network", ctx: requestFrom("203.0.113.7"), repo: &fakeUserRepo{}, wantPrefix: "203.0.113.0/24", wantPruned: true, wantUpserts: 1, wantCommit: true},
		{name: "stores the /48 for an IPv6 address", ctx: requestFrom("2001:db8:abcd:1234::1"), repo: &fakeUserRepo{}, wantPrefix: "2001:db8:abcd::/48", wantPruned: true, wantUpserts: 1, wantCommit: true},
		{name: "request without client info writes nothing", ctx: context.Background(), repo: &fakeUserRepo{}},
		{name: "prune error is wrapped and nothing is stored", ctx: requestFrom("203.0.113.7"), repo: &fakeUserRepo{pruneTrustedErr: pruneErr}, wantErr: pruneErr, wantPruned: true},
		{name: "upsert error is wrapped and not committed", ctx: requestFrom("203.0.113.7"), repo: &fakeUserRepo{upsertTrustedErr: upsertErr}, wantErr: upsertErr, wantPrefix: "203.0.113.0/24", wantPruned: true, wantUpserts: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewLoginChallengeService(tt.repo, nil, testCodeSecret, nopLogger())
			before := time.Now().UTC()

			err := svc.RememberNetwork(tt.ctx, userID)
			after := time.Now().UTC()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if (tt.repo.prunedTrustedCalls > 0) != tt.wantPruned {
				t.Errorf("prune calls = %d, want pruned = %v", tt.repo.prunedTrustedCalls, tt.wantPruned)
			}
			if tt.repo.upsertTrustedCalls != tt.wantUpserts {
				t.Fatalf("upsert calls = %d, want %d", tt.repo.upsertTrustedCalls, tt.wantUpserts)
			}
			if committed := slices.Contains(tt.repo.calls, "Commit"); committed != tt.wantCommit {
				t.Errorf("committed = %v, want %v (calls %v)", committed, tt.wantCommit, tt.repo.calls)
			}
			if tt.wantPruned {
				if b := tt.repo.prunedTrustedBefore; b.Before(before.Add(-common.LOGIN_TRUSTED_NETWORK_TTL)) || b.After(after.Add(-common.LOGIN_TRUSTED_NETWORK_TTL)) {
					t.Errorf("prune before = %v, want now minus the trusted network TTL", b)
				}
			}
			if tt.wantUpserts == 0 {
				return
			}
			if tt.repo.trustedUserID != userID || tt.repo.upsertTrustedPrefix != tt.wantPrefix {
				t.Errorf("upsert got user %s prefix %q, want %s %q", tt.repo.trustedUserID, tt.repo.upsertTrustedPrefix, userID, tt.wantPrefix)
			}
			if v := tt.repo.upsertTrustedAt; v.Before(before) || v.After(after) {
				t.Errorf("verifiedAt = %v, want between %v and %v", v, before, after)
			}
		})
	}
}

func TestLoginChallengeService_Consume(t *testing.T) {
	svc := NewLoginChallengeService(nil, nil, testCodeSecret, nopLogger())
	id := uuid.New()
	userID := uuid.New()
	future := time.Now().UTC().Add(time.Minute)
	valid := users.LoginChallenge{ID: id, UserID: userID, CodeHash: svc.hashCode(id, "123456"), ExpiresAt: future,
		Location: &users.SessionLocation{City: "Amsterdam"}}
	with := func(edit func(c *users.LoginChallenge)) users.LoginChallenge {
		c := valid
		edit(&c)
		return c
	}
	now := time.Now().UTC()

	tests := []struct {
		name         string
		repo         *fakeUserRepo
		code         string
		wantErr      error
		wantAttempts int
	}{
		{name: "unknown challenge is invalid", repo: &fakeUserRepo{challengeErr: fmt.Errorf("x: %w", gorm.ErrRecordNotFound)}, code: "123456", wantErr: exceptions.ErrLoginCodeInvalid},
		{name: "expired challenge is invalid", repo: &fakeUserRepo{challenge: with(func(c *users.LoginChallenge) { c.ExpiresAt = now.Add(-time.Second) })}, code: "123456", wantErr: exceptions.ErrLoginCodeInvalid},
		{name: "used challenge is invalid", repo: &fakeUserRepo{challenge: with(func(c *users.LoginChallenge) { c.ConsumedAt = &now })}, code: "123456", wantErr: exceptions.ErrLoginCodeInvalid},
		{name: "challenge out of attempts is invalid even with the right code", repo: &fakeUserRepo{challenge: with(func(c *users.LoginChallenge) { c.Attempts = common.LOGIN_CODE_MAX_ATTEMPTS })}, code: "123456", wantErr: exceptions.ErrLoginCodeInvalid},
		{name: "wrong code records the attempt and commits it", repo: &fakeUserRepo{challenge: valid}, code: "654321", wantErr: exceptions.ErrLoginCodeInvalid, wantAttempts: 1},
		{name: "right code consumes the challenge", repo: &fakeUserRepo{challenge: valid}, code: "123456"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc.userRepo = tt.repo
			gotUser, loc, err := svc.Consume(context.Background(), id, tt.code)
			if tt.repo.challengeAttempts != tt.wantAttempts {
				t.Errorf("attempts = %d, want %d", tt.repo.challengeAttempts, tt.wantAttempts)
			}
			if tt.wantAttempts > 0 && tt.repo.calls[len(tt.repo.calls)-1] != "Commit" {
				t.Errorf("failed attempt was not committed: calls = %v", tt.repo.calls)
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotUser != userID || loc.City != "Amsterdam" || !tt.repo.challengeConsumed {
				t.Errorf("got user %s loc %+v consumed %v", gotUser, loc, tt.repo.challengeConsumed)
			}
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{name: "keeps the first letter and the domain", in: "shivam@gmail.com", want: "s****@gmail.com"},
		{name: "handles a multi-byte first letter", in: "étienne@example.fr", want: "é****@example.fr"},
		{name: "invalid address gives no hint", in: "nobody", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskEmail(tt.in); got != tt.want {
				t.Errorf("maskEmail(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func withPassword(t *testing.T, u users.User) users.User {
	t.Helper()
	u.PasswordHash = mustHash(t, loginTestPassword)
	return u
}
