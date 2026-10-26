package service

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alexedwards/argon2id"
	"github.com/xvq/go-template/internal/core/config"
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
	testutil.UseCore(t, config.Defaults(), db, nil, testLogger())
	admins, err := NewAdmin()
	if err != nil {
		t.Fatal(err)
	}
	password := strings.Repeat("🔐", 30)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO .*admins").WithArgs("admin", "管理员", passwordHashArgument{password}, true, 1, sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	created, err := admins.Create(context.Background(), "admin", "管理员", password, true)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*admins").WithArgs(passwordHashArgument{password}, sqlmock.AnyArg(), created.ID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := admins.ResetPassword(context.Background(), created.ID, password); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
