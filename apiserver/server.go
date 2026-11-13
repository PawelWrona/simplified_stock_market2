package apiserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"stock/config"
	"stock/store"
	"time"
)

type ApiServer struct {
	store  *store.Store
	config *config.Config
}

func New(store *store.Store, config *config.Config) *ApiServer {
	return &ApiServer{
		store:  store,
		config: config,
	}
}

func (s *ApiServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /stocks", s.CreateStocksHandler)
	mux.HandleFunc("GET /stocks", s.GetAllStocksHandler)
	mux.HandleFunc("GET /wallets/{wallet_id}", s.GetWalletByIdHandler)
	mux.HandleFunc("POST /wallets/{wallet_id}/stocks/{stock_name}", s.TradeHandler)
	mux.HandleFunc("GET /wallets/{wallet_id}/stocks/{stock_name}", s.GetStockQuantityByWalletIdHandler)
	mux.HandleFunc("GET /log", s.GetLogHandler)
	mux.HandleFunc("POST /chaos", s.ChaosHandler)

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(s.config.Host, s.config.Port),
		Handler: mux,
	}

	serverErr := make(chan error, 1)

	go func() {
		log.Printf("Listening on %s\n", httpServer.Addr)

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("http server failed: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("htttp server shutdown failed: %w", err)
		}

		return nil
	}
}
