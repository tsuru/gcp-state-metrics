package gcp

import (
	"context"

	computev1 "cloud.google.com/go/compute/apiv1"
	"google.golang.org/api/iterator"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
)

func ListURLMaps(ctx context.Context, project, region string) ([]computepb.UrlMap, error) {
	urlMapCli, err := computev1.NewRegionUrlMapsRESTClient(ctx)
	if err != nil {
		return nil, err
	}
	urlMapIt := urlMapCli.List(ctx, &computepb.ListRegionUrlMapsRequest{
		Project: project,
		Region:  region,
	})
	var result []computepb.UrlMap
	for {
		urlMap, err := urlMapIt.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		result = append(result, *urlMap)
	}
	return result, nil
}
