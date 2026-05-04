package command

import (
	"fmt"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	appEmbed "github.com/user/go-template"
	"github.com/user/go-template/internal/config"
	"github.com/user/go-template/internal/support"
)

func init() {
	support.RegisterCommand(&support.Command{
		Name: "migrate",
		Desc: "Run database migrations",
		Actions: []*support.Action{
			{Name: "up", Desc: "Apply all up migrations", Run: runUp},
			{Name: "down", Desc: "Rollback all down migrations", Run: runDown},
			{Name: "steps", Desc: "Migrate N steps (+up, -down)", Run: runSteps},
			{Name: "version", Desc: "Show current migration version", Run: runVersion},
		},
	})
}

func newMigrate() (*migrate.Migrate, error) {
	cfg := config.AppConfig
	conn := cfg.DefaultDB()
	if conn == nil {
		return nil, fmt.Errorf("no default database configured")
	}

	source, err := iofs.New(appEmbed.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("migrate source init failed: %w", err)
	}

	return migrate.NewWithSourceInstance("iofs", source, buildDSN(conn))
}

func runUp(args []string) error {
	m, err := newMigrate()
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up failed: %w", err)
	}
	fmt.Println("migrate up done")
	return nil
}

func runDown(args []string) error {
	if !confirm() {
		return nil
	}
	m, err := newMigrate()
	if err != nil {
		return err
	}
	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate down failed: %w", err)
	}
	fmt.Println("migrate down done")
	return nil
}

func runSteps(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: migrate steps <N>")
	}
	n, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid step count: %s", args[0])
	}
	if n < 0 && !confirm() {
		return nil
	}
	m, err := newMigrate()
	if err != nil {
		return err
	}
	if err := m.Steps(n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate steps failed: %w", err)
	}
	fmt.Println("migrate steps done")
	return nil
}

func runVersion(args []string) error {
	m, err := newMigrate()
	if err != nil {
		return err
	}
	ver, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("migrate version failed: %w", err)
	}
	fmt.Printf("version: %d, dirty: %v\n", ver, dirty)
	return nil
}

func confirm() bool {
	fmt.Print("Confirm? (yes/no): ")
	var input string
	fmt.Scanln(&input)
	return input == "yes"
}

func buildDSN(cfg *config.DBConfig) string {
	switch cfg.Driver {
	case "mysql":
		return fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	case "postgres":
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	case "sqlite":
		return fmt.Sprintf("sqlite3://%s", cfg.File)
	default:
		return ""
	}
}
