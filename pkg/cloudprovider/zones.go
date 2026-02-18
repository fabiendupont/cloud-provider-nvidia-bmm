package cloudprovider

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"
)

// GetZone returns the Zone containing the current zone and locality region that the program is running in
func (c *NvidiaBMMCloud) GetZone(ctx context.Context) (cloudprovider.Zone, error) {
	zone := cloudprovider.Zone{
		FailureDomain: c.getZoneFromSiteID(c.siteID),
		Region:        c.getRegionFromSiteID(c.siteID),
	}

	return zone, nil
}

// GetZoneByProviderID returns the Zone containing the zone and region for a specific provider ID
func (c *NvidiaBMMCloud) GetZoneByProviderID(ctx context.Context, providerID string) (cloudprovider.Zone, error) {
	// Parse provider ID to get site ID
	// For now, use the configured site ID
	zone := cloudprovider.Zone{
		FailureDomain: c.getZoneFromSiteID(c.siteID),
		Region:        c.getRegionFromSiteID(c.siteID),
	}

	return zone, nil
}

// GetZoneByNodeName returns the Zone containing the zone and region for a specific node
func (c *NvidiaBMMCloud) GetZoneByNodeName(ctx context.Context, nodeName types.NodeName) (cloudprovider.Zone, error) {
	// All nodes in an NVIDIA BMM cluster are in the same site/zone
	zone := cloudprovider.Zone{
		FailureDomain: c.getZoneFromSiteID(c.siteID),
		Region:        c.getRegionFromSiteID(c.siteID),
	}

	return zone, nil
}
