# Backend Technical Test at Scalingo

## Features

Retrieve information about GitHub repositories (at least 100 different ones from [latest public events](https://api.github.com/events))

Results can be filtered with an [expression language](https://expr-lang.org/docs/language-definition)

## Execution

docker-compose.yml is meant to use a .env file

The GITHUB_ACCESS_TOKEN environment variable is required ([unauthenticated rate limit are low](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28), you can use a [personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens) without any special scopes)

Other environment variable are readed :

- GITHUB_EVENT_API_URL with default "https://api.github.com/events"
- GITHUB_EVENT_API_PAGE_SIZE with default 100 (GitHub API allow 100 and default to 30)
- REFRESH with default "5m" : automatic cache refresh delay
- MAX_CALL with default 90 : limit the number of concurrent requests (GitHub API secondary rate limit is 100 concurrent requests)

```
docker compose up
```

Application will be then running on port `5000`

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

TODO
