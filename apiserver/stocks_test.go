package apiserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"stock/store"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestCreateStocksHandler_RequestBodyValidation(t *testing.T) {
	//Arrange
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "invalid JSON",
			body:       `{"stocks":[}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing stocks",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "null stocks",
			body:       `{"stocks":null}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty stock name",
			body:       `{"stocks":[{"name":"", "quantity":10}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "negative quantity",
			body:       `{"stocks":[{"name": "AAPL", "quantity": -10}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "duplicate stocks",
			body:       `{"stocks":[{"name": "AAPL", "quantity": 10}, {"name": "AAPL", "quantity": 5}]}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/stocks", strings.NewReader(tt.body))

			recorder := httptest.NewRecorder()
			server := &ApiServer{}

			//Act
			server.CreateStocksHandler(recorder, req)

			//Assert
			if recorder.Code != tt.wantStatus {
				t.Errorf("expected status 400, got %d", recorder.Code)
			}
		})
	}
}

func TestCreateStocksHandler_Redis(t *testing.T) {
	//Arrange
	tests := []struct {
		name     string
		initial  map[string]string
		body     string
		expected map[string]string
	}{
		{
			name:    "Create stocks",
			initial: nil,
			body: `{
				"stocks": [
					{"name": "AAPL", "quantity":100},
					{"name": "GOOG", "quantity": 50}
				]
			}`,
			expected: map[string]string{
				"AAPL": "100",
				"GOOG": "50",
			},
		},
		{
			name: "Replace stocks",
			initial: map[string]string{
				"AAPL": "100",
				"GOOG": "50",
			},
			body: `{
				"stocks": [
					{"name": "TEST", "quantity": 70}
				]
			}`,
			expected: map[string]string{
				"TEST": "70",
			},
		},
		{
			name: "Clear stocks",
			initial: map[string]string{
				"AAPL": "100",
			},
			body:     `{"stocks": []}`,
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := miniredis.RunT(t)

			rdb := redis.NewClient(&redis.Options{
				Addr: mr.Addr(),
			})
			t.Cleanup(func() {
				_ = rdb.Close()
			})

			ctx := context.Background()

			if len(tt.initial) > 0 {
				if err := rdb.HSet(ctx, "stocks", tt.initial).Err(); err != nil {
					t.Fatal(err)
				}
			}

			server := &ApiServer{
				store: store.New(rdb),
			}

			req := httptest.NewRequest("POST", "/stocks", strings.NewReader(tt.body))
			recorder := httptest.NewRecorder()

			//Act
			server.CreateStocksHandler(recorder, req)

			//Assert - http
			if recorder.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
			}

			//Assert - redis
			stocks, err := rdb.HGetAll(ctx, "stocks").Result()
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(stocks, tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, stocks)
			}

		})
	}
}
