package apiserver

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"stock/store"
)

type StocksPayload struct {
	Stocks []store.Stock `json:"stocks"`
}

type StocksCreator interface {
	Create(ctx context.Context, stocks []store.Stock) error
}

func (s *ApiServer) CreateStocksHandler(w http.ResponseWriter, r *http.Request) {
	var req StocksPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Stocks == nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	seen := make(map[string]bool, len(req.Stocks))

	for _, stock := range req.Stocks {
		if stock.Name == "" || stock.Quantity < 0 || seen[stock.Name] {
			http.Error(w, "Invalid stocks", http.StatusBadRequest)
			return
		}

		seen[stock.Name] = true
	}

	if err := s.store.Stocks.Create(r.Context(), req.Stocks); err != nil {
		log.Printf("failed to create stocks: %v", err)

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *ApiServer) GetAllStocksHandler(w http.ResponseWriter, r *http.Request) {
	stocks, err := s.store.Stocks.ListAll(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&StocksPayload{Stocks: stocks})
}
