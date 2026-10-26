package query

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/xvq/go-template/internal/apperror"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type pageRecord struct {
	ID       uint64
	Active   bool
	Children []pageChild `gorm:"foreignKey:RecordID"`
}

func (pageRecord) TableName() string { return "page_records" }

type pageChild struct {
	ID       uint64
	RecordID uint64
}

func (pageChild) TableName() string { return "page_children" }

func mockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	conn, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		conn.Close()
	})
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return db, mock
}

func TestPaginatePreservesQueryAndPreloads(t *testing.T) {
	db, mock := mockDB(t)
	base := db.WithContext(context.Background()).Model(&pageRecord{}).
		Where("active = ?", true).
		Scopes(func(db *gorm.DB) *gorm.DB { return db.Where("id > ?", 1) }).
		Order("id DESC").Preload("Children").Limit(99).Offset(88)
	countSQL := "SELECT count(*) FROM `page_records` WHERE active = ? AND id > ?"
	pageSQL := "SELECT * FROM `page_records` WHERE active = ? AND id > ? ORDER BY id DESC LIMIT ? OFFSET ?"
	mock.ExpectQuery(countSQL).WithArgs(true, 1).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(pageSQL).WithArgs(true, 1, 2, 2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "active"}).AddRow(7, true))
	mock.ExpectQuery("SELECT * FROM `page_children` WHERE `page_children`.`record_id` = ?").WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"id", "record_id"}).AddRow(9, 7))
	page, err := Paginate[pageRecord](base, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if page.Pagination.Total != 3 || page.Pagination.CurrentPage != 2 || page.Pagination.PerPage != 2 || page.Pagination.LastPage != 2 || page.Pagination.Count != 1 || len(page.Data) != 1 || page.Data[0].ID != 7 || len(page.Data[0].Children) != 1 {
		t.Fatalf("unexpected page: %+v", page)
	}
	limit := base.Statement.Clauses["LIMIT"].Expression.(clause.Limit)
	if *limit.Limit != 99 || limit.Offset != 88 || len(base.Statement.Selects) != 0 {
		t.Fatal("pagination mutated the original query")
	}
	// Reuse the original query; scopes and ordering must survive both operations.
	mock.ExpectQuery(countSQL).WithArgs(true, 1).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(pageSQL).WithArgs(true, 1, 2, 4).
		WillReturnRows(sqlmock.NewRows([]string{"id", "active"}))
	page, err = Paginate[pageRecord](base, 3, 2)
	if err != nil || page.Pagination.Total != 3 || page.Pagination.CurrentPage != 3 || page.Pagination.LastPage != 2 || page.Pagination.Count != 0 || page.Data == nil || len(page.Data) != 0 {
		t.Fatalf("out-of-range page lost total or empty array: %+v, %v", page, err)
	}
}

func TestPaginatePropagatesErrors(t *testing.T) {
	failure := errors.New("database unavailable")
	for _, stage := range []string{"count", "find"} {
		t.Run(stage, func(t *testing.T) {
			db, mock := mockDB(t)
			count := mock.ExpectQuery("SELECT count(*) FROM `page_records`")
			if stage == "count" {
				count.WillReturnError(failure)
			} else {
				count.WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
				mock.ExpectQuery("SELECT * FROM `page_records` ORDER BY id DESC LIMIT ?").WithArgs(20).WillReturnError(failure)
			}
			_, err := Paginate[pageRecord](db.Model(&pageRecord{}).Order("id DESC"), 1, 20)
			if !errors.Is(err, failure) {
				t.Fatalf("%s error was lost: %v", stage, err)
			}
		})
	}
}

func TestPaginateRejectsInvalidParametersBeforeQuery(t *testing.T) {
	db, _ := mockDB(t)
	for _, pair := range [][2]int{{0, 20}, {-1, 20}, {1000001, 20}, {1, 0}, {1, -1}, {1, 101}} {
		_, err := Paginate[pageRecord](db.Model(&pageRecord{}), pair[0], pair[1])
		if !errors.Is(err, apperror.ErrInvalidPagination) {
			t.Fatalf("expected pagination error for %v, got %v", pair, err)
		}
	}
}

func TestPaginateHonorsCanceledContext(t *testing.T) {
	db, _ := mockDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Paginate[pageRecord](db.WithContext(ctx).Model(&pageRecord{}), 1, 20)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation was lost: %v", err)
	}
}
