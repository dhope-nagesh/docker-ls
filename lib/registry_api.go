package lib

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mayflower/docker-ls/lib/connector"
)

type registryApi struct {
	cfg       Config
	connector connector.Connector
}

func (r *registryApi) endpointUrl(path string) *url.URL {
	url := r.cfg.registryUrl
	// Format URL if registry url extra suffix.
	// For example staging_pgai-platform in https://docker.enterprisedb.com/staging_pgai-platform
	var registryName string
	if strings.Trim(url.Path, " ") != "" {
		registryName = strings.Trim(url.Path, "/")
	}
	fmt.Println("url", url.Path)
	processedPath := path
	if registryName != "" {
		splitPath := strings.Split(path, "/")
		if len(splitPath) > 1 {
			processedPath = fmt.Sprintf("/%s/%s", splitPath[0], registryName)
			for i := 1; i < len(splitPath); i++ {
				processedPath = fmt.Sprintf("%s/%s", processedPath, splitPath[i])
			}
		}
	}

	url.Path = processedPath
	return &url
}

func (r *registryApi) paginatedRequestEndpointUrl(path string, lastApiResponse *http.Response) (url *url.URL, err error) {
	url = r.endpointUrl(path)

	if lastApiResponse != nil {
		linkHeader := lastApiResponse.Header.Get("link")

		if linkHeader != "" {
			// This is a hack to work around what looks like a bug in the registry:
			// the supplied link URL currently lacks scheme and host
			scheme, host := url.Scheme, url.Host

			url, err = parseLinkToNextHeader(linkHeader)

			if err != nil {
				return
			}

			if url.Scheme == "" {
				url.Scheme = scheme
			}

			if url.Host == "" {
				url.Host = host
			}
		}
	} else {
		queryParams := url.Query()
		queryParams.Set("n", strconv.Itoa(int(r.pageSize())))
		url.RawQuery = queryParams.Encode()
	}

	return
}

func (r *registryApi) pageSize() uint {
	return r.cfg.pageSize
}

func (r *registryApi) GetStatistics() connector.Statistics {
	return r.connector.GetStatistics()
}

func NewRegistryApi(cfg Config) (api RegistryApi, err error) {
	err = cfg.Validate()
	if err != nil {
		return
	}

	cfg.LoadCredentialsFromDockerConfig()

	registry := &registryApi{
		cfg: cfg,
	}

	registry.connector = createConnector(&registry.cfg)

	api = registry
	return
}
