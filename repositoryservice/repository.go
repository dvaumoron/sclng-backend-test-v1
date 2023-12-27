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

type Repository struct {
	FullName  string                    `json:"full_name"`
	Owner     string                    `json:"owner"`
	Name      string                    `json:"repository"`
	Languages map[string]map[string]int `json:"languages"`
}

func MakeRepository(fullName string, owner string, name string, languages map[string]int) Repository {
	converted := make(map[string]map[string]int, len(languages))
	for name, bytes := range languages {
		converted[name] = map[string]int{"bytes": bytes}
	}
	return Repository{FullName: fullName, Owner: owner, Name: name, Languages: converted}
}

type RepositoryService <-chan []Repository

func Make(log logrus.FieldLogger, eventApiUrl string, eventPageSize int, refresh time.Duration, maxCall int, accessToken string) RepositoryService {
	var urlBuilder strings.Builder
	urlBuilder.WriteString(eventApiUrl)
	urlBuilder.WriteString("?per_page=")
	urlBuilder.WriteString(strconv.Itoa(eventPageSize))
	urlBuilder.WriteString("&page=")

	var authorizationBuilder strings.Builder
	authorizationBuilder.WriteString("Bearer ")
	authorizationBuilder.WriteString(accessToken)

	repositoriesChan := make(chan []Repository)
	go manageUpdate(log, repositoriesChan, urlBuilder.String(), refresh, maxCall, authorizationBuilder.String())
	return repositoriesChan
}

func (rs RepositoryService) List() []Repository {
	return <-rs
}

func manageUpdate(log logrus.FieldLogger, repositoriesChan chan<- []Repository, eventPageUrl string, refresh time.Duration, maxCall int, authorizationHeader string) {
	repositoriesCache := retrieveRepositoriesData(log, eventPageUrl, maxCall, authorizationHeader)
	repositoriesUpdateChan := make(chan []Repository)
	// assumes update time is shorter than refresh tick
	go updateCache(log, repositoriesUpdateChan, eventPageUrl, refresh, maxCall, authorizationHeader)
	for {
		select {
		case repositoriesChan <- repositoriesCache:
		case repositoriesCache = <-repositoriesUpdateChan:
		}
	}
}

func retrieveRepositoriesData(log logrus.FieldLogger, eventPageUrl string, maxCall int, authorizationHeader string) []Repository {
	urls := make(map[string]empty, 100)
	for i := 1; len(urls) < 100; i++ {
		extractRepositoriesUrl(log, urls, eventPageUrl, authorizationHeader, i)
	}

	retrievers := make([]func(chan<- Repository), len(urls))
	for url := range urls {
		urlCopy := url // avoid closure capture
		retrievers = append(retrievers, func(repositoryChan chan<- Repository) {
			retrieveRepositoryData(log, repositoryChan, urlCopy, authorizationHeader)
		})
	}
	return limitedconcurrent.LaunchLimited(retrievers, maxCall)
}

func updateCache(log logrus.FieldLogger, repositoriesUpdateChan chan<- []Repository, eventPageUrl string, refresh time.Duration, maxCall int, authorizationHeader string) {
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

	var events []map[string]any
	if err := json.Unmarshal(data, &events); err != nil {
		log.WithError(err).Error("Fail to parse event api response")
		return
	}

	for _, event := range events {
		if repo, ok := event["repo"].(map[string]any); ok {
			if repoUrl, _ := repo["url"].(string); repoUrl != "" {
				urls[repoUrl] = empty{}
			}
		}
	}
}

func retrieveRepositoryData(log logrus.FieldLogger, repositoryChan chan<- Repository, repositoryUrl string, authorizationHeader string) {
	repositoryData := githubApiGetRequest(log, repositoryUrl, authorizationHeader)
	if len(repositoryData) == 0 {
		return
	}

	fullName, owner, name := "todo", "todo", "todo"

	languagesUrl := "todo"
	languagesData := githubApiGetRequest(log, languagesUrl, authorizationHeader)
	if len(languagesData) == 0 {
		return
	}

	languages := map[string]int{}
	// TODO
	repositoryChan <- MakeRepository(fullName, owner, name, languages)
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
