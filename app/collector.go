package app

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tsuru/gcp-state-metrics/gcp"
)

const (
	defaultSyncInterval = 5 * time.Minute
)

var (
	urlMapMatchersLabels = []string{"url_map", "host", "path", "backend_service"}
	urlMapMatchersDesc   = prometheus.NewDesc("gcp_url_map_matchers", "GCP URL map matchers.", urlMapMatchersLabels, nil)
)

type gcpCollector struct {
	sync.RWMutex

	config        *config
	syncRunning   chan struct{}
	requestsLimit chan struct{}
	lastSync      time.Time
	urlMaps       []gcp.URLMap
}

func newGCPCollector(conf *config) (*gcpCollector, error) {
	collector := &gcpCollector{
		config:        conf,
		syncRunning:   make(chan struct{}, 1),
		requestsLimit: make(chan struct{}, conf.maxRequests),
	}
	return collector, prometheus.Register(collector)
}

func (p *gcpCollector) sync() {
	defer func() {
		p.Lock()
		p.lastSync = time.Now()
		p.Unlock()
	}()

	client := &gcp.URLMapClient{}
	urlMaps, err := client.List(context.Background(), p.config.gcpProject, p.config.gcpRegion)
	if err != nil {
		log.Printf("Failed to list url maps, err %v", err)
	}

	p.Lock()

	p.urlMaps = urlMaps
	p.Unlock()
}

func (p *gcpCollector) checkSync() {
	if time.Since(p.lastSync) > p.config.syncInterval {
		select {
		case p.syncRunning <- struct{}{}:
		default:
			// sync already in progress
			return
		}
		go func() {
			defer func() {
				log.Print("[sync] finished GCP sync")
				<-p.syncRunning
			}()
			log.Print("[sync] starting GCP sync")
			p.sync()
		}()
	}
}

func (p *gcpCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- urlMapMatchersDesc
}

func (p *gcpCollector) Collect(ch chan<- prometheus.Metric) {
	p.RLock()
	defer p.RUnlock()
	p.checkSync()

	for _, urlMap := range p.urlMaps {
		p.collectURLMap(ch, urlMap)
	}
}

func (p *gcpCollector) collectURLMap(ch chan<- prometheus.Metric, urlMap gcp.URLMap) {
	m := map[string]gcp.URLMapPathMatcher{}
	for _, pathMatcher := range urlMap.PatchMatchers {
		m[pathMatcher.Name] = pathMatcher
	}
	for _, hostRule := range urlMap.HostRules {
		pathMatcher := m[hostRule.PathMatcher]

		p.collectURLMapPathMatcher(ch, urlMap, hostRule, pathMatcher)

	}
}

func (p *gcpCollector) collectURLMapPathMatcher(ch chan<- prometheus.Metric, urlMap gcp.URLMap, hostRule gcp.URLMapHostRule, pathMatcher gcp.URLMapPathMatcher) {
	for _, host := range hostRule.Hosts {
		for _, pathRule := range pathMatcher.PathRules {
			for _, path := range pathRule.Paths {
				sn := serviceBackendName(pathRule.Service)
				ch <- prometheus.MustNewConstMetric(urlMapMatchersDesc, prometheus.GaugeValue, 1.0, urlMap.Name, host, path, sn)
			}
		}
	}
}

func serviceBackendName(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}
