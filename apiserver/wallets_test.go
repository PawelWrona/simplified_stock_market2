package apiserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestGetWalletByIdHandler(t *testing.T) {
	//Arrange
	tests := []struct {
		name             string
		walletId         string
		walletExists     bool
		initialStocks    map[string]int64
		expectedStatus   int
		expectedStocks   map[string]int64
		invalidQuantity  bool
		redisUnavailable bool
		emptyWalletId    bool
	}{
		{
			name:         "existing wallet with stocks",
			walletId:     "123",
			walletExists: true,
			initialStocks: map[string]int64{
				"AAPL": 20,
				"GOOG": 50,
			},
			expectedStatus: http.StatusOK,
			expectedStocks: map[string]int64{
				"AAPL": 20,
				"GOOG": 50,
			},
		},
		{
			name:           "wallet not found",
			walletId:       "999",
			walletExists:   false,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "existing empty wallet",
			walletId:       "456",
			walletExists:   true,
			expectedStatus: http.StatusOK,
			expectedStocks: map[string]int64{},
		},
		{
			name:            "invalid stock quantity",
			walletId:        "123",
			walletExists:    true,
			invalidQuantity: true,
			expectedStatus:  http.StatusInternalServerError,
		},
		{
			name:             "redis unavailable",
			walletId:         "123",
			walletExists:     true,
			redisUnavailable: true,
			expectedStatus:   http.StatusInternalServerError,
		},
		{
			name:           "empty wallet ID",
			walletId:       "",
			emptyWalletId:  true,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, rdb, mr := setupTestServer(t)

			ctx := context.Background()

			if tt.walletExists {
				if err := rdb.SAdd(ctx, "wallets", tt.walletId).Err(); err != nil {
					t.Fatal(err)
				}
			}

			if tt.invalidQuantity {
				if err := rdb.HSet(ctx, "wallet:"+tt.walletId, "AAPL", "abc").Err(); err != nil {
					t.Fatal(err)
				}
			}

			if len(tt.initialStocks) > 0 {
				values := make(map[string]interface{}, len(tt.initialStocks))

				for name, quantity := range tt.initialStocks {
					values[name] = quantity
				}

				if err := rdb.HSet(ctx, "wallet:"+tt.walletId, values).Err(); err != nil {
					t.Fatal(err)
				}
			}

			if tt.redisUnavailable {
				mr.Close()
			}

			req := httptest.NewRequest("GET", "/wallets/"+tt.walletId, nil)
			if !tt.emptyWalletId {
				req.SetPathValue("wallet_id", tt.walletId)
			}
			recorder := httptest.NewRecorder()

			//Act
			server.GetWalletByIdHandler(recorder, req)

			//Assert - HTTP status
			if recorder.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.expectedStatus, recorder.Code, recorder.Body.String())
			}

			if tt.expectedStatus != http.StatusOK {
				return
			}

			contentType := recorder.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected application/json, got %s", contentType)
			}

			//Assert - response body
			var response GetWalletByIdResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("invalid JSON response: %v", err)
			}

			if response.Id != tt.walletId {
				t.Errorf("expected wallet ID %s, got %s", tt.walletId, response.Id)
			}

			if response.Stocks == nil {
				t.Fatal("expected non-nil stocks slice")
			}

			got := make(map[string]int64, len(response.Stocks))
			for _, stock := range response.Stocks {
				got[stock.Name] = stock.Quantity
			}

			if len(response.Stocks) != len(tt.expectedStocks) {
				t.Errorf("expected %d stocks, got %d", len(tt.expectedStocks), len(response.Stocks))
			}

			if !reflect.DeepEqual(got, tt.expectedStocks) {
				t.Errorf("expected stocks %v, got %v", tt.expectedStocks, got)
			}
		})
	}
}

