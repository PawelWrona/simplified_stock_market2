package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"stock/apiserver"
	"stock/config"
	"stock/database"
	"stock/store"
	"syscall"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	config := config.New()

	rdb, err := database.NewRedisDb()
	if err != nil {
		return err
	}
	defer rdb.Close()

	dataStore := store.New(rdb)

	apiserver := apiserver.New(dataStore, config)

	if err := apiserver.Start(ctx); err != nil {
		return err
	}

	return nil
}
