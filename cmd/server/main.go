package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/agenthub/server/internal/config"
	"github.com/agenthub/server/internal/handler"
	"github.com/agenthub/server/internal/middleware"
	"github.com/agenthub/server/internal/pkg/conflict"
	"github.com/agenthub/server/internal/pkg/events"
	"github.com/agenthub/server/internal/pkg/ws"
	"github.com/agenthub/server/internal/repository"
	"github.com/agenthub/server/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Setup structured logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", "agenthub").Logger()
	if cfg.IsDevelopment() {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	logger.Info().
		Str("port", cfg.Port).
		Str("env", cfg.Env).
		Msg("starting AgentHub server")

	// Connect to PostgreSQL
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to ping database")
	}
	logger.Info().Msg("connected to PostgreSQL")

	// Run migrations
	if err := runMigrations(ctx, pool); err != nil {
		logger.Fatal().Err(err).Msg("failed to run migrations")
	}
	logger.Info().Msg("database migrations applied")

	// Initialize infrastructure
	eventBus := events.NewBus()
	wsHub := ws.NewHub()
	go wsHub.Run(nil)

	conflictResolver := conflict.NewResolver()

	// Initialize repositories
	workspaceRepo := repository.NewWorkspaceRepository(pool)
	agentRepo := repository.NewAgentRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)
	messageRepo := repository.NewMessageRepository(pool)
	artifactRepo := repository.NewArtifactRepository(pool)
	contextRepo := repository.NewContextRepository(pool)
	syncRepo := repository.NewSyncRepository(pool)
	dailyReportRepo := repository.NewDailyReportRepository(pool)

	// Initialize services
	workspaceService := service.NewWorkspaceService(workspaceRepo, agentRepo, eventBus, cfg.JWT.Secret, cfg.JWT.Expire)
	taskService := service.NewTaskService(taskRepo, syncRepo, eventBus)
	messagingService := service.NewMessagingService(messageRepo, syncRepo, eventBus, wsHub)
	artifactService := service.NewArtifactService(artifactRepo, syncRepo, eventBus, conflictResolver)
	contextService := service.NewContextService(contextRepo, syncRepo, eventBus, conflictResolver)
	syncEngine := service.NewSyncEngine(syncRepo, wsHub, eventBus, conflictResolver)
	dailyReportService := service.NewDailyReportService(dailyReportRepo, taskRepo)
	orchestratorService := service.NewOrchestratorService(
		taskService, messagingService, contextService, agentRepo,
		cfg.Orchestrator.CheckInterval, cfg.Orchestrator.StaleTaskHours,
	)

	// Start orchestrator background loop
	orchestratorService.Start()
	defer orchestratorService.Stop()

	// Setup Gin router
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware(cfg.Env, nil))

	// Health check (no auth required)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "1.0.0",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Rate limiter
	rateLimiter := middleware.NewRateLimiter(100)
	defer rateLimiter.Stop()

	// API v1 routes
	authMW := middleware.AuthMiddleware(cfg.JWT.Secret)
	v1 := router.Group("/api/v1")
	v1.Use(authMW)
	v1.Use(middleware.RateLimitMiddleware(rateLimiter))

	// Register all handlers
	workspaceHandler := handler.NewWorkspaceHandler(workspaceService)
	workspaceHandler.RegisterRoutes(v1, authMW)

	taskHandler := handler.NewTaskHandler(taskService)
	taskHandler.RegisterRoutes(v1)

	messageHandler := handler.NewMessageHandler(messagingService)
	messageHandler.RegisterRoutes(v1)

	artifactHandler := handler.NewArtifactHandler(artifactService)
	artifactHandler.RegisterRoutes(v1)

	contextHandler := handler.NewContextHandler(contextService)
	contextHandler.RegisterRoutes(v1)

	syncHandler := handler.NewSyncHandler(syncEngine)
	syncHandler.RegisterRoutes(v1)

	dailyReportHandler := handler.NewDailyReportHandler(dailyReportService)
	dailyReportHandler.RegisterRoutes(v1)

	// WebSocket handler (uses query param auth, not middleware)
	wsHandler := handler.NewWSHandlerWithConfig(wsHub, cfg.JWT.Secret, cfg.WS.PingInterval, cfg.WS.PongTimeout)
	wsHandler.RegisterRoutes(router)

	// Wire up event bus to broadcast WebSocket events to all workspaces
	eventBus.SubscribeAll(func(event events.Event) {
		wsID, err := uuid.Parse(event.WorkspaceID)
		if err != nil {
			return
		}
		data := map[string]interface{}{
			"event":     event.Type,
			"data":      event.Data,
			"timestamp": event.Timestamp.UTC().Format(time.RFC3339),
		}
		payload, err := json.Marshal(data)
		if err != nil {
			return
		}
		wsHub.BroadcastToWorkspace(wsID, payload)
	})

	// Start HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("server failed")
		}
	}()

	logger.Info().Str("addr", ":"+cfg.Port).Msg("AgentHub server started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal().Err(err).Msg("server forced to shutdown")
	}

	logger.Info().Msg("server stopped")
}

// runMigrations executes SQL migration files in order.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		return fmt.Errorf("migrations directory not found")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		content, err := os.ReadFile(filepath.Join(migrationsDir, file))
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", file, err)
		}

		if _, err := pool.Exec(ctx, string(content)); err != nil {
			// Ignore "already exists" errors for idempotent migrations
			if !strings.Contains(err.Error(), "already exists") &&
				!strings.Contains(err.Error(), "duplicate key") {
				return fmt.Errorf("executing migration %s: %w", file, err)
			}
		}
	}

	return nil
}

// findMigrationsDir searches for the migrations directory relative to the executable.
func findMigrationsDir() string {
	candidates := []string{
		"migrations",
		"./migrations",
		"../migrations",
		"../../migrations",
	}

	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "migrations"))
	}

	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return ""
}
