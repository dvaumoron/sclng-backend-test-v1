package repositoryservice

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dvaumoron/sclng-backend-test-v1/limitedconcurrent"
	"github.com/sirupsen/logrus"
)

type empty = struct{}
type JsonObject = map[string]any

type RepositoryService <-chan []JsonObject

var (
	marker = empty{}

	keepField = map[string]empty{
		"name": marker, "full_name": marker, "description": marker, "forks_count": marker, "watchers_count": marker, "topics": marker,
	}

	flattenField = map[string]string{
		"owner": "login", "licenses": "key", "organization": "login",
	}

	fetchField = map[string]string{
		"languages_url": "languages",
	}

	cleanedSize = len(keepField) + len(flattenField) + len(fetchField)
)

func Make(log logrus.FieldLogger, eventApiUrl string, eventPageSize int, refresh time.Duration, maxCall int, accessToken string) RepositoryService {
	var urlBuilder strings.Builder
	urlBuilder.WriteString(eventApiUrl)
	urlBuilder.WriteString("?per_page=")
	urlBuilder.WriteString(strconv.Itoa(eventPageSize))
	urlBuilder.WriteString("&page=")

	var authorizationBuilder strings.Builder
	authorizationBuilder.WriteString("Bearer ")
	authorizationBuilder.WriteString(accessToken)

	repositoriesChan := make(chan []JsonObject)
	go manageUpdate(log, repositoriesChan, urlBuilder.String(), refresh, maxCall, authorizationBuilder.String())
	return repositoriesChan
}

// ! no defensive copy of cached value
func (rs RepositoryService) List() []JsonObject {
	return <-rs
}

func manageUpdate(log logrus.FieldLogger, repositoriesChan chan<- []JsonObject, eventPageUrl string, refresh time.Duration, maxCall int, authorizationHeader string) {
	repositoriesCache := retrieveRepositoriesData(log, eventPageUrl, maxCall, authorizationHeader)
	repositoriesUpdateChan := make(chan []JsonObject)
	// assumes update time is shorter than refresh tick
	go updateCache(log, repositoriesUpdateChan, eventPageUrl, refresh, maxCall, authorizationHeader)
	for {
		select {
		case repositoriesChan <- repositoriesCache:
		case repositoriesCache = <-repositoriesUpdateChan:
		}
	}
}

func retrieveRepositoriesData(log logrus.FieldLogger, eventPageUrl string, maxCall int, authorizationHeader string) []JsonObject {
	urls := make(map[string]empty, 100)
	for i := 1; len(urls) < 100; i++ {
		extractRepositoriesUrl(log, urls, eventPageUrl, authorizationHeader, i)
	}

	retrievers := make([]func(chan<- JsonObject), 0, len(urls))
	for url := range urls {
		urlCopy := url // avoid closure capture
		retrievers = append(retrievers, func(repositoryChan chan<- JsonObject) {
			retrieveRepositoryData(log, repositoryChan, urlCopy, authorizationHeader)
		})
	}
	return limitedconcurrent.LaunchLimited(retrievers, maxCall)
}

func updateCache(log logrus.FieldLogger, repositoriesUpdateChan chan<- []JsonObject, eventPageUrl string, refresh time.Duration, maxCall int, authorizationHeader string) {
	for range time.Tick(refresh) {
		if data := retrieveRepositoriesData(log, eventPageUrl, maxCall, authorizationHeader); data != nil {
			repositoriesUpdateChan <- data
		}
	}
}

func extractRepositoriesUrl(log logrus.FieldLogger, urls map[string]empty, eventPageUrl string, authorizationHeader string, page int) {
	var urlBuilder strings.Builder
	urlBuilder.WriteString(eventPageUrl)
	urlBuilder.WriteString(strconv.Itoa(page))

	data := githubApiGetRequest(log, urlBuilder.String(), authorizationHeader)

	var events []JsonObject
	if err := json.Unmarshal(data, &events); err != nil {
		log.WithError(err).Error("Fail to parse event api response")
		return
	}

	for _, event := range events {
		if repo, ok := event["repo"].(JsonObject); ok {
			if repoUrl, _ := repo["url"].(string); repoUrl != "" {
				urls[repoUrl] = marker
			}
		}
	}
}

func retrieveRepositoryData(log logrus.FieldLogger, repositoryChan chan<- JsonObject, repositoryUrl string, authorizationHeader string) {
	repositoryData := githubApiGetRequest(log, repositoryUrl, authorizationHeader)
	if len(repositoryData) == 0 {
		return
	}

	var repository JsonObject
	err := json.Unmarshal(repositoryData, &repository)
	if err != nil {
		log.WithError(err).Error("Fail to parse repository api response")
		return
	}

	cleanedRepository := make(JsonObject, cleanedSize)
	for key, value := range repository {
		if _, ok := keepField[key]; ok {
			cleanedRepository[key] = value
			continue
		}

		if subKey, ok := flattenField[key]; ok {
			if castedValue, okCast := value.(JsonObject); okCast {
				cleanedRepository[key] = castedValue[subKey]
				continue
			}
			log.WithField("flattenField", key).Error("Unable to flatten : can not cast to JsonObject")
			return
		}

		newKey, ok := fetchField[key]
		if !ok {
			continue
		}

		url, _ := value.(string)
		if url == "" {
			log.WithField(key, value).Error("Unable to fetch url : empty or non string")
			return
		}

		data := githubApiGetRequest(log, url, authorizationHeader)
		if len(data) == 0 {
			return
		}

		var parsed any
		if err = json.Unmarshal(data, &parsed); err != nil {
			log.WithError(err).Error("Fail to parse fetched response")
			return
		}
		cleanedRepository[newKey] = parsed
	}

	repositoryChan <- cleanedRepository
}

func githubApiGetRequest(log logrus.FieldLogger, callUrl string, authorizationHeader string) []byte {
	request, err := http.NewRequest(http.MethodGet, callUrl, nil)
	if err != nil {
		log.WithError(err).Error("Fail to create api request")
		return nil
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", authorizationHeader)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.WithError(err).Error("Fail during api request")
		return nil
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		log.WithError(err).Error("Fail to read api response")
		return nil
	}
	return data
}
