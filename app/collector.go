package app

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tsuru/gcp-state-metrics/gcp"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
)

const (
	defaultSyncInterval = 5 * time.Minute
)

var (
	urlMapMatchersLabels = []string{"url_map", "host", "path", "backend_service"}
	projectLabels        = []string{"project", "region"}

	urlMapMatchersDesc = prometheus.NewDesc("gcp_url_map_matchers", "GCP URL map matchers.", urlMapMatchersLabels, nil)
	projectDesc        = prometheus.NewDesc("gcp_project", "GCP Project", projectLabels, nil)
)

type gcpCollector struct {
	sync.RWMutex

	config        *config
	syncRunning   chan struct{}
	requestsLimit chan struct{}
	lastSync      time.Time
	urlMaps       []computepb.UrlMap
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

	urlMaps, err := gcp.ListURLMaps(context.Background(), p.config.gcpProject, p.config.gcpRegion)
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
	ch <- projectDesc
	ch <- urlMapMatchersDesc
}

func (p *gcpCollector) Collect(ch chan<- prometheus.Metric) {
	p.RLock()
	defer p.RUnlock()
	p.checkSync()

	ch <- prometheus.MustNewConstMetric(projectDesc, prometheus.GaugeValue, 1.0, p.config.gcpProject, p.config.gcpRegion)

	for _, urlMap := range p.urlMaps {
		p.collectURLMap(ch, urlMap)
	}
}

func (p *gcpCollector) collectURLMap(ch chan<- prometheus.Metric, urlMap computepb.UrlMap) {
	m := map[string]computepb.PathMatcher{}
	for _, pathMatcher := range urlMap.PathMatchers {
		var name string
		if pathMatcher.Name != nil {
			name = *pathMatcher.Name
		}
		m[name] = *pathMatcher
	}
	for _, hostRule := range urlMap.HostRules {
		var pathMatcherName string
		if hostRule.PathMatcher != nil {
			pathMatcherName = *hostRule.PathMatcher
		}
		pathMatcher := m[pathMatcherName]
		p.collectURLMapPathMatcher(ch, urlMap, *hostRule, pathMatcher)
	}
}

func (p *gcpCollector) collectURLMapPathMatcher(ch chan<- prometheus.Metric, urlMap computepb.UrlMap, hostRule computepb.HostRule, pathMatcher computepb.PathMatcher) {
	for _, host := range hostRule.Hosts {
		for _, pathRule := range pathMatcher.PathRules {
			for _, path := range pathRule.Paths {
				var urlMapName string
				if urlMap.Name != nil {
					urlMapName = *urlMap.Name
				}
				var serviceName string
				if pathRule.Service != nil {
					serviceName = *pathRule.Service
				}
				sn := serviceBackendName(serviceName)
				ch <- prometheus.MustNewConstMetric(urlMapMatchersDesc, prometheus.GaugeValue, 1.0, urlMapName, host, path, sn)
			}
		}
	}
}

func serviceBackendName(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}
