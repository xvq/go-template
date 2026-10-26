package service

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

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
)

type passwordHashArgument struct{ password string }

func (a passwordHashArgument) Match(value driver.Value) bool {
	hash, ok := value.(string)
	if !ok || !strings.HasPrefix(hash, "$argon2id$") {
		return false
	}
	matched, err := argon2id.ComparePasswordAndHash(a.password, hash)
	if err != nil || !matched {
		return false
	}
	// A difference after byte 72 must still change the result.
	matched, err = argon2id.ComparePasswordAndHash(a.password[:len(a.password)-4]+"🔓", hash)
	return err == nil && !matched
}

func TestCreateAndResetStoreArgon2idForUnicodePasswords(t *testing.T) {
	db, mock := mockDB(t)
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	defer client.Close()
	testutil.UseCore(t, config.Defaults(), db, client, testLogger())
	admins, err := NewAdmin()
	if err != nil {
		t.Fatal(err)
	}
	password := strings.Repeat("🔐", 30)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO .*admins").WithArgs("admin", "管理员", passwordHashArgument{password}, true, sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	created, err := admins.Create(context.Background(), "admin", "管理员", password, true)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := core.Session.Issue(context.Background(), "admin", created.ID, time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*admins.*FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(created.ID))
	mock.ExpectExec("UPDATE .*admins").WithArgs(passwordHashArgument{password}, sqlmock.AnyArg(), created.ID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := admins.ResetPassword(context.Background(), created.ID, password); err != nil {
		t.Fatal(err)
	}
	if _, err := core.Session.Validate(context.Background(), token); !errors.Is(err, session.ErrInvalid) {
		t.Fatal("reset did not revoke session", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestChangePasswordChecksCurrentPasswordUnderLock(t *testing.T) {
	db, mock := mockDB(t)
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	defer client.Close()
	testutil.UseCore(t, config.Defaults(), db, client, testLogger())
	admins, err := NewAdmin()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	oldPassword := "old-password"
	newPassword := strings.Repeat("🔐", 30)
	oldHash, err := argon2id.CreateHash(oldPassword, &passwordhash.Params)
	if err != nil {
		t.Fatal(err)
	}
	newHash, err := argon2id.CreateHash(newPassword, &passwordhash.Params)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := core.Session.Issue(ctx, "admin", 1, time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*admins.*FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(1, oldHash))
	mock.ExpectExec("UPDATE .*admins").WithArgs(passwordHashArgument{newPassword}, sqlmock.AnyArg(), 1).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := admins.ChangePassword(ctx, 1, oldPassword, newPassword); err != nil {
		t.Fatal(err)
	}
	if _, err := core.Session.Validate(ctx, token); !errors.Is(err, session.ErrInvalid) {
		t.Fatal("change did not revoke session", err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*admins.*FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(1, newHash))
	mock.ExpectRollback()
	if err := admins.ChangePassword(ctx, 1, oldPassword, "another-password"); !errors.Is(err, apperror.ErrCredentials) {
		t.Fatal("stale password accepted", err)
	}
	redisServer.SetError("ERR unavailable")
	if err := admins.revokeAllSessions(ctx, 1); err == nil {
		t.Fatal("revocation failure was hidden")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
