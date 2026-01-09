// Package analytics provides click event capture and processing.
package analytics

import (
	"fmt"
	"os"
	"time"
)

// NewConsumerID creates a stable-ish consumer ID for Redis consumer groups.
func NewConsumerID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "worker"
	}
	return fmt.Sprintf("%s-%d-%d", host, os.Getpid(), time.Now().UnixNano())
}
