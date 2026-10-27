//go:build integration

package integration

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/app/admin/service"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/core/bootstrap"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/infrastructure/cli"
	"github.com/xvq/go-template/internal/core/infrastructure/database"
	"github.com/xvq/go-template/internal/core/infrastructure/taskqueue"
	"github.com/xvq/go-template/internal/core/testutil"
	settingsservice "github.com/xvq/go-template/internal/service/settings"
	"gorm.io/gorm"
)

// Uses ONLY explicitly supplied test services. A fresh randomly named database
// is created and removed; existing application tables are never touched.
func environment(t *testing.T) (config.Config, *gorm.DB, *redis.Client, *slog.Logger) {
	t.Helper()
	addr, raddr := os.Getenv("GOTEMPLATE_TEST_MYSQL_ADDRESS"), os.Getenv("GOTEMPLATE_TEST_REDIS_ADDRESS")
	if addr == "" || raddr == "" {
		t.Skip("set GOTEMPLATE_TEST_MYSQL_ADDRESS and GOTEMPLATE_TEST_REDIS_ADDRESS for real integration tests")
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 8)
	if _, err = rand.Read(b); err != nil {
		t.Fatal(err)
	}
	name := "gotemplate_test_" + hex.EncodeToString(b)
	c := config.Defaults()
	c.Database.Host = host
	c.Database.Port = n
	c.Database.Username = os.Getenv("GOTEMPLATE_TEST_MYSQL_USER")
	if c.Database.Username == "" {
		c.Database.Username = "root"
	}
	c.Database.Password = os.Getenv("GOTEMPLATE_TEST_MYSQL_PASSWORD")
	admin, err := sql.Open("mysql", database.DSN(c.Database))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, e := admin.Exec("DROP DATABASE `" + name + "`"); e != nil {
			t.Errorf("remove test database: %v", e)
		}
		admin.Close()
	})
	c.Database.Database = name
	c.Redis.Address = raddr
	c.Redis.Password = os.Getenv("GOTEMPLATE_TEST_REDIS_PASSWORD")
	// Tests exclusively use a separate, initially empty Redis database.
	c.Redis.DB = 15
	if value := os.Getenv("GOTEMPLATE_TEST_REDIS_DB"); value != "" {
		c.Redis.DB, err = strconv.Atoi(value)
		if err != nil || c.Redis.DB < 1 {
			t.Fatal("GOTEMPLATE_TEST_REDIS_DB must be a positive database number")
		}
	}
	migration, err := cli.OpenMigrator(c.Database)
	if err != nil {
		t.Fatal(err)
	}
	if err = migration.Up(); err != nil {
		migration.Close()
		t.Fatal(err)
	}
	a, e := migration.Close()
	if a != nil || e != nil {
		t.Fatal(a, e)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	db, err := database.Open(c.Database, log)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool, _ := db.DB(); pool.Close() })
	r := redis.NewClient(&redis.Options{Addr: raddr, Password: c.Redis.Password, DB: c.Redis.DB})
	if err = r.Ping(context.Background()).Err(); err != nil {
		r.Close()
		t.Fatal(err)
	}
	count, err := r.DBSize(context.Background()).Result()
	if err != nil {
		r.Close()
		t.Fatal(err)
	}
	if count != 0 {
		r.Close()
		t.Fatalf("Redis test DB %d must be empty; existing keys will not be removed", c.Redis.DB)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		keys := r.Scan(ctx, 0, "*", 100).Iterator()
		for keys.Next(ctx) {
			if err := r.Del(ctx, keys.Val()).Err(); err != nil {
				t.Error(err)
			}
		}
		if err := keys.Err(); err != nil {
			t.Error(err)
		}
		r.Close()
	})
	testutil.UseCore(t, c, db, r, log)
	return c, db, r, log
}

