package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/kratos2377/vortex-matchmaker/domain/matchmaking"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
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
	WorkerTimeScheduleInSeconds    int64         `mapstructure:"WORKER_TIME_SCHEDULE_IN_SECONDS"`
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

const (
	KafkaServer = "localhost:9092"
	KafkaTopic  = "user-matchmaking"
)

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
		Addr:            cfg.RedisAddress,
		DB:              cfg.RedisDB,
		Password:        cfg.RedisPassword,
		MaxRetries:      3,
		ConnMaxIdleTime: 3 * time.Minute,
	})

	_, err = redisClient.Ping(context.Background()).Result()

	if err != nil {
		log.Println(err)
		return
	}

	conn, err := kafka.DialLeader(context.Background(), "tcp",
		"localhost:9092", "user-matchmaking", 0)
	if err != nil {
		fmt.Println("failed to dial leader")
	}

	defer conn.Close()

	matchmakingUseCase := matchmaking.NewMatchPlayersUseCase(conn, redisClient, matchmaking.MatchPlayerUseCaseConfig{
		MinCountPerMatch:    cfg.MatchmakerMinPlayersPerSession,
		MaxCountPerMatch:    cfg.MatchmakerMaxPlayersPerSession,
		TicketsRedisSetName: cfg.RedisTicketsSetName,
		MatchesRedisSetName: cfg.RedisMatchesSetName,
		Timeout:             cfg.MatchmakerTimeout,
		CountPerIteration:   cfg.RedisCountPerIteration,
	})

	_, err = s.NewJob(
		gocron.DurationJob(
			10*time.Second,
		),
		gocron.NewTask(matchmakingUseCase.MatchPlayers, context.Background()),
	)

	//fmt.Println("Started new matchmaking job with Id=", nj.ID())

	if err != nil {
		log.Fatal(err)
	}

	s.Start()

	select {
	case <-time.After(100 * time.Minute):
	}

	// when you're done, shut it down
	err = s.Shutdown()
	if err != nil {
		// handle error
		log.Println("Error while shutting down matchmaking scheduler")
	}
}
