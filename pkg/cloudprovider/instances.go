package cloudprovider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"

	"github.com/fabiendupont/cloud-provider-nvidia-bmm/pkg/providerid"
)

// InstanceExists checks if the instance exists for the given node
func (c *NvidiaBMMCloud) InstanceExists(ctx context.Context, node *v1.Node) (bool, error) {
	providerID := node.Spec.ProviderID
	if providerID == "" {
		return false, fmt.Errorf("node %s has no provider ID", node.Name)
	}

	instanceUUID, err := parseProviderID(providerID)
	if err != nil {
		return false, fmt.Errorf("failed to parse provider ID: %w", err)
	}

	// Check if instance exists in NVIDIA BMM
	resp, err := c.nvidiaBmmClient.GetInstanceWithResponse(ctx, c.orgName, instanceUUID, nil)
	if err != nil {
		klog.Warningf("Instance %s not found: %v", instanceUUID, err)
		return false, nil
	}

	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		klog.Warningf("Instance %s not found, status %d", instanceUUID, resp.StatusCode())
		return false, nil
	}

	return true, nil
}

// InstanceShutdown checks if the instance is shutdown
func (c *NvidiaBMMCloud) InstanceShutdown(ctx context.Context, node *v1.Node) (bool, error) {
	providerID := node.Spec.ProviderID
	if providerID == "" {
		return false, fmt.Errorf("node %s has no provider ID", node.Name)
	}

	instanceUUID, err := parseProviderID(providerID)
	if err != nil {
		return false, fmt.Errorf("failed to parse provider ID: %w", err)
	}

	// Get instance status from NVIDIA BMM
	resp, err := c.nvidiaBmmClient.GetInstanceWithResponse(ctx, c.orgName, instanceUUID, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get instance: %w", err)
	}

	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return false, fmt.Errorf("failed to get instance, status %d", resp.StatusCode())
	}

	instance := resp.JSON200

	// Check if instance is in a shutdown or terminating state
	if instance.Status != nil {
		switch *instance.Status {
		case "Terminating", "Terminated", "Error":
			return true, nil
		default:
			return false, nil
		}
	}

	return false, nil
}

// InstanceMetadata returns metadata for the instance
func (c *NvidiaBMMCloud) InstanceMetadata(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
	providerID := node.Spec.ProviderID
	if providerID == "" {
		return nil, fmt.Errorf("node %s has no provider ID", node.Name)
	}

	instanceUUID, err := parseProviderID(providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse provider ID: %w", err)
	}

	// Get instance details from NVIDIA BMM
	resp, err := c.nvidiaBmmClient.GetInstanceWithResponse(ctx, c.orgName, instanceUUID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("failed to get instance, status %d", resp.StatusCode())
	}

	instance := resp.JSON200

	// Extract node addresses from instance interfaces
	addresses := []v1.NodeAddress{}
	if instance.Interfaces != nil {
		for _, iface := range *instance.Interfaces {
			if iface.IpAddresses != nil {
				for _, ipAddr := range *iface.IpAddresses {
					addresses = append(addresses, v1.NodeAddress{
						Type:    v1.NodeInternalIP,
						Address: ipAddr,
					})
				}
			}
		}
	}

	// Add hostname
	addresses = append(addresses, v1.NodeAddress{
		Type:    v1.NodeHostName,
		Address: node.Name,
	})

	// Determine zone from site ID
	zone := c.getZoneFromSiteID(c.siteID)

	// Determine instance type
	instanceType := "unknown"
	if instance.Id != nil {
		instanceType = "nvidia-bmm-instance"
	}

	metadata := &cloudprovider.InstanceMetadata{
		ProviderID:    providerID,
		InstanceType:  instanceType,
		NodeAddresses: addresses,
		Zone:          zone,
		Region:        c.getRegionFromSiteID(c.siteID),
	}

	klog.V(4).Infof("Instance metadata for %s: %+v", node.Name, metadata)

	return metadata, nil
}

// parseProviderID extracts the instance ID UUID from the provider ID format
// Format: nvidia-bmm://org/tenant/site/instance-id
func parseProviderID(providerIDStr string) (uuid.UUID, error) {
	parsed, err := providerid.ParseProviderID(providerIDStr)
	if err != nil {
		return uuid.UUID{}, err
	}
	return parsed.InstanceID, nil
}

// getZoneFromSiteID returns a zone identifier based on the site ID
func (c *NvidiaBMMCloud) getZoneFromSiteID(siteID string) string {
	// In a real implementation, this would map site IDs to actual zones
	// For now, use the site ID as the zone
	return fmt.Sprintf("nvidia-bmm-zone-%s", siteID)
}

// getRegionFromSiteID returns a region identifier based on the site ID
func (c *NvidiaBMMCloud) getRegionFromSiteID(siteID string) string {
	// In a real implementation, this would map site IDs to regions
	// For now, extract a region from the site ID or use a default
	parts := strings.Split(siteID, "-")
	if len(parts) > 0 {
		return fmt.Sprintf("nvidia-bmm-region-%s", parts[0])
	}
	return "nvidia-bmm-region-default"
}
