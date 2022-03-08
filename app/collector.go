package app

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tsuru/gcp-state-metrics/gcp"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
)

var (
	urlMapMatchersLabels  = []string{"url_map", "host", "path", "backend_service"}
	projectLabels         = []string{"project", "region"}
	clusterLabels         = []string{"name"}
	forwardingRulesLabels = []string{
		"forwading_rule",
		"address",
		"type",
		"tier",
		"protocol",
		"kubernetes_resource",
		"kubernetes_namespace",
		"kubernetes_name",
	}

	urlMapMatchersDesc = prometheus.NewDesc("gcp_url_map_matchers", "GCP URL map matchers.", urlMapMatchersLabels, nil)
	projectDesc        = prometheus.NewDesc("gcp_project", "GCP Project", projectLabels, nil)
	gkeClusterNameDesc = prometheus.NewDesc("gcp_gke_cluster", "GCP GKE Cluster", clusterLabels, nil)
	forwadingRulesDesc = prometheus.NewDesc("gcp_forwarding_rules", "GCP Forwading Rules.", forwardingRulesLabels, nil)
)

type gcpCollector struct {
	sync.RWMutex

	config          *config
	syncRunning     chan struct{}
	lastSync        time.Time
	urlMaps         []computepb.UrlMap
	forwardingRules []computepb.ForwardingRule
}

func newGCPCollector(conf *config) (*gcpCollector, error) {
	collector := &gcpCollector{
		config:      conf,
		syncRunning: make(chan struct{}, 1),
	}
	return collector, prometheus.Register(collector)
}

func (p *gcpCollector) sync() {
	defer func() {
		p.Lock()
		p.lastSync = time.Now()
		p.Unlock()
	}()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), defaultSyncTimeout)
	defer cancel()

	urlMaps, err := gcp.ListURLMaps(timeoutCtx, p.config.gcpProject, p.config.gcpRegion)
	if err != nil {
		log.Printf("Failed to list url maps, err %v", err)
	}

	forwardingRules, err := gcp.ListForwardingRules(timeoutCtx, p.config.gcpProject, p.config.gcpRegion)
	if err != nil {
		log.Printf("Failed to list forwarding rules, err %v", err)
	}

	p.Lock()
	p.urlMaps = urlMaps
	p.forwardingRules = forwardingRules
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
	ch <- gkeClusterNameDesc
	ch <- urlMapMatchersDesc
	ch <- forwadingRulesDesc
}

func (p *gcpCollector) Collect(ch chan<- prometheus.Metric) {
	p.RLock()
	defer p.RUnlock()
	p.checkSync()

	ch <- prometheus.MustNewConstMetric(projectDesc, prometheus.GaugeValue, 1.0, p.config.gcpProject, p.config.gcpRegion)
	ch <- prometheus.MustNewConstMetric(gkeClusterNameDesc, prometheus.GaugeValue, 1.0, p.config.clusterName)

	for _, urlMap := range p.urlMaps {
		p.collectURLMap(ch, urlMap)
	}

	for _, fr := range p.forwardingRules {
		p.collectForwardingRule(ch, fr)
	}
}
