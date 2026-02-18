package cloudprovider

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	restclient "github.com/NVIDIA/carbide-rest/client"
	"github.com/fabiendupont/cloud-provider-nvidia-bmm/pkg/providerid"
)

// mockClient is a minimal mock for testing
type mockNvidiaBMMClient struct {
	getInstance func(
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
	if m.getInstance != nil {
		return m.getInstance(ctx, org, instanceId, params, reqEditors...)
	}
	return nil, nil
}

func TestInstanceExists(t *testing.T) {
	instanceID := uuid.New()
	pid := providerid.NewProviderID("test-org", "test-tenant", "test-site", instanceID)

	tests := []struct {
		name       string
		node       *v1.Node
		mockClient *mockNvidiaBMMClient
		want       bool
		wantErr    bool
	}{
		{
			name: "instance exists",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
				Spec: v1.NodeSpec{
					ProviderID: pid.String(),
				},
			},
			mockClient: &mockNvidiaBMMClient{
				getInstance: func(
					ctx context.Context, org string, instanceId uuid.UUID,
					params *restclient.GetInstanceParams,
					reqEditors ...restclient.RequestEditorFn,
				) (*restclient.GetInstanceResponse, error) {
					return &restclient.GetInstanceResponse{
						HTTPResponse: &http.Response{StatusCode: 200},
						JSON200: &restclient.Instance{
							Id:   &instanceID,
							Name: ptr("test-instance"),
						},
					}, nil
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "instance not found",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
				Spec: v1.NodeSpec{
					ProviderID: pid.String(),
				},
			},
			mockClient: &mockNvidiaBMMClient{
				getInstance: func(
					ctx context.Context, org string, instanceId uuid.UUID,
					params *restclient.GetInstanceParams,
					reqEditors ...restclient.RequestEditorFn,
				) (*restclient.GetInstanceResponse, error) {
					return &restclient.GetInstanceResponse{
						HTTPResponse: &http.Response{StatusCode: 404},
					}, nil
				},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloud := &NvidiaBMMCloud{
				nvidiaBmmClient: tt.mockClient,
				orgName:         "test-org",
				siteID:          "test-site",
			}

			got, err := cloud.InstanceExists(context.Background(), tt.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("InstanceExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("InstanceExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseProviderID(t *testing.T) {
	instanceID := uuid.New()
	pid := providerid.NewProviderID("myorg", "mytenant", "mysite", instanceID)

	parsed, err := parseProviderID(pid.String())
	if err != nil {
		t.Fatalf("parseProviderID() failed: %v", err)
	}

	if parsed != instanceID {
		t.Errorf("Expected instance ID %s, got %s", instanceID, parsed)
	}
}

func TestParseProviderID_Invalid(t *testing.T) {
	tests := []struct {
		name       string
		providerID string
		wantErr    bool
	}{
		{"empty", "", true},
		{"invalid format", "invalid-format", true},
		{"missing parts", "nvidia-bmm://org/site", true},
		{"invalid uuid", "nvidia-bmm://org/site/not-a-uuid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseProviderID(tt.providerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseProviderID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
