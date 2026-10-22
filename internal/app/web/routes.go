package web

import "github.com/gofiber/fiber/v3"

// Register is the extension point for project-specific website APIs.
// No end-user schema, login flow, or business endpoints are included.
func Register(app *fiber.App) { app.Group("/api/web") }
