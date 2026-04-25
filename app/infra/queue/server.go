package queue

import (
	"github.com/hibiken/asynq"

	"go-boilerplate/app/infra/queue/handlers"
	"go-boilerplate/app/infra/queue/tasks"
	"go-boilerplate/app/shared/ports"
)

type Server struct {
	srv *asynq.Server
	mux *asynq.ServeMux
}

func NewServer(redisAddr, redisPassword string, queueDB, concurrency int) *Server {
	opt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       queueDB,
	}
	srv := asynq.NewServer(opt, asynq.Config{
		Concurrency: concurrency,
	})
	return &Server{
		srv: srv,
		mux: asynq.NewServeMux(),
	}
}

func (s *Server) RegisterHandlers(notifier ports.Notifier) {
	emailHandler := handlers.NewEmailHandler(notifier)
	s.mux.HandleFunc(tasks.TypeSendEmail, emailHandler.Process)
	s.mux.HandleFunc(tasks.TypeExampleTask, handlers.ProcessExampleTask)
}

func (s *Server) Start() error {
	return s.srv.Run(s.mux)
}

func (s *Server) Stop() {
	s.srv.Shutdown()
}