func call(t *testing.T, app *fiber.App, method, path, token, body string, want int) map[string]any {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != want {
		t.Fatalf("%s %s got %d want %d: %v", method, path, resp.StatusCode, want, out)
	}
	return out
}
func TestAdminAPISettingsAndMigrations(t *testing.T) {
	c, _, redisClient, _ := environment(t)
	ctx := context.Background()
	admins, err := service.NewAdmin()
	if err != nil {
		t.Fatal(err)
	}
	first, err := admins.Create(ctx, "admin", "管理员", "secure-password", true)
	if err != nil {
		t.Fatal(err)
	}
	app, err := bootstrap.NewServer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	call(t, app, "GET", "/api/admin/admins", "", "", 401)
	call(t, app, "POST", "/api/admin/login", "", `{"username":"admin","password":"wrong"}`, 422)
	call(t, app, "POST", "/api/admin/login", "", `{"username":"admin","password":"wrong-password"}`, 401)
	login := call(t, app, "POST", "/api/admin/login", "", `{"username":"admin","password":"secure-password"}`, 200)
	token := login["data"].(map[string]any)["token"].(string)
	call(t, app, "GET", "/health/ready", "", "", 200)
	me := call(t, app, "GET", "/api/admin/me", token, "", 200)
	if _, ok := me["data"].(map[string]any)["password_hash"]; ok {
		t.Fatal("password hash leaked")
	}
	created := call(t, app, "POST", "/api/admin/admins", token, `{"username":"second","password":"secure-password","display_name":"第二个管理员"}`, 201)
	id := uint64(created["data"].(map[string]any)["id"].(float64))
	secondLogin := call(t, app, "POST", "/api/admin/login", "", `{"username":"second","password":"secure-password"}`, 200)
	secondToken := secondLogin["data"].(map[string]any)["token"].(string)
	listed := call(t, app, "GET", "/api/admin/admins?page=1&page_size=1", token, "", 200)["data"].(map[string]any)
	meta := listed["pagination"].(map[string]any)
	if meta["total"] != float64(2) || meta["per_page"] != float64(1) || meta["last_page"] != float64(2) || meta["current_page"] != float64(1) || meta["count"] != float64(1) || len(listed["data"].([]any)) != 1 {
		t.Fatalf("unexpected admin pagination: %v", listed)
	}
	empty := call(t, app, "GET", "/api/admin/admins?page=3&page_size=1", token, "", 200)["data"].(map[string]any)
	meta = empty["pagination"].(map[string]any)
	if meta["total"] != float64(2) || meta["last_page"] != float64(2) || meta["current_page"] != float64(3) || meta["count"] != float64(0) || len(empty["data"].([]any)) != 0 {
		t.Fatalf("out-of-range admin page lost total: %v", empty)
	}
	call(t, app, "POST", "/api/admin/admins", token, `{"username":"second","password":"secure-password"}`, 409)
	call(t, app, "PUT", fmt.Sprintf("/api/admin/admins/%d", id), token, `{"display_name":"第二个管理员","enabled":false}`, 200)
	call(t, app, "GET", "/api/admin/me", secondToken, "", 401)
	settingsList := call(t, app, "GET", "/api/admin/settings", token, "", 200)
	firstSetting := settingsList["data"].([]any)[0].(map[string]any)
	if firstSetting["default"] != "Go Template" || firstSetting["value"] != "Go Template" {
		t.Fatalf("missing typed defaults: %v", firstSetting)
	}
	if redisClient.Type(ctx, "settings").Val() != "hash" {
		t.Fatal("settings cache must be a hash")
	}
	if ttl, err := redisClient.Do(ctx, "TTL", "settings").Int64(); err != nil || ttl != -1 {
		t.Fatalf("cache is not permanent: ttl=%d err=%v", ttl, err)
	}
	call(t, app, "PATCH", "/api/admin/settings", token, `{"values":{"site.name":"Integration","site.maintenance":true}}`, 200)
	if name, err := settingsservice.String(ctx, "site.name"); err != nil || name != "Integration" {
		t.Fatal(name, err)
	}
	call(t, app, "PATCH", "/api/admin/settings", token, `{"values":{"site.maintenance":"true"}}`, 422)
	call(t, app, "GET", "/api/admin/login-logs", token, "", 200)
	call(t, app, "DELETE", fmt.Sprintf("/api/admin/admins/%d", id), token, "", 200)
	call(t, app, "DELETE", fmt.Sprintf("/api/admin/admins/%d", first.ID), token, "", 409)
	call(t, app, "PUT", "/api/admin/me/password", token, `{"current_password":"secure-password","password":"new-secure-password"}`, 200)
	call(t, app, "GET", "/api/admin/me", token, "", 401)
	t.Run("concurrent password changes", func(t *testing.T) {
		start := make(chan struct{})
		results := make(chan error, 2)
		for _, password := range []string{"concurrent-one", "concurrent-two"} {
			go func() {
				<-start
				results <- admins.ChangePassword(ctx, first.ID, "new-secure-password", password)
			}()
		}
		close(start)
		// Drain both workers before any fatal assertion or fixture cleanup.
		outcomes := []error{<-results, <-results}
		successes, rejected := 0, 0
		for _, err := range outcomes {
			if err == nil {
				successes++
			} else if errors.Is(err, apperror.ErrCredentials) {
				rejected++
			} else {
				t.Fatal(err)
			}
		}
		if successes != 1 || rejected != 1 {
			t.Fatalf("successes=%d rejected=%d", successes, rejected)
		}
	})
	migration, err := cli.OpenMigrator(c.Database)
	if err != nil {
		t.Fatal(err)
	}
	defer migration.Close()
	if err = migration.Steps(-1); err != nil {
		t.Fatal(err)
	}
	if err = migration.Up(); err != nil {
		t.Fatal(err)
	}
}

