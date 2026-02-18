package cloudprovider

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"

	restclient "github.com/NVIDIA/carbide-rest/client"
)

const (
	// ProviderName is the name of the NVIDIA BMM cloud provider
	ProviderName = "nvidia-bmm"

	// Default environment variable names for configuration
	EnvEndpoint = "NVIDIA_BMM_ENDPOINT"
	EnvOrgName  = "NVIDIA_BMM_ORG_NAME"
	EnvToken    = "NVIDIA_BMM_TOKEN"
	EnvSiteID   = "NVIDIA_BMM_SITE_ID"
	EnvTenantID = "NVIDIA_BMM_TENANT_ID"
)

// NvidiaBMMClientInterface defines the methods we need from the NVIDIA BMM REST client
type NvidiaBMMClientInterface interface {
	GetInstanceWithResponse(
		ctx context.Context, org string, instanceId uuid.UUID,
		params *restclient.GetInstanceParams,
		reqEditors ...restclient.RequestEditorFn,
	) (*restclient.GetInstanceResponse, error)
}

// NvidiaBMMCloud implements the Kubernetes cloud provider interface for NVIDIA BMM
type NvidiaBMMCloud struct {
	nvidiaBmmClient NvidiaBMMClientInterface
	orgName         string
	siteID          string
	tenantID        string
}

func init() {
	cloudprovider.RegisterCloudProvider(ProviderName, func(config io.Reader) (cloudprovider.Interface, error) {
		return NewNvidiaBMMCloud(config)
	})
}

// NewNvidiaBMMCloud creates a new NVIDIA BMM cloud provider instance
func NewNvidiaBMMCloud(config io.Reader) (cloudprovider.Interface, error) {
	// Parse configuration
	cfg, err := parseConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create NVIDIA BMM API client
	nvidiaBmmClient, err := restclient.NewClientWithAuth(
		cfg.Endpoint,
		cfg.Token,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create NVIDIA BMM client: %w", err)
	}

	klog.Infof("NVIDIA BMM cloud provider initialized for org=%s, site=%s", cfg.OrgName, cfg.SiteID)

	return &NvidiaBMMCloud{
		nvidiaBmmClient: nvidiaBmmClient,
		orgName:         cfg.OrgName,
		siteID:          cfg.SiteID,
		tenantID:        cfg.TenantID,
	}, nil
}

// NewNvidiaBMMCloudWithClient creates a new NVIDIA BMM cloud provider with injected client (for testing)
func NewNvidiaBMMCloudWithClient(
	client NvidiaBMMClientInterface, orgName, siteID, tenantID string,
) cloudprovider.Interface {
	return &NvidiaBMMCloud{
		nvidiaBmmClient: client,
		orgName:         orgName,
		siteID:          siteID,
		tenantID:        tenantID,
	}
}

// Initialize provides the cloud provider with the client builder and may be called multiple times
func (c *NvidiaBMMCloud) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
	klog.Info("Initializing NVIDIA BMM cloud provider")
}

// LoadBalancer returns a LoadBalancer interface
// NVIDIA BMM does not currently support load balancers
func (c *NvidiaBMMCloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return nil, false
}

// Instances returns an Instances interface (deprecated, use InstancesV2)
func (c *NvidiaBMMCloud) Instances() (cloudprovider.Instances, bool) {
	return nil, false
}

// InstancesV2 returns an InstancesV2 interface for node lifecycle management
func (c *NvidiaBMMCloud) InstancesV2() (cloudprovider.InstancesV2, bool) {
	return c, true
}

// Zones returns a Zones interface
func (c *NvidiaBMMCloud) Zones() (cloudprovider.Zones, bool) {
	return c, true
}

// Clusters returns a Clusters interface (deprecated)
func (c *NvidiaBMMCloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// Routes returns a Routes interface
// NVIDIA BMM does not currently support routes
func (c *NvidiaBMMCloud) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

// ProviderName returns the cloud provider name
func (c *NvidiaBMMCloud) ProviderName() string {
	return ProviderName
}

// HasClusterID returns true if the cluster has a cluster ID
func (c *NvidiaBMMCloud) HasClusterID() bool {
	return true
}

// Config holds the NVIDIA BMM cloud provider configuration
type Config struct {
	// Endpoint is the NVIDIA BMM API endpoint URL
	Endpoint string `yaml:"endpoint"`

	// OrgName is the NVIDIA BMM organization name
	OrgName string `yaml:"orgName"`

	// Token is the NVIDIA BMM API authentication token
	Token string `yaml:"token"`

	// SiteID is the NVIDIA BMM site UUID
	SiteID string `yaml:"siteId"`

	// TenantID is the NVIDIA BMM tenant UUID
	TenantID string `yaml:"tenantId"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	if c.OrgName == "" {
		return fmt.Errorf("orgName is required")
	}
	if c.Token == "" {
		return fmt.Errorf("token is required")
	}
	if c.SiteID == "" {
		return fmt.Errorf("siteId is required")
	}
	if c.TenantID == "" {
		return fmt.Errorf("tenantId is required")
	}
	return nil
}

// parseConfig parses the cloud provider configuration from YAML or environment variables
func parseConfig(config io.Reader) (*Config, error) {
	cfg := &Config{}

	// First, try to parse from config file (YAML)
	if config != nil {
		data, err := io.ReadAll(config)
		if err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}

		if len(data) > 0 {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to unmarshal YAML config: %w", err)
			}
			klog.V(4).Info("Loaded configuration from YAML file")
		}
	}

	// Override with environment variables if present
	if endpoint := os.Getenv(EnvEndpoint); endpoint != "" {
		cfg.Endpoint = endpoint
		klog.V(4).Infof("Using endpoint from environment: %s", endpoint)
	}
	if orgName := os.Getenv(EnvOrgName); orgName != "" {
		cfg.OrgName = orgName
		klog.V(4).Infof("Using orgName from environment: %s", orgName)
	}
	if token := os.Getenv(EnvToken); token != "" {
		cfg.Token = token
		klog.V(4).Info("Using token from environment")
	}
	if siteID := os.Getenv(EnvSiteID); siteID != "" {
		cfg.SiteID = siteID
		klog.V(4).Infof("Using siteID from environment: %s", siteID)
	}
	if tenantID := os.Getenv(EnvTenantID); tenantID != "" {
		cfg.TenantID = tenantID
		klog.V(4).Infof("Using tenantID from environment: %s", tenantID)
	}

	return cfg, nil
}
