package store

import "github.com/redis/go-redis/v9"

type Store struct {
	Stocks  *StockStore
	Wallets *WalletStore
	Log     *LogStore
}

func New(rdb *redis.Client) *Store {
	return &Store{
		Stocks:  NewStockStore(rdb),
		Wallets: NewWalletStore(rdb),
		Log:     NewLogStore(rdb),
	}
}
