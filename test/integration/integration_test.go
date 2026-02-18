package integration

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"

	restclient "github.com/NVIDIA/carbide-rest/client"
	nvidiabmmprovider "github.com/fabiendupont/cloud-provider-nvidia-bmm/pkg/cloudprovider"
)

var (
	ctx        context.Context
	cancel     context.CancelFunc
	cloud      cloudprovider.Interface
	mockClient *mockNvidiaBMMClient
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cloud Provider Integration Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())

	// Create mock client
	mockClient = &mockNvidiaBMMClient{}

	// Create cloud provider with mock client
	cloud = nvidiabmmprovider.NewNvidiaBMMCloudWithClient(
		mockClient,
		"test-org",
		"8a880c71-fe4b-4e43-9e24-ebfcb8a84c5f",
		"b013708a-99f0-47b2-a630-cabb4ae1d3df",
	)
})

var _ = AfterSuite(func() {
	cancel()
})

// mockHTTPResponse creates a mock HTTP response with the given status code
func mockHTTPResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader([]byte{})),
		Header:     make(http.Header),
	}
}

// mockNvidiaBMMClient for testing
type mockNvidiaBMMClient struct {
	getInstanceFunc func(
		ctx context.Context, org string, instanceId uuid.UUID,
		params *restclient.GetInstanceParams,
		reqEditors ...restclient.RequestEditorFn,
	) (*restclient.GetInstanceResponse, error)
}

func (m *mockNvidiaBMMClient) GetInstanceWithResponse(
	ctx context.Context, org string, instanceId uuid.UUID,
	params *restclient.GetInstanceParams,
	reqEditors ...restclient.RequestEditorFn,
) (*restclient.GetInstanceResponse, error) {
	if m.getInstanceFunc != nil {
		return m.getInstanceFunc(ctx, org, instanceId, params, reqEditors...)
	}

	// Default: return a running instance with IP addresses
	status := restclient.InstanceStatus("Running")
	ipAddresses := []string{"10.100.1.10"}

	return &restclient.GetInstanceResponse{
		HTTPResponse: mockHTTPResponse(200),
		JSON200: &restclient.Instance{
			Id:     &instanceId,
			Name:   ptr("test-instance"),
			Status: &status,
			Interfaces: &[]restclient.Interface{
				{
					IpAddresses: &ipAddresses,
				},
			},
		},
	}, nil
}

var _ = Describe("InstancesV2 Interface", func() {
	var (
		node       *corev1.Node
		instanceID uuid.UUID
	)

	BeforeEach(func() {
		instanceID = uuid.MustParse("12345678-1234-1234-1234-123456789abc")
		providerID := "nvidia-bmm://test-org/test-tenant/8a880c71-fe4b-4e43-9e24-ebfcb8a84c5f/" + instanceID.String()

		node = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node",
			},
			Spec: corev1.NodeSpec{
				ProviderID: providerID,
			},
		}
	})

	Describe("InstanceExists", func() {
		It("should return true for existing instance", func() {
			instancesV2, supported := cloud.InstancesV2()
			Expect(supported).To(BeTrue())

			exists, err := instancesV2.InstanceExists(ctx, node)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should return false for non-existent instance", func() {
			// Override mock to return 404
			mockClient.getInstanceFunc = func(
				ctx context.Context, org string, instanceId uuid.UUID,
				params *restclient.GetInstanceParams,
				reqEditors ...restclient.RequestEditorFn,
			) (*restclient.GetInstanceResponse, error) {
				return &restclient.GetInstanceResponse{
					HTTPResponse: mockHTTPResponse(404),
				}, nil
			}

			instancesV2, _ := cloud.InstancesV2()
			exists, err := instancesV2.InstanceExists(ctx, node)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())

			// Reset mock
			mockClient.getInstanceFunc = nil
		})
	})

	Describe("InstanceShutdown", func() {
		It("should return false for running instance", func() {
			instancesV2, _ := cloud.InstancesV2()

			shutdown, err := instancesV2.InstanceShutdown(ctx, node)
			Expect(err).NotTo(HaveOccurred())
			Expect(shutdown).To(BeFalse())
		})

		It("should return true for terminated instance", func() {
			// Override mock to return terminated status
			mockClient.getInstanceFunc = func(
				ctx context.Context, org string, instanceId uuid.UUID,
				params *restclient.GetInstanceParams,
				reqEditors ...restclient.RequestEditorFn,
			) (*restclient.GetInstanceResponse, error) {
				status := restclient.InstanceStatus("Terminated")
				return &restclient.GetInstanceResponse{
					HTTPResponse: mockHTTPResponse(200),
					JSON200: &restclient.Instance{
						Id:     &instanceId,
						Status: &status,
					},
				}, nil
			}

			instancesV2, _ := cloud.InstancesV2()
			shutdown, err := instancesV2.InstanceShutdown(ctx, node)
			Expect(err).NotTo(HaveOccurred())
			Expect(shutdown).To(BeTrue())

			// Reset mock
			mockClient.getInstanceFunc = nil
		})
	})

	Describe("InstanceMetadata", func() {
		It("should return metadata with addresses and zone", func() {
			instancesV2, _ := cloud.InstancesV2()

			metadata, err := instancesV2.InstanceMetadata(ctx, node)
			Expect(err).NotTo(HaveOccurred())
			Expect(metadata).NotTo(BeNil())
			Expect(metadata.ProviderID).To(Equal(node.Spec.ProviderID))
			Expect(metadata.NodeAddresses).NotTo(BeEmpty())
			Expect(metadata.Zone).To(ContainSubstring("nvidia-bmm-zone"))
			Expect(metadata.Region).To(ContainSubstring("nvidia-bmm-region"))
		})
	})
})

var _ = Describe("Zones Interface", func() {
	Describe("GetZone", func() {
		It("should return zone and region", func() {
			zones, supported := cloud.Zones()
			Expect(supported).To(BeTrue())

			zone, err := zones.GetZone(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(zone.FailureDomain).To(ContainSubstring("nvidia-bmm-zone"))
			Expect(zone.Region).To(ContainSubstring("nvidia-bmm-region"))
		})
	})

	Describe("GetZoneByProviderID", func() {
		It("should return zone for provider ID", func() {
			zones, _ := cloud.Zones()
			providerID := "nvidia-bmm://test-org/test-tenant/" +
				"8a880c71-fe4b-4e43-9e24-ebfcb8a84c5f/" +
				"12345678-1234-1234-1234-123456789abc"

			zone, err := zones.GetZoneByProviderID(ctx, providerID)
			Expect(err).NotTo(HaveOccurred())
			Expect(zone.FailureDomain).To(ContainSubstring("nvidia-bmm-zone"))
			Expect(zone.Region).To(ContainSubstring("nvidia-bmm-region"))
		})
	})

	Describe("GetZoneByNodeName", func() {
		It("should return zone for node name", func() {
			zones, _ := cloud.Zones()
			nodeName := types.NodeName("test-node")

			zone, err := zones.GetZoneByNodeName(ctx, nodeName)
			Expect(err).NotTo(HaveOccurred())
			Expect(zone.FailureDomain).To(ContainSubstring("nvidia-bmm-zone"))
			Expect(zone.Region).To(ContainSubstring("nvidia-bmm-region"))
		})
	})
})

func ptr[T any](v T) *T {
	return &v
}
