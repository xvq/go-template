# Repository conventions

- Read README.md and docs/development.md before changing architecture.
- This is a pure API template: Fiber v3, GORM/MySQL, Redis sessions, ozzo-validation, optional Asynq. Do not add end-user business, a frontend, RBAC, or another framework without a request.
- Keep application code in internal/app/<app>/{handler,validator,service,resource,middleware}; register routes explicitly in routes.go. Handler order matters: middleware first, final handler last.
- Application services own business rules and return models/business results; resources own JSON field selection and never query the database. Keep reusable capabilities in internal/service/<capability>, separate from application business services. Use resource.Page[T] for service pagination and resource helpers for output conversion.
- Resources transform database results. Compose responses such as login directly in handlers. Define only per-resource item mappings; call shared Collection/Paginate helpers directly instead of adding per-resource wrappers.
- Inject dependencies in bootstrap. Do not introduce global mutable DB/Redis/config singletons or implicit init registration.
- Use real context.Context from c.Context() in services; never retain pooled fiber.Ctx. Streaming has a separate bounded lifecycle; see response.Stream.
- Keep original GORM query capabilities. No generic repository layer just to wrap CRUD. Use explicit updates for zero values. Always handle query errors.
- No AutoMigrate in application startup. Add versioned SQL migrations, do not edit released migrations. AutoMigrate is allowed only for isolated integration-test fixtures.
- Queue is disabled by default. No queue side effects when disabled. Business idempotency/outbox/compensation belongs to downstream projects, not this template.
- Passwords/tokens/SQL parameters/request bodies must not enter logs. No real credentials in tracked files.
- Each command initializes only required components. Help must work without config or services.
- Keep internal/infrastructure/cli/root.go limited to root options and explicit command registration. Business commands belong in internal/command with register.go; they must not import cli. Put each standalone command in its own file; related subcommands such as migrate stay together. Initialize resources only when executing a command.
- Keep the scheduler engine and Task type in internal/infrastructure/scheduler; it must not import concrete tasks. Register scheduled tasks explicitly in internal/schedule/register.go, with each task in its own file. heartbeat.go is a removable example task; keep its registration explicit.
- Run gofmt, go test ./..., and go vet ./... for Go changes. For integration changes, run the documented opt-in tests against disposable services; distinguish skipped tests from passed tests.
- Keep README and API/development docs consistent with behavior. Do not push or create commits unless asked.
