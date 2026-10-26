package command

import (
	"bufio"
	"context"
	"fmt"
	"github.com/spf13/cobra"
	adminrequest "github.com/xvq/go-template/internal/app/admin/request"
	"github.com/xvq/go-template/internal/app/admin/service"
	"github.com/xvq/go-template/internal/core/bootstrap"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/infrastructure/validator"
	"golang.org/x/term"
	"io"
	"os"
	"strings"
	"time"
)

func newAdminCreate(path *string) *cobra.Command {
	var username, name string
	var stdin bool
	cmd := &cobra.Command{Use: "admin:create", Short: "创建管理员，交互输入密码或使用 --password-stdin", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if err := validator.Init(); err != nil {
			return err
		}
		password, err := readPassword(cmd, stdin)
		if err != nil {
			return err
		}
		req := adminrequest.CreateAdmin{Username: username, DisplayName: name, Password: password}
		if err = validator.Struct(&req); err != nil {
			return err
		}
		c, err := config.Load(*path)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()
		r, err := bootstrap.Open(ctx, c, bootstrap.Options{Name: "admin:create", Database: true})
		if err != nil {
			return err
		}
		defer r.Close()
		admins, err := service.NewAdmin()
		if err != nil {
			return err
		}
		a, err := admins.Create(ctx, username, name, password, true)
		if err != nil {
			return err
		}
		cmd.Printf("管理员已创建: id=%d username=%s\n", a.ID, a.Username)
		return nil
	}}
	cmd.Flags().StringVar(&username, "username", "", "账号（必填）")
	cmd.MarkFlagRequired("username")
	cmd.Flags().StringVar(&name, "name", "", "显示名称")
	cmd.Flags().BoolVar(&stdin, "password-stdin", false, "从标准输入读取一行密码（适用于脚本）")
	return cmd
}
func readPassword(cmd *cobra.Command, stdin bool) (string, error) {
	if stdin {
		reader := bufio.NewReader(io.LimitReader(cmd.InOrStdin(), 1024))
		p, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		return strings.TrimSuffix(strings.TrimSuffix(p, "\n"), "\r"), nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("无交互终端，请使用 --password-stdin")
	}
	fmt.Fprint(cmd.ErrOrStderr(), "密码: ")
	p, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", err
	}
	fmt.Fprint(cmd.ErrOrStderr(), "确认密码: ")
	again, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", err
	}
	if string(p) != string(again) {
		return "", fmt.Errorf("两次密码不一致")
	}
	return string(p), nil
}
