package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type LogStore struct {
	rdb *redis.Client
}

func NewLogStore(rdb *redis.Client) *LogStore {
	return &LogStore{
		rdb: rdb,
	}
}

type Log struct {
	Type      string `json:"type"`
	WalletId  string `json:"wallet_id"`
	StockName string `json:"stock_name"`
}

func (s *LogStore) ListAll(ctx context.Context) ([]Log, error) {
	result, err := s.rdb.LRange(ctx, "log", 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}

	logs := make([]Log, 0, len(result))
	for _, entry := range result {
		var log Log

		if err := json.Unmarshal([]byte(entry), &log); err != nil {
			return nil, fmt.Errorf("failed to decode log: %w", err)
		}

		logs = append(logs, log)
	}

	return logs, nil
}