func TestGetStockQuantityByWalletIdHandler(t *testing.T) {
	//Arrange
	tests := []struct {
		name             string
		walletId         string
		walletExists     bool
		stockName        string
		stockNameExists  bool
		initialStockName string
		initialQuantity  int64
		expectedQuantity int64
		expectedStatus   int
		redisUnavailable bool
	}{
		{
			name:             "wallet and stock exist",
			walletId:         "123",
			walletExists:     true,
			stockName:        "AAPL",
			stockNameExists:  true,
			initialQuantity:  20,
			expectedQuantity: 20,
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "wallet exists but is empty, stock does not exist",
			walletId:         "123",
			walletExists:     true,
			stockName:        "AAPL",
			stockNameExists:  false,
			expectedQuantity: 0,
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "wallet has GOOG but requested AAPL does not exist",
			walletId:         "123",
			walletExists:     true,
			stockName:        "AAPL",
			stockNameExists:  true,
			initialStockName: "GOOG",
			initialQuantity:  50,
			expectedQuantity: 0,
			expectedStatus:   http.StatusOK,
		},
		{
			name:           "empty wallet ID",
			stockName:      "AAPL",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "empty stock name",
			walletId:       "123",
			walletExists:   true,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "wallet does not exist",
			walletId:       "123",
			stockName:      "AAPL",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:             "stock quantity equals 0",
			walletId:         "123",
			walletExists:     true,
			stockName:        "AAPL",
			stockNameExists:  true,
			initialQuantity:  0,
			expectedQuantity: 0,
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "stock quantity value is very high",
			walletId:         "123",
			walletExists:     true,
			stockName:        "AAPL",
			stockNameExists:  true,
			initialQuantity:  3000000000,
			expectedQuantity: 3000000000,
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "redis unavailable",
			walletId:         "123",
			walletExists:     true,
			stockName:        "AAPL",
			redisUnavailable: true,
			expectedStatus:   http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, rdb, mr := setupTestServer(t)

			ctx := context.Background()

			if tt.walletExists {
				if err := rdb.SAdd(ctx, "wallets", tt.walletId).Err(); err != nil {
					t.Fatal(err)
				}
			}

			if tt.stockNameExists {
				name := tt.stockName
				if tt.initialStockName != "" {
					name = tt.initialStockName
				}

				if err := rdb.HSet(ctx, "wallet:"+tt.walletId, name, tt.initialQuantity).Err(); err != nil {
					t.Fatal(err)
				}
			}

			if tt.redisUnavailable {
				mr.Close()
			}

			req := httptest.NewRequest("GET", "/wallets/"+tt.walletId+"/stocks/"+tt.stockName, nil)
			req.SetPathValue("wallet_id", tt.walletId)
			req.SetPathValue("stock_name", tt.stockName)
			recorder := httptest.NewRecorder()

			//Act
			server.GetStockQuantityByWalletIdHandler(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.expectedStatus, recorder.Code, recorder.Body.String())
			}

			if tt.expectedStatus != http.StatusOK {
				return
			}

			contentType := recorder.Header().Get("Content-Type")
			if contentType != "text/plain" {
				t.Errorf("expected text/plain, got %s", contentType)
			}

			//Assert
			got := recorder.Body.String()
			qty, err := strconv.ParseInt(got, 10, 64)
			if err != nil {
				t.Fatalf("expected response to be parsable into int64, got: %s", got)
			}

			if qty != tt.expectedQuantity {
				t.Errorf("expected quantity %d, got %d", tt.expectedQuantity, qty)
			}
		})
	}
}

