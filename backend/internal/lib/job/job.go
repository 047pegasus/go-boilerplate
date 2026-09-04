package job

import (
	"github.com/047pegasus/go-boilerplate/internal/config"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

type JobService struct {
	Client *asynq.Client
	server *asynq.Server
	logger *zerolog.Logger
}

func NewJobService(logger *zerolog.Logger, cfg *config.Config) *JobService {
	redisAddr := cfg.Cache.Address
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr: redisAddr,
	})
	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6, //high priority for urgent emails
				"default":  3, //default priority for most emails
				"low":      1, //low priority for non-urgent emails
			},
		},
	)
	return &JobService{
		Client: client,
		server: server,
		logger: logger,
	}
}

func (j *JobService) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskWelcome, j.handleWelcomeEmailTask)
	j.logger.Info().Msg("Starting Background Job Service...")
	if err := j.server.Start(mux); err != nil {
		return err
	}
	return nil
}

func (j *JobService) Stop() {
	j.logger.Info().Msg("Stopping Background Job Service !!")
	j.server.Shutdown()
	_ = j.Client.Close()
}
