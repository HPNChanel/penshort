package webhook

import (
	"net"
	"net/http"
	"time"
)

const (
	// ClientTimeout is the total request timeout.
	ClientTimeout = 30 * time.Second
	// DialTimeout is the connection timeout.
	DialTimeout = 10 * time.Second
	// TLSHandshakeTimeout is the TLS negotiation timeout.
	TLSHandshakeTimeout = 10 * time.Second
	// ResponseHeaderTimeout is time to wait for response headers.
	ResponseHeaderTimeout = 15 * time.Second
)

// NewHTTPClient creates an HTTP client configured for webhook delivery.
// It has appropriate timeouts and does not follow redirects.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: ClientTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   DialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   TLSHandshakeTimeout,
			ResponseHeaderTimeout: ResponseHeaderTimeout,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			DisableCompression:    false,
		},
		// Don't follow redirects - security measure
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// HTTPHeaders contains the standard webhook headers.
type HTTPHeaders struct {
	Signature  string // X-Penshort-Signature
	Timestamp  string // X-Penshort-Timestamp
	DeliveryID string // X-Penshort-Delivery-Id
}

// HeaderNames for webhook requests.
const (
	HeaderSignature  = "X-Penshort-Signature"
	HeaderTimestamp  = "X-Penshort-Timestamp"
	HeaderDeliveryID = "X-Penshort-Delivery-Id"
)

// SetWebhookHeaders applies webhook headers to an HTTP request.
func SetWebhookHeaders(req *http.Request, headers HTTPHeaders) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderSignature, headers.Signature)
	req.Header.Set(HeaderTimestamp, headers.Timestamp)
	req.Header.Set(HeaderDeliveryID, headers.DeliveryID)
	req.Header.Set("User-Agent", "Penshort-Webhook/1.0")
}
