package bootstrap

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/xvq/go-template/internal/app/admin"
	"github.com/xvq/go-template/internal/app/web"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/infrastructure/validator"
	"github.com/xvq/go-template/internal/core/response"
	"github.com/xvq/go-template/internal/middleware"
)

// NewServer uses the globals initialized by Open.
func NewServer(ctx context.Context) (*fiber.App, error) {
	if err := validator.Init(); err != nil {
		return nil, err
	}
	c := core.Config
	errors := response.ErrorHandler()
	fiberConfig := fiber.Config{AppName: c.App.Name, Immutable: true, ReadTimeout: c.Server.ReadTimeout, WriteTimeout: c.Server.WriteTimeout, IdleTimeout: c.Server.IdleTimeout, BodyLimit: c.Server.BodyLimit, ErrorHandler: errors}
	if len(c.Server.TrustedProxies) > 0 {
		fiberConfig.TrustProxy = true
		fiberConfig.TrustProxyConfig = fiber.TrustProxyConfig{Proxies: c.Server.TrustedProxies}
		fiberConfig.ProxyHeader = "X-Forwarded-For"
	}
	app := fiber.New(fiberConfig)
	app.Use(middleware.RequestLifecycle(ctx, errors))
	app.Use(recover.New(recover.Config{PanicHandler: response.PanicHandler}))
	if len(c.Server.CORSOrigins) > 0 {
		app.Use(cors.New(cors.Config{AllowOrigins: c.Server.CORSOrigins, AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}, AllowHeaders: []string{"Authorization", "Content-Type"}, ExposeHeaders: []string{"X-Request-ID"}}))
	}
	app.Get("/health/live", func(c fiber.Ctx) error { return response.OK(c, fiber.Map{"status": "ok"}) })
	app.Get("/health/ready", func(c fiber.Ctx) error {
		check, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()
		pool, err := core.DB.DB()
		if err != nil {
			return fiber.NewError(503, "服务尚未就绪")
		}
		if pool.PingContext(check) != nil || core.Redis.Ping(check).Err() != nil {
			return fiber.NewError(503, "服务尚未就绪")
		}
		return response.OK(c, fiber.Map{"status": "ready"})
	})
	if err := admin.Register(app); err != nil {
		return nil, err
	}
	web.Register(app)
	return app, nil
}

func RunServer(ctx context.Context, c config.Config) error {
	r, err := Open(ctx, c, Options{
		Name:     "server",
		Database: true,
		Redis:    true,
		HTTP:     true,
		Queue:    c.Queue.Enabled,
	})
	if err != nil {
		return err
	}
	defer r.Close()
	// Keep in-flight contexts alive during graceful drain; cancel after shutdown.
	lifetime, cancel := context.WithCancel(context.Background())
	defer cancel()
	app, err := NewServer(lifetime)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(c.Server.Host, strconv.Itoa(c.Server.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	core.Log.Info("HTTP server started", "address", addr)
	return serve(ctx, app, ln, c.Server.ShutdownTimeout, cancel)
}

type readyListener struct {
	net.Listener
	ready chan struct{}
	once  sync.Once
}

func (l *readyListener) Accept() (net.Conn, error) {
	l.once.Do(func() { close(l.ready) })
	return l.Listener.Accept()
}

func serve(ctx context.Context, app *fiber.App, ln net.Listener, timeout time.Duration, cancelRequests context.CancelFunc) error {
	defer ln.Close()
	listener := &readyListener{Listener: ln, ready: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- app.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true}) }()
	// Wait until fasthttp has registered its listener before calling Shutdown.
	// This also handles a signal received immediately during startup.
	select {
	case <-listener.ready:
	case err := <-done:
		return err
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, stop := context.WithTimeout(context.Background(), timeout)
		defer stop()
		err := app.ShutdownWithContext(shutdown)
		cancelRequests()
		if err != nil {
			return fmt.Errorf("HTTP shutdown: %w", err)
		}
		return <-done
	}
}
