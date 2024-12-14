package main

import (
	"context"
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/kratos2377/vortex-matchmaker/domain/matchmaking"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type Config struct {
	RedisAddress                   string        `mapstructure:"REDIS_ADDRESS"`
	RedisPassword                  string        `mapstructure:"REDIS_PASSWORD"`
	RedisDB                        int           `mapstructure:"REDIS_DB"`
	RedisTicketsSetName            string        `mapstructure:"REDIS_TICKETS_SET_NAME"`
	RedisMatchesSetName            string        `mapstructure:"REDIS_MATCHES_SET_NAME"`
	RedisCountPerIteration         int64         `mapstructure:"REDIS_COUNT_PER_ITERATION"`
	MatchmakerMinPlayersPerSession int32         `mapstructure:"MATCHMAKER_MIN_PLAYERS_PER_SESSION"`
	MatchmakerMaxPlayersPerSession int32         `mapstructure:"MATCHMAKER_MAX_PLAYERS_PER_SESSION"`
	MatchmakerTimeout              time.Duration `mapstructure:"MATCHMAKER_TIMEOUT"`
	WorkerTimeScheduleInSeconds    time.Duration `mapstructure:"WORKER_TIME_SCHEDULE_IN_SECONDS"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}

func main() {
	cfg, err := LoadConfig("./app/matchmaking")
	if err != nil {
		log.Fatal(err)
	}

	s, err := gocron.NewScheduler()

	if err != nil {
		log.Fatal(err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		DB:       cfg.RedisDB,
		Password: cfg.RedisPassword,
	})

	matchmakingUseCase := matchmaking.NewMatchPlayersUseCase(redisClient, matchmaking.MatchPlayerUseCaseConfig{
		MinCountPerMatch:    cfg.MatchmakerMinPlayersPerSession,
		MaxCountPerMatch:    cfg.MatchmakerMaxPlayersPerSession,
		TicketsRedisSetName: cfg.RedisTicketsSetName,
		MatchesRedisSetName: cfg.RedisMatchesSetName,
		Timeout:             cfg.MatchmakerTimeout,
		CountPerIteration:   cfg.RedisCountPerIteration,
	})

	_, err = s.NewJob(
		gocron.DurationJob(
			cfg.WorkerTimeScheduleInSeconds*time.Second,
		),
		gocron.NewTask(matchmakingUseCase.MatchPlayers, context.Background()),
	)

	//fmt.Println("Started new matchmaking job with Id=", nj.ID())

	if err != nil {
		log.Fatal(err)
	}

	s.Start()
}
