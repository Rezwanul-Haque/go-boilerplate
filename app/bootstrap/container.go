package bootstrap

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"go-boilerplate/app/features/health"
	usersFeature "go-boilerplate/app/features/users"
	dbUsers "go-boilerplate/app/infra/database/users"
	"go-boilerplate/app/infra/notification"
	"go-boilerplate/app/shared/config"
	"go-boilerplate/app/shared/token"
	// scaffold:container-imports
)

type Container struct {
	TokenMaker    token.Maker
	HashFn        func(ctx context.Context, userID uuid.UUID) (string, error)
	HealthHandler *health.Handler
	UsersHandler  *usersFeature.Handler
	// scaffold:container-fields
}

func NewContainer(db *sql.DB, cfg *config.Config) *Container {
	tokenMaker := token.NewJWTMaker(cfg.JWTSecret)
	notifier := notification.NewMockNotifier()

	usersRepo := dbUsers.NewPgRepository(db)
	usersSvc := usersFeature.NewService(usersRepo, usersRepo, notifier, tokenMaker)

	hashFn := func(ctx context.Context, userID uuid.UUID) (string, error) {
		user, err := usersRepo.FindByID(ctx, userID)
		if err != nil {
			return "", err
		}
		return user.PasswordHash, nil
	}

	// scaffold:container-wire
	return &Container{
		TokenMaker:    tokenMaker,
		HashFn:        hashFn,
		HealthHandler: health.NewHandler(db),
		UsersHandler:  usersFeature.NewHandler(usersSvc),
		// scaffold:container-init
	}
}
