package service

import (
	"github.com/047pegasus/go-boilerplate/internal/lib/job"
	"github.com/047pegasus/go-boilerplate/internal/repository"
	"github.com/047pegasus/go-boilerplate/internal/server"
)

type Services struct {
	Auth *AuthService
	Job  *job.JobService
}

func NewServices(server *server.Server, repos *repository.Repositories) (*Services, error) {
	authService := NewAuthService(server)
	return &Services{
		Auth: authService,
		Job:  server.Job,
	}, nil
}
