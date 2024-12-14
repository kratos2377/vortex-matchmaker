package main

import (
	"log"
	"net/http"
	"time"

	"github.com/kratos2377/vortex-matchmaker/app/api/handlers"
	"github.com/kratos2377/vortex-matchmaker/domain/matchmaking"
	"github.com/kratos2377/vortex-matchmaker/domain/tickets"
	"github.com/spf13/viper"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Port                           string        `mapstructure:"PORT"`
	RedisAddress                   string        `mapstructure:"REDIS_ADDRESS"`
	RedisPassword                  string        `mapstructure:"REDIS_PASSWORD"`
	RedisDB                        int           `mapstructure:"REDIS_DB"`
	RedisTicketsSetName            string        `mapstructure:"REDIS_TICKETS_SET_NAME"`
	RedisMatchesSetName            string        `mapstructure:"REDIS_MATCHES_SET_NAME"`
	RedisCountPerIteration         int64         `mapstructure:"REDIS_COUNT_PER_ITERATION"`
	MatchmakerMinPlayersPerSession int32         `mapstructure:"MATCHMAKER_MIN_PLAYERS_PER_SESSION"`
	MatchmakerMaxPlayersPerSession int32         `mapstructure:"MATCHMAKER_MAX_PLAYERS_PER_SESSION"`
	MatchmakerTimeout              time.Duration `mapstructure:"MATCHMAKER_TIMEOUT"`
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
	cfg, err := LoadConfig("./app/api")
	if err != nil {
		panic(err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		DB:       cfg.RedisDB,
		Password: cfg.RedisPassword,
	})

	ticketsAPIUseCases := &struct {
		*tickets.CreateTicketUseCase
		*tickets.GetTicketUseCase
	}{
		CreateTicketUseCase: tickets.NewCreateTicketUseCase(redisClient, cfg.RedisTicketsSetName),
		GetTicketUseCase:    tickets.NewGetTicketUseCase(redisClient, cfg.RedisTicketsSetName, cfg.RedisMatchesSetName),
	}

	matchmakingAPIUseCases := &struct {
		*matchmaking.MatchPlayersUseCase
	}{
		MatchPlayersUseCase: matchmaking.NewMatchPlayersUseCase(redisClient, matchmaking.MatchPlayerUseCaseConfig{
			MinCountPerMatch:    cfg.MatchmakerMinPlayersPerSession,
			MaxCountPerMatch:    cfg.MatchmakerMaxPlayersPerSession,
			TicketsRedisSetName: cfg.RedisTicketsSetName,
			MatchesRedisSetName: cfg.RedisMatchesSetName,
			Timeout:             cfg.MatchmakerTimeout,
			CountPerIteration:   cfg.RedisCountPerIteration,
		}),
	}

	apiUseCases := handlers.UseCases{
		TicketsAPIUseCases:     ticketsAPIUseCases,
		MatchmakingAPIUseCases: matchmakingAPIUseCases,
	}

	server := handlers.NewServer(apiUseCases)

	log.Fatal(http.ListenAndServe(":8000", server))

}
