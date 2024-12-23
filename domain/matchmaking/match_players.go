package matchmaking

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/kratos2377/vortex-matchmaker/domain/entities"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

type MatchPlayersUseCaseRedisGateway interface {
	HScan(ctx context.Context, key string, cursor uint64, match string, count int64) *redis.ScanCmd
	ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd
	ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
}

type MatchPlayerUseCaseConfig struct {
	MinCountPerMatch    int32
	MaxCountPerMatch    int32
	TicketsRedisSetName string
	MatchesRedisSetName string
	Timeout             time.Duration
	CountPerIteration   int64
}
type MatchPlayersUseCase struct {
	redisGateway MatchPlayersUseCaseRedisGateway
	cfg          MatchPlayerUseCaseConfig
	conn         *kafka.Conn
}

func NewMatchPlayersUseCase(conn *kafka.Conn, redisClient MatchPlayersUseCaseRedisGateway, config MatchPlayerUseCaseConfig) *MatchPlayersUseCase {
	return &MatchPlayersUseCase{conn: conn, redisGateway: redisClient, cfg: config}
}

type MatchPlayerInput struct {
	MinCount int32
	MaxCount int32
}

type MatchPlayersOutput struct {
	CreatedSessions []PlayerSession
	GameType        string
}

type EligiblePlayerStruct struct {
	PlayerId       string
	PlayerUsername string
}

type PlayerSession struct {
	SessionID string
	PlayerIds []EligiblePlayerStruct
	GameType  string
}

