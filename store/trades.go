package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var tradeScript = redis.NewScript(`
	local bankKey = KEYS[1]
	local walletKey = KEYS[2]
	local walletsKey = KEYS[3]
	local logKey = KEYS[4]

	local walletId = ARGV[1]
	local stockName = ARGV[2]
	local tradeType = ARGV[3]
	local logEntry = ARGV[4]

	local function checkType(key, expected)
    local actual = redis.call("TYPE", key).ok
    return actual == "none" or actual == expected
	end

	if not checkType(bankKey, "hash") then
		return redis.error_reply("invalid bank key type")
	end

	if not checkType(walletKey, "hash") then
		return redis.error_reply("invalid wallet key type")
	end

	if not checkType(walletsKey, "set") then
		return redis.error_reply("invalid wallets key type")
	end

	if not checkType(logKey, "list") then
		return redis.error_reply("invalid log key type")
	end

	local bankValue = redis.call("HGET", bankKey, stockName)
	if not bankValue then
		return 1
	end

	local bankQty = tonumber(bankValue)
	if not bankQty then
		return redis.error_reply("invalid bank quantity")
	end

	if tradeType == "buy" then
		if bankQty < 1 then
			return 2
		end

		redis.call("HINCRBY", bankKey, stockName, -1)
		redis.call("HINCRBY", walletKey, stockName, 1)
	elseif tradeType == "sell" then
		local walletStockValue = redis.call("HGET", walletKey, stockName)

		local walletStockQty = tonumber(walletStockValue or "0")
		if not walletStockQty then
			return redis.error_reply("invalid wallet quantity")
		end

		if walletStockQty < 1 then
			return 3
		end

		redis.call("HINCRBY", walletKey, stockName, -1)
		redis.call("HINCRBY", bankKey, stockName, 1)
	else
		return redis.error_reply("invalid trade type")
	end

	redis.call("SADD", walletsKey, walletId)
	redis.call("RPUSH", logKey, logEntry)

	return 0
`)

var (
	ErrStockNotFound            = errors.New("stock not found")
	ErrInsufficientBankStocks   = errors.New("insufficient bank stocks")
	ErrInsufficientWalletStocks = errors.New("insufficient wallet stocks")
)

type TradeLog struct {
	Type      string `json:"type"`
	WalletId  string `json:"wallet_id"`
	StockName string `json:"stock_name"`
}

func (s *WalletStore) Trade(ctx context.Context, walletId string, stockName string, tradeType string) error {
	if tradeType != "buy" && tradeType != "sell" {
		return fmt.Errorf("invalid trade type: %q", tradeType)
	}

	entry, err := json.Marshal(TradeLog{
		Type:      tradeType,
		WalletId:  walletId,
		StockName: stockName,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal trade log: %w", err)
	}

	keys := []string{"stocks", "wallet:" + walletId, "wallets", "log"}

	result, err := tradeScript.Run(ctx, s.rdb, keys, walletId, stockName, tradeType, string(entry)).Int()
	if err != nil {
		return fmt.Errorf("failed to execute trade: %w", err)
	}

	switch result {
	case 0:
		return nil
	case 1:
		return ErrStockNotFound
	case 2:
		return ErrInsufficientBankStocks
	case 3:
		return ErrInsufficientWalletStocks
	default:
		err := fmt.Errorf("unexpected trade result: %d", result)
		fmt.Println(err)
		return err
	}
}
