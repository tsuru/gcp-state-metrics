package app

import (
	"encoding/json"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
)

type frDesc struct {
	ServiceName string `json:"kubernetes.io/service-name"`
	IngressName string `json:"kubernetes.io/ingress-name"`

	NewServiceName string `json:"networking.gke.io/service-name"`
}

func (p *gcpCollector) collectForwardingRule(ch chan<- prometheus.Metric, fr computepb.ForwardingRule) {
	var desc frDesc
	var kubeResource string
	var kubeNamespace string
	var kubeName string
	err := json.Unmarshal([]byte(strVal(fr.Description)), &desc)
	if err == nil {
		var fullName string
		if desc.NewServiceName != "" {
			kubeResource = "service"
			fullName = desc.NewServiceName
		} else if desc.ServiceName != "" {
			kubeResource = "service"
			fullName = desc.ServiceName
		} else if desc.IngressName != "" {
			kubeResource = "ingress"
			fullName = desc.IngressName
		}
		if fullName != "" {
			parts := strings.SplitN(fullName, "/", 2)
			if len(parts) == 2 {
				kubeNamespace = parts[0]
				kubeName = parts[1]
			}
		}
	}
	labels := []string{
		strVal(fr.Name),
		strVal(fr.IPAddress),
		strVal(fr.LoadBalancingScheme),
		strVal(fr.NetworkTier),
		strVal(fr.IPProtocol),
		kubeResource,
		kubeNamespace,
		kubeName,
	}
	ch <- prometheus.MustNewConstMetric(forwadingRulesDesc, prometheus.GaugeValue, 1.0, labels...)
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