// MatchPlayers tries to match all tickets opened by players.
// If a player's ticket exceeds the expiration time, reduces by one the amount of players
// needed for a perfect match. After that, if no match is found, sets the ticket as expired,
// so it can no longer match with other players.
func (m *MatchPlayersUseCase) MatchPlayers(ctx context.Context) (MatchPlayersOutput, error) {
	var cursor uint64
	var tickets []string
	var err error

	log.Println("Matching Players...")
	var matchedSessions []PlayerSession
	alreadyMatchedPlayers := map[string]bool{}
	var gameTypeInt int64
	for {
		result := m.redisGateway.HScan(ctx, m.cfg.TicketsRedisSetName, cursor, "", m.cfg.CountPerIteration)
		tickets, cursor, err = result.Result()
		if err != nil {
			return MatchPlayersOutput{}, err
		}

		for i := 0; i < len(tickets); i = i + 2 {
			if alreadyMatchedPlayers[tickets[i]] == true {
				continue
			}

			playerTicketBytes := []byte(tickets[i+1])

			var playerTicket entities.MatchmakingTicket
			err = json.Unmarshal(playerTicketBytes, &playerTicket)
			if err != nil {
				return MatchPlayersOutput{}, err
			}

			for _, param := range playerTicket.MatchParameters {
				if param.Type == "game_type" {
					gameTypeInt = int64(param.Value)
					break
				}
			}

			// We don't try to match with anyone if the ticket has expired
			if playerTicket.Status == entities.MatchmakingStatus_Expired {
				continue
			}

			hasExpired := time.Now().Unix() > playerTicket.CreatedAt+int64(m.cfg.Timeout.Seconds())

			maxCountForThisPlayer := m.cfg.MaxCountPerMatch
			// when has reached the time limit, we decrease the max amount for a perfect by 1
			// if hasExpired && maxCountForThisPlayer-1 >= m.cfg.MinCountPerMatch {
			// 	maxCountForThisPlayer--
			// }

			var eligibleOpponents []EligiblePlayerStruct
			// Append the player
			eligibleOpponents = append(eligibleOpponents, EligiblePlayerStruct{
				PlayerId:       playerTicket.PlayerId,
				PlayerUsername: playerTicket.PlayerUsername,
			})

			eligibleOpponentsCountMap := map[string]int{}
			for _, parameter := range playerTicket.MatchParameters {
				var result *redis.StringSliceCmd
				switch parameter.Operator {
				case entities.MatchmakingTicketParameterOperator_Equal:
					result = m.redisGateway.ZRangeByScore(ctx, string(parameter.Type), &redis.ZRangeBy{
						Min:   fmt.Sprint(parameter.Value),
						Max:   fmt.Sprint(parameter.Value),
						Count: int64(m.cfg.MaxCountPerMatch),
					})
				// case entities.MatchmakingTicketParameterOperator_GreaterThan:
				// 	result = m.redisGateway.ZRangeByScore(ctx, string(parameter.Type), &redis.ZRangeBy{
				// 		Min:   fmt.Sprintf("(%f", parameter.Value),
				// 		Max:   "+inf",
				// 		Count: int64(m.cfg.MaxCountPerMatch),
				// 	})
				// case entities.MatchmakingTicketParameterOperator_SmallerThan:
				// 	result = m.redisGateway.ZRangeByScore(ctx, string(parameter.Type), &redis.ZRangeBy{
				// 		Min:   "0",
				// 		Max:   fmt.Sprintf("(%f", parameter.Value),
				// 		Count: int64(m.cfg.MaxCountPerMatch),
				// 	})
				// case entities.MatchmakingTicketParameterOperator_NotEqual:
				// 	// TODO: support not equal operator
				// 	continue
				default:
					// TODO: return error
					continue
				}

				// This will return the player ids of the eligible opponents
				foundOpponents, err := result.Result()
				println("FOund oppnents are")
				fmt.Printf("%+v", foundOpponents)
				if err != nil {
					return MatchPlayersOutput{}, err
				}

				for _, opponentStruct := range foundOpponents {

					println("Inside foundOpponents loop")

					var opponent EligiblePlayerStruct
					json.Unmarshal([]byte(opponentStruct), &opponent)

					if opponent.PlayerId == playerTicket.PlayerId {
						continue
					}
					c, ok := eligibleOpponentsCountMap[opponent.PlayerId]
					if !ok {
						eligibleOpponentsCountMap[opponent.PlayerId] = 1
					} else {
						eligibleOpponentsCountMap[opponent.PlayerId] = c + 1
					}

					if eligibleOpponentsCountMap[opponent.PlayerId] == len(playerTicket.MatchParameters) {
						eligibleOpponents = append(eligibleOpponents, opponent)
					}

					if int32(len(eligibleOpponents)) == m.cfg.MaxCountPerMatch {
						break
					}
				}

			}

			// Found a match!
			println("Length of eligible oppnents is")
			println(len(eligibleOpponents))
			if int32(len(eligibleOpponents)) == maxCountForThisPlayer {
				// this could be an id or the address of a game server match
				gameSessionId := uuid.New().String()
				matchedSessions = append(matchedSessions, PlayerSession{PlayerIds: eligibleOpponents, SessionID: gameSessionId})
				for _, opponent := range eligibleOpponents {
					for _, parameter := range playerTicket.MatchParameters {
						if err = m.redisGateway.ZRem(ctx, string(parameter.Type), opponent.PlayerId).Err(); err != nil {
							log.Println(err)
							return MatchPlayersOutput{}, err
						}
					}
					if err = m.redisGateway.HDel(ctx, m.cfg.TicketsRedisSetName, opponent.PlayerId).Err(); err != nil {
						return MatchPlayersOutput{}, err
					}
					alreadyMatchedPlayers[opponent.PlayerId] = true

					// creates a registry in Matches for each opponent
					playerTicket.Status = entities.MatchmakingStatus_Found
					playerTicket.GameSessionId = gameSessionId
					m.redisGateway.HSet(ctx, m.cfg.MatchesRedisSetName, opponent.PlayerId, playerTicket)
				}
				// sets the ticket as expired and removes from parameters sets, so it is not tried again
			} else if hasExpired {
				playerTicket.Status = entities.MatchmakingStatus_Expired
				if err = m.redisGateway.HSet(ctx, m.cfg.TicketsRedisSetName, playerTicket.PlayerId, playerTicket).Err(); err != nil {
					return MatchPlayersOutput{}, err
				}

				for _, parameter := range playerTicket.MatchParameters {
					if err = m.redisGateway.ZRem(ctx, string(parameter.Type), playerTicket.PlayerId).Err(); err != nil {
						return MatchPlayersOutput{}, err
					}
				}
			}

		}

		// Finished iterating through matchmaking tickets
		if cursor == 0 {
			break
		}
	}

	log.Println("Matched Players: ", matchedSessions)

	var gameTypeString string
	if gameTypeInt == 0 {
		gameTypeString = "normal"
	} else {
		gameTypeString = "staked"
	}

	if len(matchedSessions) != 0 {
		converted_json, err := json.Marshal(MatchPlayersOutput{
			CreatedSessions: matchedSessions,
			GameType:        gameTypeString,
		})

		if err != nil {
			log.Fatal("Error while converting matchedSessions to json marshal")
		}

		m.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

		_, err = m.conn.WriteMessages(kafka.Message{
			Key:   []byte("match-found"),
			Value: []byte(string(converted_json))})

		if err != nil {
			log.Panicln("Error while publishing messages to key")
		}
	}

	return MatchPlayersOutput{
		CreatedSessions: matchedSessions,
		GameType:        gameTypeString,
	}, nil
}
