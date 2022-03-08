package app

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultSyncInterval = 5 * time.Minute
	defaultSyncTimeout  = 2 * time.Minute
)

type config struct {
	port         string
	syncInterval time.Duration
	maxRequests  int
	gcpProject   string
	gcpRegion    string
	clusterName  string
}

func readConfig() (*config, error) {
	gcpProject := os.Getenv("GCP_PROJECT")
	gcpRegion := os.Getenv("GCP_REGION")

	if gcpProject == "" || gcpRegion == "" {
		return nil, errors.New("GCP_PROJECT and GCP_REGION must be defined")
	}

	syncInterval, _ := time.ParseDuration(os.Getenv("SYNC_INTERVAL"))
	maxRequests, _ := strconv.Atoi(os.Getenv("MAX_REQUESTS"))
	conf := &config{
		port:         os.Getenv("PORT"),
		syncInterval: syncInterval,
		maxRequests:  maxRequests,
		gcpProject:   gcpProject,
		gcpRegion:    gcpRegion,
	}
	if conf.syncInterval == 0 {
		conf.syncInterval = defaultSyncInterval
	}
	if conf.port == "" {
		conf.port = "19283"
	}

	var err error
	conf.clusterName, err = discoverClusterName()
	if err != nil {
		log.Println("Could not discover cluster name:", err)
	}

	return conf, nil
}

func discoverClusterName() (string, error) {
	req, err := http.NewRequest(http.MethodGet, "http://metadata/computeMetadata/v1/instance/attributes/cluster-name", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	client := &http.Client{
		Timeout:   time.Second * 2,
		Transport: http.DefaultTransport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("unable to retrieve cluster name, status code: " + resp.Status)
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}
