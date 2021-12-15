package app

import (
	"errors"
	"os"
	"strconv"
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
	return conf, nil
}
