package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/core"
	passwordhash "github.com/xvq/go-template/internal/core/infrastructure/password"
	"github.com/xvq/go-template/internal/core/infrastructure/session"
	"github.com/xvq/go-template/internal/model"
	"github.com/xvq/go-template/internal/util/query"
	"github.com/xvq/go-template/internal/util/str"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AdminService struct {
	dummyHash string
}

func NewAdmin() (*AdminService, error) {
	h, err := argon2id.CreateHash("not-a-real-account-password", &passwordhash.Params)
	if err != nil {
		return nil, err
	}
	return &AdminService{dummyHash: h}, nil
}

// LoginResult describes a business outcome, not the HTTP JSON contract.
type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	Admin     model.Admin
}

func (s *AdminService) Login(ctx context.Context, username, password, ip, ua string) (*LoginResult, error) {
	var a model.Admin
	err := core.DB.WithContext(ctx).Where("username = ?", username).First(&a).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	hash := s.dummyHash
	if err == nil {
		hash = a.PasswordHash
	}
	matched, passErr := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil || passErr != nil || !matched || !a.Enabled {
		s.audit(ctx, a.ID, username, false, "invalid_credentials", ip, ua)
		return nil, apperror.ErrCredentials
	}
	token, d, err := core.Session.Issue(ctx, "admin", a.ID, a.AuthVersion, core.Config.Auth.Admin.TokenTTL, core.Config.Auth.Admin.AllowMultipleSessions)
	if err != nil {
		s.audit(ctx, a.ID, username, false, "session_unavailable", ip, ua)
		return nil, err
	}
	s.audit(ctx, a.ID, username, true, "success", ip, ua)
	return &LoginResult{Token: token, ExpiresAt: time.Unix(d.ExpiresAt, 0).UTC(), Admin: a}, nil
}
func (s *AdminService) audit(ctx context.Context, id uint64, username string, success bool, reason, ip, ua string) {
	var adminID *uint64
	if id != 0 {
		adminID = &id
	}
	row := model.AdminLoginLog{AdminID: adminID, Username: str.Truncate(username, 64), Success: success, Reason: reason, IP: str.Truncate(ip, 64), UserAgent: str.Truncate(ua, 512)}
	if err := core.DB.WithContext(ctx).Create(&row).Error; err != nil {
		core.Log.Error("login audit write failed", "error_type", fmt.Sprintf("%T", err))
	}
}
func (s *AdminService) Authenticate(ctx context.Context, token string) (*model.Admin, error) {
	d, err := core.Session.Validate(ctx, token)
	if errors.Is(err, session.ErrInvalid) {
		return nil, apperror.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	if d.Realm != "admin" {
		return nil, apperror.ErrUnauthorized
	}
	var a model.Admin
	err = core.DB.WithContext(ctx).First(&a, d.SubjectID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	if !a.Enabled || a.AuthVersion != d.Version {
		return nil, apperror.ErrUnauthorized
	}
	return &a, nil
}
func (s *AdminService) Logout(ctx context.Context, token string) error {
	return core.Session.Revoke(ctx, token)
}
func (s *AdminService) Create(ctx context.Context, username, name, password string, enabled bool) (model.Admin, error) {
	h, err := argon2id.CreateHash(password, &passwordhash.Params)
	if err != nil {
		return model.Admin{}, err
	}
	a := model.Admin{Username: username, DisplayName: name, PasswordHash: h, Enabled: enabled, AuthVersion: 1}
	if err = core.DB.WithContext(ctx).Create(&a).Error; err != nil {
		return model.Admin{}, mapDBError(err)
	}
	return a, nil
}
func (s *AdminService) Get(ctx context.Context, id uint64) (model.Admin, error) {
	var a model.Admin
	err := core.DB.WithContext(ctx).First(&a, id).Error
	return a, mapDBError(err)
}
func (s *AdminService) List(ctx context.Context, page, size int) (query.Page[model.Admin], error) {
	return query.Paginate[model.Admin](
		core.DB.WithContext(ctx).Model(&model.Admin{}).Order("id DESC"), page, size,
	)
}

type AdminUpdate struct {
	DisplayName string
	Enabled     bool
}

func (s *AdminService) Update(ctx context.Context, id uint64, in AdminUpdate) (model.Admin, error) {
	var a model.Admin
	revoke := false
	err := core.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock enabled rows in stable order so concurrent disable/delete operations
		// cannot both remove the last active administrator.
		var active []model.Admin
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("enabled = ?", true).Order("id").Find(&active).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, id).Error; err != nil {
			return err
		}
		if !in.Enabled && a.Enabled && len(active) <= 1 {
			return apperror.ErrLastAdmin
		}
		changes := map[string]any{"display_name": in.DisplayName, "enabled": in.Enabled}
		if a.Enabled != in.Enabled {
			changes["auth_version"] = gorm.Expr("auth_version + 1")
			revoke = true
		}
		if err := tx.Model(&a).Updates(changes).Error; err != nil {
			return err
		}
		return tx.First(&a, id).Error
	})
	if err != nil {
		return model.Admin{}, mapDBError(err)
	}
	if revoke {
		s.revokeAll(ctx, id)
	}
	return a, nil
}
func (s *AdminService) Delete(ctx context.Context, id uint64) error {
	err := core.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var active []model.Admin
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("enabled = ?", true).Order("id").Find(&active).Error; err != nil {
			return err
		}
		var a model.Admin
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, id).Error; err != nil {
			return err
		}
		if a.Enabled && len(active) <= 1 {
			return apperror.ErrLastAdmin
		}
		return tx.Delete(&a).Error
	})
	if err != nil {
		return mapDBError(err)
	}
	s.revokeAll(ctx, id)
	return nil
}
func (s *AdminService) ChangePassword(ctx context.Context, id uint64, current, newPassword string) error {
	var a model.Admin
	if err := core.DB.WithContext(ctx).First(&a, id).Error; err != nil {
		return mapDBError(err)
	}
	matched, err := argon2id.ComparePasswordAndHash(current, a.PasswordHash)
	if err != nil || !matched {
		return apperror.ErrCredentials
	}
	return s.setPassword(ctx, id, newPassword, &a.AuthVersion)
}
func (s *AdminService) ResetPassword(ctx context.Context, id uint64, password string) error {
	return s.setPassword(ctx, id, password, nil)
}
func (s *AdminService) setPassword(ctx context.Context, id uint64, password string, expected *uint64) error {
	h, err := argon2id.CreateHash(password, &passwordhash.Params)
	if err != nil {
		return err
	}
	db := core.DB.WithContext(ctx).Model(&model.Admin{}).Where("id = ?", id)
	if expected != nil {
		db = db.Where("auth_version = ?", *expected)
	}
	r := db.Updates(map[string]any{"password_hash": h, "auth_version": gorm.Expr("auth_version + 1")})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		if expected != nil {
			return apperror.ErrUnauthorized
		}
		return apperror.ErrNotFound
	}
	s.revokeAll(ctx, id)
	return nil
}
func (s *AdminService) revokeAll(ctx context.Context, id uint64) {
	// AuthVersion/status are checked in MySQL on every request. Even if Redis
	// cleanup fails, old sessions cannot authenticate; TTL cleans up storage.
	if core.Session != nil {
		if err := core.Session.RevokeAll(ctx, "admin", id); err != nil {
			core.Log.Error("session cleanup failed; database already invalidated identity", "admin_id", id, "error", err)
		}
	}
}
func (s *AdminService) LoginLogs(ctx context.Context, page, size int) (query.Page[model.AdminLoginLog], error) {
	return query.Paginate[model.AdminLoginLog](
		core.DB.WithContext(ctx).Model(&model.AdminLoginLog{}).Order("id DESC"), page, size,
	)
}

func mapDBError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperror.ErrNotFound
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperror.ErrConflict
	}
	return err
}