func TestTradeHandler_Success(t *testing.T) {
	//Arrange
	tests := []struct {
		name                  string
		body                  string
		initialBankStocks     map[string]string
		walletExistsInitially bool
		walletId              string
		stockName             string
		initialWalletStocks   map[string]string
		expectedBankStocks    map[string]string
		expectedWalletStocks  map[string]string
	}{
		{
			name:                  "Buy with creation of new wallet",
			body:                  `{"type": "buy"}`,
			initialBankStocks:     map[string]string{"AAPL": "10"},
			walletExistsInitially: false,
			walletId:              "123",
			stockName:             "AAPL",
			expectedBankStocks:    map[string]string{"AAPL": "9"},
			expectedWalletStocks:  map[string]string{"AAPL": "1"},
		},
		{
			name:                  "Buy with existing wallet",
			body:                  `{"type": "buy"}`,
			initialBankStocks:     map[string]string{"AAPL": "10"},
			walletExistsInitially: true,
			walletId:              "123",
			stockName:             "AAPL",
			initialWalletStocks:   map[string]string{"AAPL": "3"},
			expectedBankStocks:    map[string]string{"AAPL": "9"},
			expectedWalletStocks:  map[string]string{"AAPL": "4"},
		},
		{
			name:                  "Buy with existing, empty wallet",
			body:                  `{"type": "buy"}`,
			initialBankStocks:     map[string]string{"AAPL": "10"},
			walletExistsInitially: true,
			walletId:              "123",
			stockName:             "AAPL",
			initialWalletStocks:   map[string]string{},
			expectedBankStocks:    map[string]string{"AAPL": "9"},
			expectedWalletStocks:  map[string]string{"AAPL": "1"},
		},
		{
			name:                  "Buy new stock with existing, not empty wallet",
			body:                  `{"type": "buy"}`,
			initialBankStocks:     map[string]string{"AAPL": "10"},
			walletExistsInitially: true,
			walletId:              "123",
			stockName:             "AAPL",
			initialWalletStocks:   map[string]string{"GOOG": "5"},
			expectedBankStocks:    map[string]string{"AAPL": "9"},
			expectedWalletStocks:  map[string]string{"GOOG": "5", "AAPL": "1"},
		},
		{
			name:                  "Sell stock",
			body:                  `{"type": "sell"}`,
			initialBankStocks:     map[string]string{"AAPL": "10"},
			walletExistsInitially: true,
			walletId:              "123",
			stockName:             "AAPL",
			initialWalletStocks:   map[string]string{"AAPL": "5"},
			expectedBankStocks:    map[string]string{"AAPL": "11"},
			expectedWalletStocks:  map[string]string{"AAPL": "4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, rdb, _ := setupTestServer(t)

			ctx := context.Background()

			if tt.walletExistsInitially {
				if err := rdb.SAdd(ctx, "wallets", tt.walletId).Err(); err != nil {
					t.Fatal(err)
				}

				if len(tt.initialWalletStocks) > 0 {
					if err := rdb.HSet(ctx, "wallet:"+tt.walletId, tt.initialWalletStocks).Err(); err != nil {
						t.Fatal(err)
					}
				}
			}

			if err := rdb.HSet(ctx, "stocks", tt.initialBankStocks).Err(); err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest("POST", "/wallets/"+tt.walletId+"/stocks/"+tt.stockName, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("wallet_id", tt.walletId)
			req.SetPathValue("stock_name", tt.stockName)
			recorder := httptest.NewRecorder()

			//Act
			server.TradeHandler(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
			}

			stocksResult, err := rdb.HGetAll(ctx, "stocks").Result()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(stocksResult, tt.expectedBankStocks) {
				t.Fatalf("expected %v, got %v", tt.expectedBankStocks, stocksResult)
			}

			walletResult, err := rdb.HGetAll(ctx, "wallet:"+tt.walletId).Result()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(walletResult, tt.expectedWalletStocks) {
				t.Fatalf("expected %v, got %v", tt.expectedWalletStocks, walletResult)
			}

			if !tt.walletExistsInitially {
				exists, err := rdb.SIsMember(ctx, "wallets", tt.walletId).Result()
				if err != nil {
					t.Fatal(err)
				}

				if !exists {
					t.Fatalf("wallet does not exist in wallets set")
				}
			}
		})
	}
}

func TestTradeHandler_Errors(t *testing.T) {
	tests := []struct {
		name                  string
		body                  string
		bankExistsInitially   bool
		initialBankStocks     map[string]string
		walletExistsInitially bool
		walletId              string
		stockName             string
		initialWalletStocks   map[string]string
		expectedStatus        int
	}{
		{
			name:                  "Buy non-existing stock",
			body:                  `{"type": "buy"}`,
			bankExistsInitially:   true,
			initialBankStocks:     map[string]string{"AAPL": "10"},
			walletExistsInitially: true,
			walletId:              "123",
			stockName:             "TSLA",
			initialWalletStocks:   map[string]string{"AAPL": "5"},
			expectedStatus:        http.StatusNotFound,
		},
		{
			name:                  "Sell non-existing stock",
			body:                  `{"type": "sell"}`,
			bankExistsInitially:   true,
			initialBankStocks:     map[string]string{"AAPL": "10"},
			walletExistsInitially: true,
			walletId:              "123",
			stockName:             "TSLA",
			initialWalletStocks:   map[string]string{"TSLA": "5"},
			expectedStatus:        http.StatusNotFound,
		},
		{
			name:                  "Buy with insufficient bank stocks",
			body:                  `{"type": "buy"}`,
			bankExistsInitially:   true,
			initialBankStocks:     map[string]string{"AAPL": "0"},
			walletExistsInitially: true,
			walletId:              "123",
			stockName:             "AAPL",
			initialWalletStocks:   map[string]string{"AAPL": "1"},
			expectedStatus:        http.StatusBadRequest,
		},
		{
			name:                  "Sell with insufficient wallet stocks",
			body:                  `{"type": "sell"}`,
			bankExistsInitially:   true,
			initialBankStocks:     map[string]string{"AAPL": "1"},
			walletExistsInitially: true,
			walletId:              "123",
			stockName:             "AAPL",
			initialWalletStocks:   map[string]string{"AAPL": "0"},
			expectedStatus:        http.StatusBadRequest,
		},
		{
			name:                  "Sell stock not owned by wallet",
			body:                  `{"type": "sell"}`,
			bankExistsInitially:   true,
			initialBankStocks:     map[string]string{"AAPL": "5"},
			walletExistsInitially: true,
			walletId:              "123",
			stockName:             "AAPL",
			initialWalletStocks:   map[string]string{"GOOG": "1"},
			expectedStatus:        http.StatusBadRequest,
		},
		{
			name:                  "Sell from non existing wallet",
			body:                  `{"type": "sell"}`,
			bankExistsInitially:   true,
			initialBankStocks:     map[string]string{"AAPL": "5"},
			walletExistsInitially: false,
			walletId:              "123",
			stockName:             "AAPL",
			expectedStatus:        http.StatusBadRequest,
		},
		{
			name:                  "Invalid trade type",
			body:                  `{"type": "test"}`,
			bankExistsInitially:   true,
			initialBankStocks:     map[string]string{"AAPL": "5"},
			walletExistsInitially: true,
			walletId:              "123",
			stockName:             "AAPL",
			initialWalletStocks:   map[string]string{"AAPL": "5"},
			expectedStatus:        http.StatusBadRequest,
		},
		{
			name:                  "Missing trade type",
			body:                  `{}`,
			bankExistsInitially:   true,
			initialBankStocks:     map[string]string{"AAPL": "5"},
			walletExistsInitially: true,
			walletId:              "123",
			stockName:             "AAPL",
			initialWalletStocks:   map[string]string{"AAPL": "5"},
			expectedStatus:        http.StatusBadRequest,
		},
		{
			name:                  "Invalid request body json",
			body:                  `{"type": `,
			bankExistsInitially:   true,
			initialBankStocks:     map[string]string{"AAPL": "5"},
			walletExistsInitially: true,
			walletId:              "123",
			stockName:             "AAPL",
			initialWalletStocks:   map[string]string{"AAPL": "5"},
			expectedStatus:        http.StatusBadRequest,
		},
		{
			name:                  "Empty request body",
			body:                  ``,
			bankExistsInitially:   true,
			initialBankStocks:     map[string]string{"AAPL": "5"},
			walletExistsInitially: true,
			walletId:              "123",
			stockName:             "AAPL",
			initialWalletStocks:   map[string]string{"AAPL": "5"},
			expectedStatus:        http.StatusBadRequest,
		},
		{
			name:           "Trade type with uppercase letters",
			body:           `{"type":"BUY"}`,
			walletId:       "123",
			stockName:      "AAPL",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Trade type is a number",
			body:           `{"type":123}`,
			walletId:       "123",
			stockName:      "AAPL",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Trade type is null",
			body:           `{"type":null}`,
			walletId:       "123",
			stockName:      "AAPL",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty wallet ID",
			body:           `{"type":"buy"}`,
			walletId:       "",
			stockName:      "AAPL",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Empty stock name",
			body:           `{"type":"buy"}`,
			walletId:       "123",
			stockName:      "",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, rdb, _ := setupTestServer(t)

			ctx := context.Background()

			if tt.walletExistsInitially {
				if err := rdb.SAdd(ctx, "wallets", tt.walletId).Err(); err != nil {
					t.Fatal(err)
				}

				if len(tt.initialWalletStocks) > 0 {
					if err := rdb.HSet(ctx, "wallet:"+tt.walletId, tt.initialWalletStocks).Err(); err != nil {
						t.Fatal(err)
					}
				}
			}

			if tt.bankExistsInitially && (len(tt.initialBankStocks) > 0) {
				if err := rdb.HSet(ctx, "stocks", tt.initialBankStocks).Err(); err != nil {
					t.Fatal(err)
				}
			}

			req := httptest.NewRequest("POST", "/wallets/"+tt.walletId+"/stocks/"+tt.stockName, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("wallet_id", tt.walletId)
			req.SetPathValue("stock_name", tt.stockName)
			recorder := httptest.NewRecorder()

			//Act
			server.TradeHandler(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.expectedStatus, recorder.Code, recorder.Body.String())
			}

			bankResult, err := rdb.HGetAll(ctx, "stocks").Result()
			if err != nil {
				t.Fatal(err)
			}

			expectedBank := tt.initialBankStocks
			if expectedBank == nil {
				expectedBank = map[string]string{}
			}

			if !reflect.DeepEqual(bankResult, expectedBank) {
				t.Errorf("bank state changed: expected %v, got %v", expectedBank, bankResult)
			}

			walletResult, err := rdb.HGetAll(ctx, "wallet:"+tt.walletId).Result()
			if err != nil {
				t.Fatal(err)
			}

			expectedWallet := tt.initialWalletStocks
			if expectedWallet == nil {
				expectedWallet = map[string]string{}
			}

			if !reflect.DeepEqual(walletResult, expectedWallet) {
				t.Errorf("wallet state changed: expected %v, got %v",
					expectedWallet, walletResult)
			}

			exists, err := rdb.SIsMember(ctx, "wallets", tt.walletId).Result()
			if err != nil {
				t.Fatal(err)
			}
			if exists != tt.walletExistsInitially {
				t.Errorf("expected wallet existence %v, got %v", tt.walletExistsInitially, exists)
			}
		})
	}
}