// These fixture models exist only in the isolated test DB. They demonstrate
// four-table nested eager loading without adding end-user business to template.
type customer struct {
	ID     uint
	Orders []order `gorm:"foreignKey:CustomerID"`
}
type order struct {
	ID         uint
	CustomerID uint
	Items      []item `gorm:"foreignKey:OrderID"`
}
type item struct {
	ID        uint
	OrderID   uint
	ProductID uint
	Product   product
}
type product struct {
	ID   uint
	Name string
}

func TestGORMNestedAssociations(t *testing.T) {
	_, db, _, _ := environment(t)
	if err := db.AutoMigrate(&customer{}, &order{}, &product{}, &item{}); err != nil {
		t.Fatal(err)
	}
	c := customer{Orders: []order{{Items: []item{{Product: product{Name: "demo"}}}}}}
	if err := db.Create(&c).Error; err != nil {
		t.Fatal(err)
	}
	var loaded customer
	if err := db.Preload("Orders.Items.Product").First(&loaded, c.ID).Error; err != nil {
		t.Fatal(err)
	}
	if len(loaded.Orders) != 1 || len(loaded.Orders[0].Items) != 1 || loaded.Orders[0].Items[0].Product.Name != "demo" {
		t.Fatalf("bad association graph: %+v", loaded)
	}
}

func TestRealQueueConsumption(t *testing.T) {
	c, _, _, log := environment(t)
	c.Queue.Enabled = true
	c.Queue.DefaultQueue = c.Database.Database
	c.Queue.Queues = map[string]int{c.Queue.DefaultQueue: 1}
	c.Queue.Concurrency = 1
	server := taskqueue.NewServer(c, log)
	mux := asynq.NewServeMux()
	done := make(chan struct{}, 1)
	mux.HandleFunc("integration:ping", func(ctx context.Context, task *asynq.Task) error {
		select {
		case done <- struct{}{}:
		default:
		}
		return nil
	})
	if err := server.Start(mux); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		server.Shutdown()
		inspector := asynq.NewInspector(taskqueue.RedisOptions(c))
		defer inspector.Close()
		if err := inspector.DeleteQueue(c.Queue.DefaultQueue, true); err != nil {
			t.Log("test queue cleanup:", err)
		}
	})
	client := taskqueue.NewClient(c)
	defer client.Close()
	if _, err := client.Enqueue(context.Background(), "integration:ping", map[string]int{"id": 1}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("worker did not consume task")
	}
}
