package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2/google"
)

type URLMapHostRule struct {
	Hosts       []string `json:"hosts"`
	PathMatcher string   `json:"pathMatcher"`
}

type URLMapPathMatcher struct {
	Name           string           `json:"name"`
	DefaultService string           `json:"defaultService"`
	PathRules      []URLMapPathRule `json:"pathRules"`
}

type URLMapPathRule struct {
	Service string   `json:"service"`
	Paths   []string `json:"paths"`
}

type URLMap struct {
	Name          string              `json:"name"`
	HostRules     []URLMapHostRule    `json:"hostRules"`
	PatchMatchers []URLMapPathMatcher `json:"pathMatchers"`
}

type URLMaps struct {
	Items []URLMap `json:"items"`
}

type URLMapClient struct{}

func (*URLMapClient) List(ctx context.Context, project, region string) ([]URLMap, error) {
	url := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/urlMaps", project, region)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	httpClient, err := google.DefaultClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	m := &URLMaps{}
	err = json.NewDecoder(resp.Body).Decode(m)
	if err != nil {
		return nil, err
	}

	return m.Items, nil
}
