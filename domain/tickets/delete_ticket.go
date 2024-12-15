package tickets

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type DeleteTicketsUseCaseRedisGateway interface {
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
}

func NewDeleteTicketUseCase(redisGateway DeleteTicketsUseCaseRedisGateway, ticketsRedisSetName, matchesRedisSetName string) *DeleteTicketUseCase {
	return &DeleteTicketUseCase{redisGateway: redisGateway, ticketsRedisSetName: ticketsRedisSetName, matchesRedisSetName: matchesRedisSetName}
}

type DeleteTicketUseCase struct {
	redisGateway        DeleteTicketsUseCaseRedisGateway
	ticketsRedisSetName string
	matchesRedisSetName string
}

type DeleteTicketInput struct {
	PlayerId string
}

type DeleteTicketOutput struct {
	success bool
}

func (c *DeleteTicketUseCase) DeleteTicket(ctx context.Context, input DeleteTicketInput) (DeleteTicketOutput, error) {

	if err := c.redisGateway.HDel(ctx, c.ticketsRedisSetName, input.PlayerId).Err(); err != nil {
		return DeleteTicketOutput{}, err
	}

	return DeleteTicketOutput{
		success: true,
	}, nil
}
