package common

import (
	"fmt"
	"log"
	"net/http"

	"k8s.io/client-go/transport"

	"github.com/chnsz/terraform-provider-kubernetes/common/signer"
)

const (
	AccessKeyConfiguration     = "hw_access_key"
	SecretKeyConfiguration     = "hw_secret_key"
	ProjectIdConfiguration     = "hw_project_id"
	SecurityTokenConfiguration = "hw_security_token"

	ProjectIdHeaderKey     = "X-Project-Id"
	SecurityTokenHeaderKey = "X-Security-Token"
)

func BuildWrappers(c *HuaweiCloudCredential, externalHeader *ExternalHeaderTransport) (transport.WrapperFunc, error) {
	if err := c.Validation(); err != nil {
		return nil, err
	}

	wrappers := make([]transport.WrapperFunc, 0)
	wrappers = append(wrappers, debugLogWrapper)

	wrappers = append(wrappers, c.TransportWrapper)
	if externalHeader != nil {
		wrappers = append(wrappers, externalHeader.TransportWrapper)
	}

	return transport.Wrappers(wrappers...), nil
}

func debugLogWrapper(rt http.RoundTripper) http.RoundTripper {
	//return logging.NewTransport("kubernetes", rt)
	return NewLogTransport("Kubernetes", rt)
}

type HuaweiCloudCredential struct {
	AccessKey     string
	SecretKey     string
	ProjectId     string
	SecurityToken string
}

func (c *HuaweiCloudCredential) Validation() error {
	if len(c.ProjectId) == 0 && (len(c.AccessKey) != 0 || len(c.SecretKey) != 0) {
		return fmt.Errorf(`"hw_project_id", "hw_access_key" and "hw_secret_key" are required`)
	}
	return nil
}

func (c *HuaweiCloudCredential) TransportWrapper(rt http.RoundTripper) http.RoundTripper {
	if err := c.Validation(); err != nil {
		return rt
	}

	if c.ProjectId == "" {
		log.Printf("[TRACE] do not use Huawei Cloud Certification transport for request")
		return rt
	}

	log.Printf("[TRACE] use Huawei Cloud Certification transport for request")
	return &HuaweiCloudAuthTransport{
		hcc:  c,
		next: rt,
	}
}

type HuaweiCloudAuthTransport struct {
	hcc  *HuaweiCloudCredential
	next http.RoundTripper
}

func (d *HuaweiCloudAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add(ProjectIdHeaderKey, d.hcc.ProjectId)
	if len(d.hcc.SecurityToken) != 0 {
		req.Header.Add(SecurityTokenHeaderKey, d.hcc.SecurityToken)
	}

	hs := &signer.Signer{
		Key:    d.hcc.AccessKey,
		Secret: d.hcc.SecretKey,
	}
	if err := hs.Sign(req); err != nil {
		log.Printf("[ERROR] error signing request: %s", err)
		return nil, err
	}

	return d.next.RoundTrip(req)
}

type ExternalHeaderTransport struct {
	headers map[string]string
	next    http.RoundTripper
}

func NewExternalHeaderTransport(headers map[string]string) *ExternalHeaderTransport {
	if len(headers) == 0 {
		return nil
	}
	return &ExternalHeaderTransport{headers: headers}
}

func (e *ExternalHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for key := range e.headers {
		req.Header.Set(key, e.headers[key])
	}
	return e.next.RoundTrip(req)
}

func (e *ExternalHeaderTransport) TransportWrapper(rt http.RoundTripper) http.RoundTripper {
	if len(e.headers) == 0 {
		return rt
	}
	e.next = rt
	return rt
}
