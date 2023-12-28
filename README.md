# Backend Technical Test at Scalingo

## Features

Retrieve information about GitHub repositories (at least 100 different ones from [latest public events](https://api.github.com/events))

Results can be filtered with an [expression language](https://expr-lang.org/docs/language-definition)

## Execution

```
docker compose up
```

docker-compose.yml is meant to use a .env file and to run Application on port `5000`

The GITHUB_ACCESS_TOKEN environment variable is required ([unauthenticated rate limit are low](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28), you can use a [personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens) without any special scopes)

Other environment variable are readed :

- GITHUB_EVENT_API_URL with default "https://api.github.com/events" (can work with the others Event API (like "https://api.github.com/orgs/{org}/events"), they have the same contract)
- GITHUB_EVENT_API_PAGE_SIZE with default 100 (GitHub API allow 100 and default to 30)
- REFRESH with default "5m" : automatic cache refresh delay
- MAX_CALL with default 90 : limit the number of concurrent requests (GitHub API secondary rate limit is 100 concurrent requests)

## Test

```
$ curl localhost:5000/ping
{ "status": "pong" }
```

or

```
$ curl "localhost:5000/repos?filter='Go'%20in%20languages"
{
  "filter": "'Go' in languages",
  "repositories": [
    {
      "description": "Tool integration platform for Kubernetes",
      "forks_count": 409,
      "full_name": "devtron-labs/devtron",
      "languages": {
        "Dockerfile": 9081,
        "Go": 7273855,
        "Makefile": 3105,
        "Mustache": 138820,
        "PLpgSQL": 3538,
        "Python": 1957,
        "Shell": 5010,
        "Smarty": 558,
        "TSQL": 17576
      },
      "license": "apache-2.0",
      "name": "devtron",
      "organization": "devtron-labs",
      "owner": "devtron-labs",
      "topics": [
        "aks",
        "appops",
        "argocd",
        "continuous-deployment",
        "dashboard",
        "deployment",
        "deployment-automation",
        "deployment-pipeline",
        "deployment-strategy",
        "devtron",
        "eks",
        "gitops",
        "gke",
        "hacktoberfest",
        "kubectl",
        "kubernetes",
        "kubernetes-dashboard",
        "kubernetes-deployment",
        "release-automation",
        "workflow-engine"
      ],
      "watchers_count": 3651
    },
    ...
  ]
}
```

## Technical overview

The [limitedconcurrent](https://github.com/dvaumoron/sclng-backend-test-v1/blob/master/limitedconcurrent/limit.go) package isolate the mecanism to dispatch task concurrently with a limited number of working goroutine (ensure the respect of GitHub API concurrent requests limit). 'func(chan<- T)' as task signature allow to handle case with no error and no value to return. Logging is delegated to task, this keep the package independant from any logging library and allows to keep log as specific as needed. However an other design will be required to handle case mixing different kind of value retrieval.

The [repositoryservice](https://github.com/dvaumoron/sclng-backend-test-v1/blob/master/repositoryservice/repository.go) package contains the logic to regularly call GitHub API to retrieve repository information and cache it. The automatic cache refresh strategy allow to always keep good response time, with the downside of sustaining calls even when there is no need. The grouping of behaviour during retrieval with keepField, flattenField and fetchField makes it possible to simplify their updating.

Finally, the [main](main.go) call RepositoryService.List with an optional filtering before returning data in JSON format.
