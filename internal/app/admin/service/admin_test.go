package service

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/config"
	"github.com/xvq/go-template/internal/infrastructure/session"
	"github.com/xvq/go-template/internal/testutil"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"log/slog"
	"testing"
	"time"
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
func TestLoginAndAccountVersionInvalidation(t *testing.T) {
	db, m := mockDB(t)
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	sessions := session.New(r, "test")
	c := config.Defaults()
	c.Redis.Prefix = "test"
	testutil.UseCore(t, c, db, r, testLogger())
	s, err := NewAdmin()
	if err != nil {
		t.Fatal(err)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("secure-password"), bcrypt.MinCost)
	m.ExpectQuery("SELECT .*admins").WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash", "enabled", "auth_version"}).AddRow(1, "admin", string(hash), true, 1))
	m.ExpectBegin()
	m.ExpectExec("INSERT INTO .*admin_login_logs").WillReturnResult(sqlmock.NewResult(1, 1))
	m.ExpectCommit()
	result, err := s.Login(context.Background(), "admin", "secure-password", "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if result.Token == "" || result.Admin.ID != 1 {
		t.Fatal("invalid login result")
	}
	m.ExpectQuery("SELECT .*admins").WillReturnRows(sqlmock.NewRows([]string{"id", "enabled", "auth_version"}).AddRow(1, true, 2))
	if _, err = s.Authenticate(context.Background(), result.Token); !errors.Is(err, apperror.ErrUnauthorized) {
		t.Fatalf("stale password version authenticated: %v", err)
	}
	m.ExpectQuery("SELECT .*admins").WillReturnRows(sqlmock.NewRows([]string{"id", "enabled", "auth_version"}).AddRow(1, false, 1))
	if _, err = s.Authenticate(context.Background(), result.Token); !errors.Is(err, apperror.ErrUnauthorized) {
		t.Fatalf("disabled account authenticated: %v", err)
	}
	web, _, err := sessions.Issue(context.Background(), "web", 1, 1, time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Authenticate(context.Background(), web); !errors.Is(err, apperror.ErrUnauthorized) {
		t.Fatal("web token accepted as admin")
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
