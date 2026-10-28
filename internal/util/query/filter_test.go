package query

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"
)

func filterSQL(t *testing.T, scopes ...func(*gorm.DB) *gorm.DB) (string, []any) {
	t.Helper()
	db, _ := mockDB(t)
	result := db.Session(&gorm.Session{DryRun: true}).Model(&pageRecord{}).
		Scopes(scopes...).Find(&[]pageRecord{})
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	return result.Statement.SQL.String(), result.Statement.Vars
}

func TestOptionalFiltersSkipMissingValues(t *testing.T) {
	sql, args := filterSQL(t,
		Search(" \t\n", "username"), Search("hello"),
		Equal[int]("id", nil), In("id", []int{}), In[int]("id", nil),
		GreaterThan[int]("id", nil), GreaterOrEqual[int]("id", nil),
		LessThan[int]("id", nil), LessOrEqual[int]("id", nil),
	)
	if sql != "SELECT * FROM `page_records`" || len(args) != 0 {
		t.Fatalf("missing values added filters: %s %v", sql, args)
	}
}

func TestEqualPreservesZeroValuesAndEnum(t *testing.T) {
	type status uint8
	for _, tc := range []struct {
		name  string
		scope func(*gorm.DB) *gorm.DB
		value any
	}{
		{"zero", Equal("status", new(0)), 0},
		{"false", Equal("active", new(false)), false},
		{"empty string", Equal("username", new("")), ""},
		{"enum", Equal("status", new(status(2))), status(2)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, args := filterSQL(t, tc.scope)
			if !reflect.DeepEqual(args, []any{tc.value}) {
				t.Fatalf("value lost: %#v", args)
			}
		})
	}
}

func TestSearchGroupsORAndBindsLiteralPattern(t *testing.T) {
	keyword := `  50%_off!\' OR 1=1 --  `
	sql, args := filterSQL(t,
		Equal("tenant_id", new(7)),
		Search(keyword, "users.username", "users.display_name"),
		Equal("enabled", new(false)),
	)
	want := "SELECT * FROM `page_records` WHERE `tenant_id` = ? AND (`users`.`username` LIKE ? ESCAPE '!' OR `users`.`display_name` LIKE ? ESCAPE '!') AND `enabled` = ?"
	pattern := `%50!%!_off!!\' OR 1=1 --%`
	if sql != want || !reflect.DeepEqual(args, []any{7, pattern, pattern, false}) {
		t.Fatalf("search grouping/escaping: %s %#v", sql, args)
	}
	if strings.Contains(sql, "1=1") {
		t.Fatal("keyword was interpolated into SQL")
	}
}

func TestComparisonFiltersAndMembership(t *testing.T) {
	start := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	sql, args := filterSQL(t,
		GreaterThan("p.id", new(0)),
		GreaterOrEqual("p.created_at", &start),
		LessThan("p.created_at", &end),
		LessOrEqual("p.id", new(100)),
		In("p.status", []int{0, 2}),
	)
	want := "SELECT * FROM `page_records` WHERE `p`.`id` > ? AND `p`.`created_at` >= ? AND `p`.`created_at` < ? AND `p`.`id` <= ? AND `p`.`status` IN (?,?)"
	if sql != want || !reflect.DeepEqual(args, []any{0, start, end, 100, 0, 2}) {
		t.Fatalf("comparison filters: %s %#v", sql, args)
	}
}

func TestFilterColumnsAreQuotedIdentifiers(t *testing.T) {
	column := "name` OR 1=1 --"
	sql, args := filterSQL(t, Equal(column, new("value")))
	want := "SELECT * FROM `page_records` WHERE `name`` OR 1=1 --` = ?"
	if sql != want || !reflect.DeepEqual(args, []any{"value"}) {
		t.Fatalf("column treated as an expression: %s %#v", sql, args)
	}
}

func TestFiltersComposeWithPaginationAndCanBeReused(t *testing.T) {
	db, mock := mockDB(t)
	search := Search(" hello ", "username", "display_name")
	active := Equal("active", new(false))
	base := db.Model(&pageRecord{}).Scopes(search, active).Order("id DESC")
	where := " WHERE (`username` LIKE ? ESCAPE '!' OR `display_name` LIKE ? ESCAPE '!') AND `active` = ?"
	for range 2 {
		mock.ExpectQuery("SELECT count(*) FROM `page_records`"+where).
			WithArgs("%hello%", "%hello%", false).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT * FROM `page_records`"+where+" ORDER BY id DESC LIMIT ?").
			WithArgs("%hello%", "%hello%", false, 20).
			WillReturnRows(sqlmock.NewRows([]string{"id", "active"}).AddRow(1, false))
		page, err := Paginate[pageRecord](base, 1, 20)
		if err != nil || len(page.Data) != 1 || page.Pagination.Total != 1 {
			t.Fatalf("filters lost during pagination/reuse: %+v %v", page, err)
		}
	}
}
