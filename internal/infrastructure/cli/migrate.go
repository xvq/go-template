package cli

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	driver "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/spf13/cobra"
	"github.com/xvq/go-template/internal/config"
	"github.com/xvq/go-template/internal/infrastructure/database"
	"github.com/xvq/go-template/migrations"
)

func OpenMigrator(c config.Database) (*migrate.Migrate, error) {
	d, err := driver.ParseDSN(database.DSN(c))
	if err != nil {
		return nil, err
	}
	d.MultiStatements = true
	db, err := sql.Open("mysql", d.FormatDSN())
	if err != nil {
		return nil, err
	}
	backend, err := migratemysql.WithInstance(db, &migratemysql.Config{DatabaseName: c.Database})
	if err != nil {
		db.Close()
		return nil, err
	}
	source, err := iofs.New(migrations.Files, ".")
	if err != nil {
		backend.Close()
		return nil, err
	}
	m, err := migrate.NewWithInstance("iofs", source, "mysql", backend)
	if err != nil {
		source.Close()
		backend.Close()
		return nil, err
	}
	return m, nil
}
func newMigrate(path *string) *cobra.Command {
	root := &cobra.Command{Use: "migrate", Short: "显式执行 SQL 迁移"}
	run := func(action func(*migrate.Migrate, *cobra.Command, []string) error) func(*cobra.Command, []string) error {
		return func(cmd *cobra.Command, args []string) (result error) {
			c, err := config.Load(*path)
			if err != nil {
				return err
			}
			if err = c.Validate("database"); err != nil {
				return err
			}
			m, err := OpenMigrator(c.Database)
			if err != nil {
				return err
			}
			defer func() { a, b := m.Close(); result = errors.Join(result, a, b) }()
			finished := make(chan struct{})
			defer close(finished)
			go func() {
				select {
				case <-cmd.Context().Done():
					select {
					case m.GracefulStop <- true:
					case <-finished:
					}
				case <-finished:
				}
			}()
			err = action(m, cmd, args)
			if errors.Is(err, migrate.ErrNoChange) {
				cmd.Println("没有待执行的迁移")
				return nil
			}
			if err != nil {
				return err
			}
			return nil
		}
	}
	root.AddCommand(&cobra.Command{Use: "up", Short: "执行所有待应用迁移", Args: cobra.NoArgs, RunE: run(func(m *migrate.Migrate, cmd *cobra.Command, args []string) error {
		err := m.Up()
		if err == nil {
			cmd.Println("迁移完成")
		}
		return err
	})})
	var all, yes bool
	down := &cobra.Command{Use: "down", Short: "回滚一步；--all 回滚全部；必须显式 --yes", Args: cobra.NoArgs, PreRunE: func(cmd *cobra.Command, args []string) error {
		if !yes {
			return fmt.Errorf("回滚会删除或修改数据，请确认后加 --yes")
		}
		return nil
	}, RunE: run(func(m *migrate.Migrate, cmd *cobra.Command, args []string) error {
		if all {
			return m.Down()
		}
		return m.Steps(-1)
	})}
	down.Flags().BoolVar(&all, "all", false, "回滚全部迁移")
	down.Flags().BoolVar(&yes, "yes", false, "确认回滚")
	root.AddCommand(down)
	root.AddCommand(&cobra.Command{Use: "version", Short: "查看版本和 dirty 状态", Args: cobra.NoArgs, RunE: run(func(m *migrate.Migrate, cmd *cobra.Command, args []string) error {
		v, dirty, err := m.Version()
		if errors.Is(err, migrate.ErrNilVersion) {
			cmd.Println("尚未应用迁移")
			return nil
		}
		if err != nil {
			return err
		}
		cmd.Printf("version=%d dirty=%t\n", v, dirty)
		return nil
	})})
	force := &cobra.Command{Use: "force VERSION", Short: "手工修复表结构后修正迁移版本（不会执行 SQL）", Args: cobra.ExactArgs(1)}
	var forceYes bool
	force.Flags().BoolVar(&forceYes, "yes", false, "确认已手工检查并修复数据库")
	force.PreRunE = func(cmd *cobra.Command, args []string) error {
		if !forceYes {
			return fmt.Errorf("force 不修复表结构；确认数据库已修复后使用 --yes")
		}
		n, err := strconv.Atoi(args[0])
		if err != nil || n < -1 {
			return fmt.Errorf("版本号无效")
		}
		return nil
	}
	force.RunE = run(func(m *migrate.Migrate, cmd *cobra.Command, args []string) error {
		n, _ := strconv.Atoi(args[0])
		return m.Force(n)
	})
	root.AddCommand(force)
	return root
}
