package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/distribution/distribution/v3/manifest/ocischema"
	"github.com/jszwec/csvutil"
	"log"
	"os"
	"os/exec"
)

type PGExtension struct {
	Name        string `csv:"name" json:"name"`
	Version     string `csv:"version" json:"version"`
	Description string `csv:"description" json:"description"`
}

type TagListJsonResponse struct {
	RepositoryName string   `json:"name"`
	Tags           []string `json:"tags"`
}

func main() {
	registry := os.Getenv("REGISTRY_URL")
	if registry == "" {
		log.Fatal("REGISTRY_URL environment variable not set")
	}
	repository := os.Getenv("REPOSITORY_NAME")
	if repository == "" {
		log.Fatal("REPOSITORY_NAME environment variable not set")
	}
	repos, err := getTags(registry, repository)
	if err != nil {
		log.Fatalf("Error getting tags: %v", err)
	}

	if err := inspectOCIManifest(registry, repository, repos.Tags); err != nil {
		log.Fatalf("Error inspecting OCI manifest: %v", err)
	}
}

func inspectOCIManifest(registry, repository string, tags []string) error {
	for _, tag := range tags {
		log.Println(fmt.Sprintf("inspecting %s:%s...", repository, tag))
		deserializedImageIndex, err := getTagOCIManifest(registry, repository, tag)
		if err != nil {
			log.Fatalf("Error getting tag OCI manifest: %v", err)
		}
		decodedBytes, err := base64.StdEncoding.DecodeString(deserializedImageIndex.Annotations["org.enterprisedb.pgextensions"])
		if err != nil {
			log.Fatalf("Error decoding base64: %v", err)
		}
		var pgExtensions []PGExtension

		if err := csvutil.Unmarshal(decodedBytes, &pgExtensions); err != nil {
			log.Fatalf("Error unmarshalling CSV: %v", err)
		}
		for _, pgExtension := range pgExtensions {
			fmt.Println(pgExtension.Name, pgExtension.Version, pgExtension.Description)
		}
	}
	return nil
}

func getTagOCIManifest(registry, repository, tag string) (*ocischema.DeserializedImageIndex, error) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("./docker-ls tag %s:%s -r %s --manifest-version 3 --raw-manifest --json --progress-indicator=false", repository, tag, registry))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error executing command: %v", err)
	}

	var deserializedImageIndex ocischema.DeserializedImageIndex
	err = json.Unmarshal(output, &deserializedImageIndex)
	if err != nil {
		return nil, fmt.Errorf("error deserializing image index: %v", err)
	}
	return &deserializedImageIndex, nil
}

func getTags(registry, repository string) (*TagListJsonResponse, error) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("./docker-ls tags %s -r %s --json --progress-indicator=false", repository, registry))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error executing command: %v", err)
	}

	var tagListJsonResponse TagListJsonResponse
	err = json.Unmarshal(output, &tagListJsonResponse)
	if err != nil {
		return nil, fmt.Errorf("error deserializing tag list: %v", err)
	}
	return &tagListJsonResponse, nil
}
