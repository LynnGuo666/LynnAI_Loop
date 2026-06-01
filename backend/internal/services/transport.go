package services

import (
	"net/http"
	"time"
)

// newSharedTransport returns an http.Transport tuned for connection reuse against
// the small number of upstream hosts the proxy and recovery probe talk to. The
// default transport keeps only 2 idle connections per host, which forces TCP/TLS
// re-handshakes once concurrency exceeds 2; here we raise the idle pool to match
// expected concurrency. Each caller gets its own transport (independent pool).
func newSharedTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 200,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
	}
}
