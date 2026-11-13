package store

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type StockStore struct {
	rdb *redis.Client
}

func NewStockStore(rdb *redis.Client) *StockStore {
	return &StockStore{
		rdb: rdb,
	}
}

type Stock struct {
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

func (s *StockStore) Create(ctx context.Context, stocks []Stock) error {
	values := make(map[string]interface{}, len(stocks))

	for _, stock := range stocks {
		values[stock.Name] = stock.Quantity
	}

	_, err := s.rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, "stocks")

		if len(values) > 0 {
			pipe.HSet(ctx, "stocks", values)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create table stocks: %w", err)
	}

	return nil
}

func (s *StockStore) ListAll(ctx context.Context) ([]Stock, error) {
	result, err := s.rdb.HGetAll(ctx, "stocks").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stocks: %w", err)
	}

	stocks := make([]Stock, 0, len(result))

	for name, quantity := range result {
		qty, err := strconv.ParseInt(quantity, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse quantity for %s: %w", name, err)
		}

		stocks = append(stocks, Stock{
			Name:     name,
			Quantity: qty,
		})
	}

	return stocks, nil
}
