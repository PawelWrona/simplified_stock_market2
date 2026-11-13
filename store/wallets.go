package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type WalletStore struct {
	rdb *redis.Client
}

func NewWalletStore(rdb *redis.Client) *WalletStore {
	return &WalletStore{
		rdb: rdb,
	}
}

var ErrWalletNotFound = errors.New("wallet not found")

func (s *WalletStore) Create(ctx context.Context, walletId string, stocks []Stock) error {
	redisKey := fmt.Sprintf("wallet:%s", walletId)

	values := make(map[string]int64, len(stocks))
	for _, stock := range stocks {
		values[stock.Name] = stock.Quantity
	}

	if err := s.rdb.HSet(ctx, redisKey, values).Err(); err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	return nil
}

func (s *WalletStore) ById(ctx context.Context, walletId string) ([]Stock, error) {
	exists, err := s.rdb.SIsMember(ctx, "wallets", walletId).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to check wallet existence: %w", err)
	}
	if !exists {
		return nil, ErrWalletNotFound
	}

	redisKey := fmt.Sprintf("wallet:%s", walletId)

	result, err := s.rdb.HGetAll(ctx, redisKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch wallet by id: %w", err)
	}

	stocks := make([]Stock, 0, len(result))
	for stockName, stockQuantity := range result {
		qty, err := strconv.ParseInt(stockQuantity, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse quantity for %s: %w", stockName, err)
		}

		stocks = append(stocks, Stock{Name: stockName, Quantity: qty})
	}

	return stocks, nil
}

func (s *WalletStore) StockByWalletId(ctx context.Context, walletId string, stockName string) (int, error) {
	walletExists, err := s.rdb.SIsMember(ctx, "wallets", walletId).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to check wallet existence: %w", err)
	}
	if !walletExists {
		return 0, ErrWalletNotFound
	}

	redisKey := fmt.Sprintf("wallet:%s", walletId)

	stockQty, err := s.rdb.HGet(ctx, redisKey, stockName).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}

		return 0, fmt.Errorf("failed to fetch wallet by id: %w", err)
	}

	qty, err := strconv.Atoi(stockQty)
	if err != nil {
		return 0, fmt.Errorf("failed to parse quantity for stockname %s: %w", stockName, err)
	}

	return qty, nil
}
