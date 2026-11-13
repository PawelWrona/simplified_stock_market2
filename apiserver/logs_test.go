package apiserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"stock/store"
	"testing"
)

func TestGetLogHandler(t *testing.T) {
	tests := []struct {
		name             string
		initialLogs      []string
		expectedLogs     []store.Log
		expectedStatus   int
		redisUnavailable bool
	}{
		{
			name:           "Empty log",
			initialLogs:    []string{},
			expectedLogs:   []store.Log{},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Single log entry",
			initialLogs: []string{
				`{"type":"buy","wallet_id":"123","stock_name":"AAPL"}`,
			},
			expectedLogs: []store.Log{
				{
					Type:      "buy",
					WalletId:  "123",
					StockName: "AAPL",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Multiple log entries",
			initialLogs: []string{
				`{"type":"buy","wallet_id":"123","stock_name":"AAPL"}`,
				`{"type":"sell","wallet_id":"124","stock_name":"GOOG"}`,
				`{"type":"buy","wallet_id":"123","stock_name":"AAPL"}`,
			},
			expectedLogs: []store.Log{
				{
					Type:      "buy",
					WalletId:  "123",
					StockName: "AAPL",
				},
				{
					Type:      "sell",
					WalletId:  "124",
					StockName: "GOOG",
				},
				{
					Type:      "buy",
					WalletId:  "123",
					StockName: "AAPL",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid log entry",
			initialLogs: []string{
				`{"type":"buy","wallet_id":"123","stock_name":`,
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "Redis unavailable",
			initialLogs: []string{
				`{"type":"buy","wallet_id":"123","stock_name":"AAPL"}`,
			},
			expectedLogs: []store.Log{
				{
					Type:      "buy",
					WalletId:  "123",
					StockName: "AAPL",
				},
			},
			expectedStatus:   http.StatusInternalServerError,
			redisUnavailable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, rdb, mr := setupTestServer(t)

			ctx := context.Background()

			for _, entry := range tt.initialLogs {
				if err := rdb.RPush(ctx, "log", entry).Err(); err != nil {
					t.Fatal(err)
				}
			}

			if tt.redisUnavailable {
				mr.Close()
			}

			req := httptest.NewRequest("GET", "/log", nil)
			recorder := httptest.NewRecorder()

			server.GetLogHandler(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.expectedStatus, recorder.Code, recorder.Body.String())
			}

			if tt.expectedStatus != http.StatusOK {
				return
			}

			var response GetLogHandlerResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("invalid JSON response: %v", err)
			}

			if !reflect.DeepEqual(response.Log, tt.expectedLogs) {
				t.Errorf("expected logs %+v, got %+v", tt.expectedLogs, response.Log)
			}
		})
	}
}
