package apiserver

import (
	"net/http"
	"os"
)

func (s *ApiServer) ChaosHandler(w http.ResponseWriter, r *http.Request) {
	os.Exit(1)
}
