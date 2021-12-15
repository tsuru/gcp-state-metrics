package app

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
)

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
