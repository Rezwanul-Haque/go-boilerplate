package bootstrap

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"go-boilerplate/app/features/health"
	"go-boilerplate/app/features/posts"
	usersFeature "go-boilerplate/app/features/users"
	cacheInfra "go-boilerplate/app/infra/cache"
	dbUsers "go-boilerplate/app/infra/database/users"
	"go-boilerplate/app/infra/httpclient"
	"go-boilerplate/app/infra/notification"
	"go-boilerplate/app/shared/config"
	"go-boilerplate/app/shared/ports"
	"go-boilerplate/app/shared/token"
	// scaffold:container-imports
)

type Container struct {
	TokenMaker    token.Maker
	HashFn        func(ctx context.Context, userID uuid.UUID) (string, error)
	Cache         ports.Cache
	HTTPClient    ports.HTTPClient
	HealthHandler *health.Handler
	UsersHandler  *usersFeature.Handler
	PostsHandler  *posts.Handler
	// scaffold:container-fields
}

func NewContainer(db *sql.DB, cfg *config.Config, log ports.Logger, redisCache ports.Cache) *Container {
	tokenMaker := token.NewJWTMaker(cfg.JWTSecret)
	notifier := notification.NewMockNotifier()
	resetRepo := cacheInfra.NewResetTokenRepo(redisCache)

	usersRepo := dbUsers.NewPgRepository(db)
	usersSvc := usersFeature.NewService(usersRepo, resetRepo, notifier, tokenMaker)

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
		Cache:         redisCache,
		HTTPClient:    httpclient.New(cfg),
		HealthHandler: health.NewHandler(db, redisCache),
		UsersHandler:  usersFeature.NewHandler(usersSvc),
		PostsHandler:  posts.NewHandler(posts.NewService(httpclient.New(cfg), redisCache)),
		// scaffold:container-init
	}
}
