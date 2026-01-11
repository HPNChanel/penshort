package webhook

import "errors"

// Sentinel errors for webhook operations.
var (
	ErrEndpointNotFound = errors.New("webhook endpoint not found")
	ErrDeliveryNotFound = errors.New("webhook delivery not found")
)
