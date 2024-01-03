// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	"github.com/chnsz/terraform-provider-kubernetes/manifest/openapi"

	// this is how client-go expects auth plugins to be loaded
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// keys into the global state storage
const (
	OAPIFoundry string = "OPENAPIFOUNDRY"
)

// getDynamicClient returns a configured unstructured (dynamic) client instance
func (s *RawProviderServer) getDynamicClient() (dynamic.Interface, error) {
	if s.dynamicClient != nil {
		return s.dynamicClient, nil
	}
	if s.clientConfig == nil {
		return nil, fmt.Errorf("cannot create dynamic client: no client config")
	}
	dynClient, err := dynamic.NewForConfig(s.clientConfig)
	if err != nil {
		return nil, err
	}
	s.dynamicClient = dynClient
	return dynClient, nil
}

// getDiscoveryClient returns a configured discovery client instance.
func (s *RawProviderServer) getDiscoveryClient() (discovery.DiscoveryInterface, error) {
	if s.discoveryClient != nil {
		return s.discoveryClient, nil
	}
	if s.clientConfig == nil {
		return nil, fmt.Errorf("cannot create discovery client: no client config")
	}
	discoClient, err := discovery.NewDiscoveryClientForConfig(s.clientConfig)
	if err != nil {
		return nil, err
	}
	s.discoveryClient = discoClient
	return discoClient, nil
}

// getRestMapper returns a RESTMapper client instance
func (s *RawProviderServer) getRestMapper() (meta.RESTMapper, error) {
	if s.restMapper != nil {
		return s.restMapper, nil
	}
	dc, err := s.getDiscoveryClient()
	if err != nil {
		return nil, err
	}

	// agr, err := restmapper.GetAPIGroupResources(dc)
	// if err != nil {
	// 	return nil, err
	// }
	// mapper := restmapper.NewDeferredDiscoveryRESTMapper(agr)

	cache := memory.NewMemCacheClient(dc)
	s.restMapper = restmapper.NewDeferredDiscoveryRESTMapper(cache)
	return s.restMapper, nil
}

// getRestClient returns a raw REST client instance
func (s *RawProviderServer) getRestClient() (rest.Interface, error) {
	if s.restClient != nil {
		return s.restClient, nil
	}
	if s.clientConfig == nil {
		return nil, fmt.Errorf("cannot create REST client: no client config")
	}
	restClient, err := rest.UnversionedRESTClientFor(s.clientConfig)
	if err != nil {
		return nil, err
	}
	s.restClient = restClient
	return restClient, nil
}

// getOAPIv2Foundry returns an interface to request tftype types from an OpenAPIv2 spec
func (s *RawProviderServer) getOAPIv2Foundry() (openapi.Foundry, error) {
	oapi2, _ := os.LookupEnv("KUBE_OPEN_API_V2")
	switch oapi2 {
	case "":
		return s.getOAPIv2FoundryFromData()
	case "server":
		return s.getOAPIv2FoundryFromServer()
	default:
		return s.getOAPIv2FoundryFromFile(oapi2)
	}
}

func (s *RawProviderServer) getOAPIv2FoundryFromServer() (openapi.Foundry, error) {
	if s.OAPIFoundry != nil {
		return s.OAPIFoundry, nil
	}

	rc, err := s.getRestClient()
	if err != nil {
		return nil, fmt.Errorf("failed get OpenAPI spec: %s", err)
	}

	rq := rc.Verb("GET").Timeout(30*time.Second).AbsPath("openapi", "v2")
	rs, err := rq.DoRaw(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed get OpenAPI spec: %s", err)
	}

	oapif, err := openapi.NewFoundryFromSpecV2(rs)
	if err != nil {
		return nil, fmt.Errorf("failed construct OpenAPI foundry: %s", err)
	}

	s.OAPIFoundry = oapif

	return oapif, nil
}

func (s *RawProviderServer) getOAPIv2FoundryFromData() (openapi.Foundry, error) {
	b, err := openapi.LoadV2data()
	if err != nil {
		return nil, err
	}
	oapif, err := openapi.NewFoundryFromSpecV2(b)
	if err != nil {
		return nil, fmt.Errorf("failed construct OpenAPI foundry: %s", err)
	}

	s.OAPIFoundry = oapif

	return oapif, nil
}

func (s *RawProviderServer) getOAPIv2FoundryFromFile(fp string) (openapi.Foundry, error) {
	b, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	oapif, err := openapi.NewFoundryFromSpecV2(b)
	if err != nil {
		return nil, fmt.Errorf("failed construct OpenAPI foundry: %s", err)
	}

	s.OAPIFoundry = oapif

	return oapif, nil
}

func loggingTransport(rt http.RoundTripper) http.RoundTripper {
	return &loggingRountTripper{
		ot: rt,
		lt: logging.NewTransport("Kubernetes API", rt),
	}
}

type loggingRountTripper struct {
	ot http.RoundTripper
	lt http.RoundTripper
}

func (t *loggingRountTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Path == "/openapi/v2" {
		// don't trace-log the OpenAPI spec document, it's really big
		return t.ot.RoundTrip(req)
	}
	return t.lt.RoundTrip(req)
}

func (s *RawProviderServer) checkValidCredentials(ctx context.Context) (diags []*tfprotov5.Diagnostic) {
	rc, err := s.getRestClient()
	if err != nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to construct REST client",
			Detail:   err.Error(),
		})
		return
	}
	vpath := []string{"/apis"}
	rs := rc.Get().AbsPath(vpath...).Do(ctx)
	if rs.Error() != nil {
		switch {
		case apierrors.IsUnauthorized(rs.Error()):
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid credentials",
				Detail:   fmt.Sprintf("The credentials configured in the provider block are not accepted by the API server. Error: %s\n\nSet TF_LOG=debug and look for '[InvalidClientConfiguration]' in the log to see actual configuration.", rs.Error().Error()),
			})
		default:
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid configuration for API client",
				Detail:   rs.Error().Error(),
			})
		}
		s.logger.Debug("[InvalidClientConfiguration]", "Config", dump(s.clientConfig))
	}
	return
}
