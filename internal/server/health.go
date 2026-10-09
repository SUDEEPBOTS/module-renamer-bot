package server

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

type HealthServer struct {
	server *http.Server
	port   string
}

func NewHealthServer() *HealthServer {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Server", "SUDEEPBOTS-HealthEngine/1.0")
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodGet {
			_, _ = fmt.Fprintf(w, `{"status":"ok","service":"module-renamer-bot","timestamp":%d}`, time.Now().Unix())
		}
	}

	mux.HandleFunc("/", handler)
	mux.HandleFunc("/health", handler)
	mux.HandleFunc("/ping", handler)

	return &HealthServer{
		port: port,
		server: &http.Server{
			Addr:         ":" + port,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
	}
}

func (h *HealthServer) Start() {
	go func() {
		_ = h.server.ListenAndServe()
	}()
}

func (h *HealthServer) Stop() {
	if h.server != nil {
		_ = h.server.Close()
	}
}
