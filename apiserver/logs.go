package apiserver

import (
	"encoding/json"
	"log"
	"net/http"
	"stock/store"
)

type GetLogHandlerResponse struct {
	Log []store.Log `json:"log"`
}

func (s *ApiServer) GetLogHandler(w http.ResponseWriter, r *http.Request) {
	logs, err := s.store.Log.ListAll(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(GetLogHandlerResponse{
		Log: logs,
	}); err != nil {
		log.Printf("failed to encode logs response: %v", err)
	}
}
