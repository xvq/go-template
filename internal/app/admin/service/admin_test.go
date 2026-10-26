package service

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alexedwards/argon2id"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/core/config"
	passwordhash "github.com/xvq/go-template/internal/core/infrastructure/password"
	"github.com/xvq/go-template/internal/core/infrastructure/session"
	"github.com/xvq/go-template/internal/core/testutil"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func mockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	conn, m, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return db, m
}
func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
func TestLoginAndLogout(t *testing.T) {
	db, m := mockDB(t)
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	c := config.Defaults()
	testutil.UseCore(t, c, db, r, testLogger())
	s, err := NewAdmin()
	if err != nil {
		t.Fatal(err)
	}
	password := strings.Repeat("中", 30) // 90 bytes: must not be truncated or rejected.
	hash, err := argon2id.CreateHash(password, &passwordhash.Params)
	if err != nil {
		t.Fatal(err)
	}
	m.ExpectQuery("SELECT .*admins").WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash", "enabled"}).AddRow(1, "admin", hash, true))
	m.ExpectBegin()
	m.ExpectExec("INSERT INTO .*admin_login_logs").WillReturnResult(sqlmock.NewResult(1, 1))
	m.ExpectCommit()
	result, err := s.Login(context.Background(), "admin", password, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if result.Token == "" || result.Admin.ID != 1 {
		t.Fatal("invalid login result")
	}
	if _, err := core.Session.Validate(context.Background(), result.Token); err != nil {
		t.Fatal(err)
	}
	if err := s.Logout(context.Background(), result.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := core.Session.Validate(context.Background(), result.Token); !errors.Is(err, session.ErrInvalid) {
		t.Fatal("logout did not revoke token", err)
	}
	if err = m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestMissingAccountAuditAndDatabaseFailure(t *testing.T) {
	db, m := mockDB(t)
	testutil.UseCore(t, config.Defaults(), db, nil, testLogger())
	s, err := NewAdmin()
	if err != nil {
		t.Fatal(err)
	}
	m.ExpectQuery("SELECT .*admins").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	m.ExpectBegin()
	m.ExpectExec("INSERT INTO .*admin_login_logs").WillReturnResult(sqlmock.NewResult(1, 1))
	m.ExpectCommit()
	if _, err = s.Login(context.Background(), "missing", "wrong", "127.0.0.1", ""); !errors.Is(err, apperror.ErrCredentials) {
		t.Fatal(err)
	}
	m.ExpectQuery("SELECT count").WillReturnError(errors.New("database offline"))
	if _, err = s.List(context.Background(), 1, 20); err == nil {
		t.Fatal("database error swallowed")
	}
	if err = m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestLastAdminDeleteRollsBack(t *testing.T) {
	db, m := mockDB(t)
	testutil.UseCore(t, config.Defaults(), db, nil, testLogger())
	s, err := NewAdmin()
	if err != nil {
		t.Fatal(err)
	}
	m.ExpectBegin()
	m.ExpectQuery("SELECT .*admins.*FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"id", "enabled"}).AddRow(1, true))
	m.ExpectQuery("SELECT .*admins.*FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"id", "enabled"}).AddRow(1, true))
	m.ExpectRollback()
	if err = s.Delete(context.Background(), 1); !errors.Is(err, apperror.ErrLastAdmin) {
		t.Fatal(err)
	}
	if err = m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
