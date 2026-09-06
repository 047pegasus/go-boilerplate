package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/047pegasus/go-boilerplate/internal/config"
	"github.com/047pegasus/go-boilerplate/internal/database"
	"github.com/047pegasus/go-boilerplate/internal/lib/job"
	loggerPkg "github.com/047pegasus/go-boilerplate/internal/logger"
	"github.com/047pegasus/go-boilerplate/internal/server/custom"
	customVkUtils "github.com/047pegasus/go-boilerplate/internal/server/custom/custom_utils"
	"github.com/rs/zerolog"
	"github.com/valkey-io/valkey-go"
)

type Server struct {
	Config        *config.Config
	Logger        *zerolog.Logger
	LoggerService *loggerPkg.LoggerService
	DB            *database.Database
	Cache         valkey.Client
	httpServer    *http.Server
	Job           *job.JobService
}

func New(cfg *config.Config, logger *zerolog.Logger, loggerService *loggerPkg.LoggerService) (*Server, error) {
	//Initialize Database
	db, err := database.New(cfg, logger, loggerService)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize/connect to database: %w", err)
	}

	//Initialize Cache Client - (I will use Valkey instead of Redis)
	valkeyClient, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{cfg.Cache.Address},
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize/connect to cache(valkey): %w", err) //rename with appropriate cache client
	}

	//Add Sentry Redis Hooks if available
	if loggerService != nil && loggerService.GetApplication() != nil {
		valkeyClient = custom.WithSentryHook(valkeyClient)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//Build Ping  for Valkey & execute it
	if _, err := customVkUtils.PingValkeyBuildAndExecute(ctx, valkeyClient); err != nil {
		logger.Error().Err(err).Msg("Valkey Unavailable, continuing startup...")
	}

	//init Job Service
	jobSvc := job.NewJobService(logger, cfg)
	jobSvc.InitHandlers(cfg, logger)
	//start the job server
	if err := jobSvc.Start(); err != nil {
		return nil, fmt.Errorf("Failed to start job service: %w", err)
	}

	//Make final struct of Server instance
	server := &Server{
		Config:        cfg,
		Logger:        logger,
		LoggerService: loggerService,
		DB:            db,
		Cache:         valkeyClient,
		Job:           jobSvc,
	}
	return server, nil
}

func (s *Server) SetupHttpServer(handler http.Handler) {
	s.httpServer = &http.Server{
		Addr:         ":" + s.Config.Server.Port,
		Handler:      handler,
		ReadTimeout:  time.Duration(s.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.Config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.Config.Server.IdleTimeout) * time.Second,
	}
}

func (s *Server) Start() error {
	if s.httpServer == nil {
		return errors.New("Server is not initialized")
	}
	s.Logger.Info().Str("port", fmt.Sprintf(s.Config.Server.Port)).Str("env", s.Config.Primary.Env).Msg("Starting server")
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	//start shutting the actual http server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("Failed to shutdown http server: %w", err)
	}
	//Shutdown DB connection and onoging transactions
	if err := s.DB.Close(); err != nil {
		return fmt.Errorf("Failed to close database connection: %w", err)
	}
	//Shutdown Cache connection
	if s.Cache != nil {
		s.Cache.Close()
	}
	//Stop job server gracefully by handling ongoing jobs
	if s.Job != nil {
		s.Job.Stop()
	}
	return nil
}
