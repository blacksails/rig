package capabilities

import (
	"context"

	"github.com/rigdev/rig-go-api/operator/api/v1/capabilities"
	"github.com/rigdev/rig/pkg/controller/mod"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Service interface {
	Get(ctx context.Context) (*capabilities.GetResponse, error)
	GetMods() *capabilities.GetPluginsResponse
}

func NewService(
	client client.Client,
	discoveryClient discovery.DiscoveryInterface,
	modManager *mod.Manager,
) Service {
	return &service{
		client:          client,
		discoveryClient: discoveryClient,
		modManager:      modManager,
	}
}

type service struct {
	client          client.Client
	discoveryClient discovery.DiscoveryInterface
	modManager      *mod.Manager
}

// Get implements Service.
func (s *service) Get(ctx context.Context) (*capabilities.GetResponse, error) {
	res := &capabilities.GetResponse{}

	ok, err := s.hasCertManager(ctx)
	if err != nil {
		return nil, err
	}
	res.Ingress = ok

	ok, err = s.hasServiceMonitor(ctx)
	if err != nil {
		return nil, err
	}
	res.HasPrometheusServiceMonitor = ok

	ok, err = s.hasCustomMetricsAPI()
	if err != nil {
		return nil, err
	}
	res.HasCustomMetrics = ok

	ok, err = s.hasVPA(ctx)
	if err != nil {
		return nil, err
	}
	res.HasVerticalPodAutoscaler = ok

	return res, nil
}

func (s *service) hasServiceMonitor(ctx context.Context) (bool, error) {
	if err := s.client.Get(ctx, client.ObjectKey{
		Name: "servicemonitors.monitoring.coreos.com",
	}, &apiextensionsv1.CustomResourceDefinition{}); errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (s *service) hasVPA(ctx context.Context) (bool, error) {
	if err := s.client.Get(ctx, client.ObjectKey{
		Name: "verticalpodautoscalers.autoscaling.k8s.io",
	}, &apiextensionsv1.CustomResourceDefinition{}); errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (s *service) hasCertManager(ctx context.Context) (bool, error) {
	if err := s.client.Get(ctx, client.ObjectKey{
		Name: "certificates.cert-manager.io",
	}, &apiextensionsv1.CustomResourceDefinition{}); errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil

}

func (s *service) hasCustomMetricsAPI() (bool, error) {
	groups, err := s.discoveryClient.ServerGroups()
	if err != nil {
		return false, err
	}

	for _, g := range groups.Groups {
		if g.Name == "custom.metrics.k8s.io" {
			return true, nil
		}
	}

	return false, nil
}

func (s *service) GetMods() *capabilities.GetPluginsResponse {
	var plugins []*capabilities.GetPluginsResponse_Plugin

	for _, p := range s.modManager.GetMods() {
		if p.IsBuiltin {
			plugins = append(plugins, &capabilities.GetPluginsResponse_Plugin{
				Plugin: &capabilities.GetPluginsResponse_Plugin_Builtin{
					Builtin: &capabilities.GetPluginsResponse_Builtin{
						Name: p.Name,
					},
				},
			})
		} else {
			plugins = append(plugins, &capabilities.GetPluginsResponse_Plugin{
				Plugin: &capabilities.GetPluginsResponse_Plugin_ThirdParty{
					ThirdParty: &capabilities.GetPluginsResponse_Thirdparty{
						Name:  p.Name,
						Image: p.Image,
					},
				},
			})
		}
	}

	return &capabilities.GetPluginsResponse{
		Plugins: plugins,
	}
}
