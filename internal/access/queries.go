package access

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	if err := r.db.WithContext(ctx).Transaction(fn); err != nil {
		return fmt.Errorf("access transaction: %w", err)
	}
	return nil
}

// A missing user is not an admin, so a token for a deleted user gets no access.
func (r *Repository) IsSuperAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	var flags []bool
	err := r.db.WithContext(ctx).Table("users").Where("id = ?", userID).Pluck("is_admin", &flags).Error
	if err != nil {
		return false, fmt.Errorf("read admin flag for %s: %w", userID, err)
	}
	return len(flags) > 0 && flags[0], nil
}

func (r *Repository) HasGrant(ctx context.Context, userID uuid.UUID, perm common.Permission, resources []common.ResourceRef) (bool, error) {
	if len(resources) == 0 {
		return false, nil
	}
	pairs := make([][]any, 0, len(resources))
	for _, res := range resources {
		pairs = append(pairs, []any{string(res.Type), res.ID})
	}
	var found bool
	err := r.db.WithContext(ctx).Raw(
		`SELECT EXISTS (SELECT 1 FROM user_permissions
		 WHERE user_id = ? AND permission = ? AND (resource_type, resource_id) IN ?)`,
		userID, string(perm), pairs,
	).Scan(&found).Error
	if err != nil {
		return false, fmt.Errorf("check %s grant for %s: %w", perm, userID, err)
	}
	return found, nil
}

// class_courses belongs to the classes domain; this is the one column access
// needs from it, so a channel grant can cover the channel's courses.
func (r *Repository) FindCourseChannelID(ctx context.Context, courseID uuid.UUID) (string, bool, error) {
	var ids []string
	err := r.db.WithContext(ctx).Raw(`SELECT channel_id::text FROM class_courses WHERE id = ?`, courseID).Scan(&ids).Error
	if err != nil {
		return "", false, fmt.Errorf("find channel of course %s: %w", courseID, err)
	}
	if len(ids) == 0 {
		return "", false, nil
	}
	return ids[0], true, nil
}

func (r *Repository) ListUserGrants(ctx context.Context, userID uuid.UUID) ([]UserPermission, error) {
	var rows []UserPermission
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("permission, resource_type, resource_id").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list grants for %s: %w", userID, err)
	}
	return rows, nil
}

// email "" lists every grant.
func (r *Repository) ListPermissions(ctx context.Context, email string) ([]PermissionView, error) {
	q := r.db.WithContext(ctx).Table("user_permissions AS p").
		Select("p.*, u.email AS user_email, u.username AS username").
		Joins("JOIN users u ON u.id = p.user_id")
	if email != "" {
		q = q.Where("u.email = ?", email)
	}
	var rows []PermissionView
	if err := q.Order("u.email, p.permission, p.resource_type, p.resource_id").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	return rows, nil
}

// Granting something the user already has is not an error; the stored row is
// returned either way.
func (r *Repository) CreatePermission(ctx context.Context, p *UserPermission) (UserPermission, error) {
	db := r.db.WithContext(ctx)
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(p).Error; err != nil {
		return UserPermission{}, fmt.Errorf("insert permission: %w", err)
	}
	var stored UserPermission
	err := db.Where("user_id = ? AND permission = ? AND resource_type = ? AND resource_id = ?",
		p.UserID, p.Permission, p.ResourceType, p.ResourceID).First(&stored).Error
	if err != nil {
		return UserPermission{}, fmt.Errorf("read permission: %w", err)
	}
	return stored, nil
}

func (r *Repository) GetPermission(ctx context.Context, id uuid.UUID) (UserPermission, error) {
	var p UserPermission
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserPermission{}, exceptions.WithCause(exceptions.ErrPermissionNotFound, err)
		}
		return UserPermission{}, fmt.Errorf("get permission %s: %w", id, err)
	}
	return p, nil
}

func (r *Repository) DeletePermission(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&UserPermission{}).Error; err != nil {
		return fmt.Errorf("delete permission %s: %w", id, err)
	}
	return nil
}

