package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratePostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"

	"go-boilerplate/app/bootstrap"
	usersFeature "go-boilerplate/app/features/users"
	"go-boilerplate/app/infra/database"
	dbUsers "go-boilerplate/app/infra/database/users"
	"go-boilerplate/app/infra/logger"
	"go-boilerplate/app/infra/notification"
	"go-boilerplate/app/shared/config"
	"go-boilerplate/app/shared/token"
	// scaffold:main-imports
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg)

	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Error("failed to connect to database", err)
		os.Exit(1)
	}
	defer db.Close()

	if cfg.RunMigrations {
		runMigrations(db)
	}

	tokenMaker := token.NewJWTMaker(cfg.JWTSecret)
	notifier := notification.NewMockNotifier()

	usersRepo := dbUsers.NewPgRepository(db)
	usersSvc := usersFeature.NewService(usersRepo, usersRepo, notifier, tokenMaker)
	usersHandler := usersFeature.NewHandler(usersSvc)
	// scaffold:feature-wire

	hashFn := func(ctx context.Context, userID uuid.UUID) (string, error) {
		user, err := usersRepo.FindByID(ctx, userID)
		if err != nil {
			return "", err
		}
		return user.PasswordHash, nil
	}

	e := bootstrap.NewEcho(log)
	bootstrap.RegisterRoutes(e, usersHandler, tokenMaker, hashFn /* scaffold:feature-call */)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.AppPort)
		log.Info("server starting", "addr", addr)
		if err := e.Start(addr); err != nil {
			log.Info("server stopped")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", err)
	}
}

func runMigrations(db *sql.DB) {
	driver, err := migratePostgres.WithInstance(db, &migratePostgres.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate driver error: %v\n", err)
		return
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://app/infra/database/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate init error: %v\n", err)
		return
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		fmt.Fprintf(os.Stderr, "migrate up error: %v\n", err)
	}
}
