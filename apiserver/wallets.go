package apiserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"stock/store"
)

type GetWalletByIdResponse struct {
	Id     string        `json:"id"`
	Stocks []store.Stock `json:"stocks"`
}

func (s *ApiServer) GetWalletByIdHandler(w http.ResponseWriter, r *http.Request) {
	walletId := r.PathValue("wallet_id")
	if len(walletId) == 0 {
		http.Error(w, "Incorrect wallet id", http.StatusNotFound)
		return
	}

	stocks, err := s.store.Wallets.ById(r.Context(), walletId)
	if err != nil {
		if errors.Is(err, store.ErrWalletNotFound) {
			http.Error(w, "Wallet not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := &GetWalletByIdResponse{
		Id:     walletId,
		Stocks: stocks,
	}

	data, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *ApiServer) GetStockQuantityByWalletIdHandler(w http.ResponseWriter, r *http.Request) {
	walletId := r.PathValue("wallet_id")
	stockName := r.PathValue("stock_name")
	if len(walletId) == 0 {
		http.Error(w, "Incorrect wallet id", http.StatusNotFound)
		return
	}
	if len(stockName) == 0 {
		http.Error(w, "Incorrect stock name", http.StatusNotFound)
		return
	}

	stockQty, err := s.store.Wallets.StockByWalletId(r.Context(), walletId, stockName)
	if err != nil {
		if errors.Is(err, store.ErrWalletNotFound) {
			http.Error(w, "Wallet not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%d", stockQty)
}

type TradeType string

const (
	Buy  TradeType = "buy"
	Sell TradeType = "sell"
)

type TradeHandlerRequest struct {
	Type TradeType `json:"type"`
}

func (s *ApiServer) TradeHandler(w http.ResponseWriter, r *http.Request) {
	var req TradeHandlerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Type != Buy && req.Type != Sell {
		http.Error(w, "Invalid trade type", http.StatusBadRequest)
		return
	}

	walletId := r.PathValue("wallet_id")
	if len(walletId) == 0 {
		http.Error(w, "Incorrect wallet id", http.StatusNotFound)
		return
	}

	stockName := r.PathValue("stock_name")
	if len(stockName) == 0 {
		http.Error(w, "Incorrect stock name", http.StatusNotFound)
		return
	}

	if err := s.store.Wallets.Trade(r.Context(), walletId, stockName, string(req.Type)); err != nil {
		switch {
		case errors.Is(err, store.ErrStockNotFound):
			http.Error(w, "Stock not found", http.StatusNotFound)
			return
		case errors.Is(err, store.ErrInsufficientBankStocks):
			http.Error(w, "Insufficient bank stocks", http.StatusBadRequest)
			return
		case errors.Is(err, store.ErrInsufficientWalletStocks):
			http.Error(w, "Insufficient wallet stocks", http.StatusBadRequest)
			return
		default:
			log.Printf("Trade failed: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