// Already-held grants are skipped, so accepting overlapping invites works.
func (r *Repository) CreatePermissions(ctx context.Context, tx *gorm.DB, perms []UserPermission) error {
	if len(perms) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&perms).Error; err != nil {
		return fmt.Errorf("insert permissions: %w", err)
	}
	return nil
}

func (r *Repository) ResourceName(ctx context.Context, res common.ResourceRef) (*string, error) {
	var query string
	switch res.Type {
	case common.RESOURCE_TYPE_CHANNEL:
		query = `SELECT COALESCE(name_custom, name_youtube) FROM class_channels WHERE id = ?`
	case common.RESOURCE_TYPE_COURSE:
		query = `SELECT title FROM class_courses WHERE id = ?`
	default:
		return nil, nil
	}
	id, err := uuid.Parse(res.ID)
	if err != nil {
		return nil, nil //nolint:nilerr // a malformed id names no resource
	}
	var names []*string
	if err := r.db.WithContext(ctx).Raw(query, id).Scan(&names).Error; err != nil {
		return nil, fmt.Errorf("find name of %s %s: %w", res.Type, res.ID, err)
	}
	if len(names) == 0 {
		return nil, nil
	}
	return names[0], nil
}

func (r *Repository) CreateInvite(ctx context.Context, tx *gorm.DB, inv *StaffInvite) error {
	if err := tx.WithContext(ctx).Create(inv).Error; err != nil {
		return fmt.Errorf("insert invite: %w", err)
	}
	return nil
}

func (r *Repository) ListInvites(ctx context.Context) ([]StaffInvite, error) {
	var rows []StaffInvite
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	return rows, nil
}

func (r *Repository) LockInvite(ctx context.Context, tx *gorm.DB, id uuid.UUID) (StaffInvite, error) {
	return lockInviteWhere(ctx, tx, "id = ?", id)
}

func (r *Repository) LockInviteByHash(ctx context.Context, tx *gorm.DB, tokenHash []byte) (StaffInvite, error) {
	return lockInviteWhere(ctx, tx, "token_hash = ?", tokenHash)
}

func (r *Repository) FindInviteByHash(ctx context.Context, tokenHash []byte) (StaffInvite, error) {
	var inv StaffInvite
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&inv).Error; err != nil {
		return StaffInvite{}, mapInviteErr(err)
	}
	return inv, nil
}

func (r *Repository) ReplaceInviteToken(ctx context.Context, tx *gorm.DB, id uuid.UUID, tokenHash []byte, expiresAt time.Time) error {
	err := tx.WithContext(ctx).Model(&StaffInvite{}).Where("id = ?", id).
		Updates(map[string]any{"token_hash": tokenHash, "expires_at": expiresAt}).Error
	if err != nil {
		return fmt.Errorf("replace token of invite %s: %w", id, err)
	}
	return nil
}

func (r *Repository) RevokeInvite(ctx context.Context, tx *gorm.DB, id uuid.UUID, now time.Time) error {
	if err := tx.WithContext(ctx).Model(&StaffInvite{}).Where("id = ?", id).Update("revoked_at", now).Error; err != nil {
		return fmt.Errorf("revoke invite %s: %w", id, err)
	}
	return nil
}

func (r *Repository) MarkInviteAccepted(ctx context.Context, tx *gorm.DB, id, userID uuid.UUID, now time.Time) error {
	err := tx.WithContext(ctx).Model(&StaffInvite{}).Where("id = ?", id).
		Updates(map[string]any{"accepted_at": now, "accepted_by": userID}).Error
	if err != nil {
		return fmt.Errorf("mark invite %s accepted: %w", id, err)
	}
	return nil
}

// The row lock makes two accepts (or an accept racing a revoke) of the same
// invite run one after the other, so it is used once.
func lockInviteWhere(ctx context.Context, tx *gorm.DB, cond string, arg any) (StaffInvite, error) {
	var inv StaffInvite
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where(cond, arg).First(&inv).Error
	if err != nil {
		return StaffInvite{}, mapInviteErr(err)
	}
	return inv, nil
}

func mapInviteErr(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return exceptions.WithCause(exceptions.ErrInviteNotFound, err)
	}
	return fmt.Errorf("find invite: %w", err)
}
