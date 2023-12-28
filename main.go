package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Scalingo/go-handlers"
	"github.com/Scalingo/go-utils/logger"
	"github.com/dvaumoron/sclng-backend-test-v1/predicate"
	"github.com/dvaumoron/sclng-backend-test-v1/repositoryservice"
)

const (
	contentType     = "Content-Type"
	jsonContentType = "application/json"

	parseFilterErrorMsg = "can not parse filter"
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
		result := make(map[string]any, 3)

		filters := r.URL.Query()["filter"]
		var filterErrors []string
		switch len(filters) {
		case 0:
			// no filter
		default:
			filterErrors = append(filterErrors, "only first filter is used")
			fallthrough
		case 1:
			result["filter"] = filters[0]
			predicate, err := predicate.ParsePredicate(filters[0])
			if err != nil {
				log.WithError(err).Error(parseFilterErrorMsg)
				filterErrors = append(filterErrors, parseFilterErrorMsg)
				break
			}

			filtered := make([]repositoryservice.JsonObject, 0, len(repositories))
			for _, repository := range repositories {
				if predicate(repository) {
					filtered = append(filtered, repository)
				}
			}
			repositories = filtered
		}

		result["repositories"] = repositories
		if len(filterErrors) != 0 {
			result["filter_errors"] = filterErrors
		}

		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		err := encoder.Encode(result)
		if err != nil {
			log.WithError(err).Error("Fail to encode JSON")
		}
		return nil
	}
}
