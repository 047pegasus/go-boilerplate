package repository

import "github.com/047pegasus/go-boilerplate/internal/server"

type Repositories struct{}

func NewRepositories(server *server.Server) *Repositories {
	return &Repositories{}
}
