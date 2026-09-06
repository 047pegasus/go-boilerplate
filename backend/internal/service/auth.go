package service

import (
	"github.com/047pegasus/go-boilerplate/internal/server"
	"github.com/clerk/clerk-sdk-go/v2"
)

type AuthService struct {
	server *server.Server
}

func NewAuthService(server *server.Server) *AuthService {
	clerk.SetKey(server.Config.Auth.SecretKey)
	return &AuthService{server: server}
}
