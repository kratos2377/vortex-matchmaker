package main

import (
	"context"
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/kratos2377/vortex-matchmaker/domain/tickets"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type Config struct {
	RedisAddress                string        `mapstructure:"REDIS_ADDRESS"`
	RedisPassword               string        `mapstructure:"REDIS_PASSWORD"`
	RedisDB                     int           `mapstructure:"REDIS_DB"`
	RedisTicketsSetName         string        `mapstructure:"REDIS_TICKETS_SET_NAME"`
	RedisCountPerIteration      int64         `mapstructure:"REDIS_COUNT_PER_ITERATION"`
	TicketsTimeBeforeToRemove   time.Duration `mapstructure:"TICKETS_TIME_BEFORE_TO_REMOVE"`
	WorkerTimeScheduleInSeconds time.Duration `mapstructure:"WORKER_TIME_SCHEDULE_IN_SECONDS"`
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
	cfg, err := LoadConfig("./app/cleaner_worker")
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

	removeExpiredTicketsUseCase := tickets.NewRemoveExpiredTicketsUseCase(redisClient, tickets.RemoveExpiredTicketsUseCaseConfig{
		TicketsRedisSetName: cfg.RedisTicketsSetName,
		TimeBeforeToRemove:  cfg.TicketsTimeBeforeToRemove,
		CountPerIteration:   cfg.RedisCountPerIteration,
	})

	_, err = s.NewJob(
		gocron.DurationJob(
			1*time.Minute,
		),

		gocron.NewTask(
			removeExpiredTicketsUseCase.RemoveExpiredTickets, context.Background()),
	)

	//fmt.Println("Started new ticket cleaning job with Id=", nj.ID())

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
		log.Println("Error while shutting down clean scheduler")
	}
}
