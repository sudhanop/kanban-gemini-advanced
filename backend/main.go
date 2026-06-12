package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/kanban-platform/backend/internal/config"
	"github.com/kanban-platform/backend/internal/database"
	"github.com/kanban-platform/backend/internal/routes"
	"github.com/kanban-platform/backend/internal/websocket"
	"github.com/kanban-platform/backend/internal/workers"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		// Try parent directory fallback
		if err := godotenv.Load("../.env"); err != nil {
			log.Println("No .env file found, using environment variables")
		}
	}

	// Initialize config
	cfg := config.Load()

	// Initialize logger
	zapLogger, _ := zap.NewProduction()
	if cfg.AppEnv == "development" {
		zapLogger, _ = zap.NewDevelopment()
	}
	defer zapLogger.Sync()

	// Initialize database connection (SQLite in-memory)
	db, err := database.InitSQLite(cfg)
	if err != nil {
		zapLogger.Fatal("Failed to connect to SQLite", zap.Error(err))
	}

	// Run migrations
	if err := database.AutoMigrateSQLite(db); err != nil {
		zapLogger.Fatal("Failed to run migrations", zap.Error(err))
	}

	// Load data from db.json if exists
	if err := database.LoadFromJSON(db); err != nil {
		zapLogger.Error("Failed to load database from db.json", zap.Error(err))
	}

	// Register save hooks for JSON persistence
	database.RegisterSaveCallbacks(db)

	// Initialize WebSocket hub
	hub := websocket.NewHub(zapLogger)
	go hub.Run()

	// Initialize background workers
	workerManager := workers.NewManager(db, hub, cfg, zapLogger)
	workerManager.Start()

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Kanban Platform API",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		ErrorHandler: customErrorHandler,
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${method} ${path}\n",
	}))
	app.Use(helmet.New())
	app.Use(compress.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.FrontendURL,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: true,
	}))
	app.Use(limiter.New(limiter.Config{
		Max:        cfg.RateLimitRequests,
		Expiration: time.Duration(cfg.RateLimitWindow) * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests. Please slow down.",
			})
		},
	}))

	// Static file serving for uploads
	app.Static("/uploads", cfg.StorageLocalPath)

	// Register all routes
	routes.Register(app, db, hub, cfg, zapLogger)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
		})
	})

	// Start server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%s", cfg.AppPort)
		zapLogger.Info("Server starting", zap.String("address", addr))
		if err := app.Listen(addr); err != nil {
			zapLogger.Fatal("Server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	workerManager.Stop()
	hub.Stop()

	if err := app.ShutdownWithContext(ctx); err != nil {
		zapLogger.Error("Error during shutdown", zap.Error(err))
	}

	zapLogger.Info("Server stopped gracefully")
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal server error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error":   message,
		"status":  code,
		"path":    c.Path(),
		"method":  c.Method(),
	})
}
