package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Scalingo/go-handlers"
	"github.com/Scalingo/go-utils/logger"
	"github.com/dvaumoron/sclng-backend-test-v1/repositoryservice"
)

const (
	contentType     = "Content-Type"
	jsonContentType = "application/json"
)

func main() {
	log := logger.Default()
	log.Info("Initializing app")
	cfg, err := newConfig()
	if err != nil {
		log.WithError(err).Error("Fail to initialize configuration")
		os.Exit(1)
	}

	repoService := repositoryservice.Make(log, cfg.EventApiUrl, cfg.EventPageSize, cfg.Refresh, cfg.MaxCall, cfg.AccessToken)

	log.Info("Initializing routes")
	// Initialize web server and configure /ping and /repos routes
	router := handlers.NewRouter(log)
	router.HandleFunc("/ping", pongHandler)
	router.HandleFunc("/repos", makeReposHandler(repoService))

	log = log.WithField("port", cfg.Port)
	log.Info("Listening...")
	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), router)
	if err != nil {
		log.WithError(err).Error("Fail to listen to the given port")
		os.Exit(2)
	}
}

func pongHandler(w http.ResponseWriter, r *http.Request, _ map[string]string) error {
	log := logger.Get(r.Context())
	w.Header().Add(contentType, jsonContentType)
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(map[string]string{"status": "pong"})
	if err != nil {
		log.WithError(err).Error("Fail to encode JSON")
	}
	return nil
}

func makeReposHandler(repoService repositoryservice.RepositoryService) func(w http.ResponseWriter, r *http.Request, _ map[string]string) error {
	return func(w http.ResponseWriter, r *http.Request, _ map[string]string) error {
		log := logger.Get(r.Context())
		w.Header().Add(contentType, jsonContentType)
		w.WriteHeader(http.StatusOK)

		repositories := repoService.List()

		// TODO filter

		err := json.NewEncoder(w).Encode(map[string]any{"repositories": repositories})
		if err != nil {
			log.WithError(err).Error("Fail to encode JSON")
		}
		return nil
	}
}
