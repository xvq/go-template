package query

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Search adds a grouped OR substring search across code-selected text columns.
// Blank keywords or no columns add no condition. %, _ and ! are literal input;
// ! is the explicit LIKE escape character, independent of MySQL backslash mode.
func Search(keyword string, columns ...string) func(*gorm.DB) *gorm.DB {
	keyword = strings.TrimSpace(keyword)
	pattern := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(keyword) + "%"
	return func(db *gorm.DB) *gorm.DB {
		if keyword == "" || len(columns) == 0 {
			return db
		}
		conditions := make([]clause.Expression, 0, len(columns))
		for _, column := range columns {
			conditions = append(conditions, clause.Expr{
				SQL:  "? LIKE ? ESCAPE '!'",
				Vars: []any{filterColumn(column), pattern},
			})
		}
		return db.Where(clause.Or(conditions...))
	}
}

// Equal adds column = value unless value is nil. Zero values are retained.
func Equal[T any](column string, value *T) func(*gorm.DB) *gorm.DB {
	return optional(value, func(v T) clause.Expression {
		return clause.Eq{Column: filterColumn(column), Value: v}
	})
}

// In adds a membership condition. An empty slice skips this optional filter;
// it must not be used to enforce an authorization allowlist that can be empty.
func In[T any](column string, values []T) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(values) == 0 {
			return db
		}
		args := make([]any, len(values))
		for i, value := range values {
			args[i] = value
		}
		return db.Where(clause.IN{Column: filterColumn(column), Values: args})
	}
}

// GreaterThan adds column > value unless value is nil.
func GreaterThan[T any](column string, value *T) func(*gorm.DB) *gorm.DB {
	return optional(value, func(v T) clause.Expression {
		return clause.Gt{Column: filterColumn(column), Value: v}
	})
}

// GreaterOrEqual adds column >= value unless value is nil.
func GreaterOrEqual[T any](column string, value *T) func(*gorm.DB) *gorm.DB {
	return optional(value, func(v T) clause.Expression {
		return clause.Gte{Column: filterColumn(column), Value: v}
	})
}

// LessThan adds column < value unless value is nil.
func LessThan[T any](column string, value *T) func(*gorm.DB) *gorm.DB {
	return optional(value, func(v T) clause.Expression {
		return clause.Lt{Column: filterColumn(column), Value: v}
	})
}

// LessOrEqual adds column <= value unless value is nil.
func LessOrEqual[T any](column string, value *T) func(*gorm.DB) *gorm.DB {
	return optional(value, func(v T) clause.Expression {
		return clause.Lte{Column: filterColumn(column), Value: v}
	})
}

func optional[T any](value *T, condition func(T) clause.Expression) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if value == nil {
			return db
		}
		return db.Where(condition(*value))
	}
}

// Columns are identifiers, not SQL expressions; support table.column and aliases.
func filterColumn(name string) clause.Column {
	if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
		return clause.Column{Table: name[:dot], Name: name[dot+1:]}
	}
	return clause.Column{Name: name}
}
