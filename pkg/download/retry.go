package download

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
)

// SleepFunc pauses the execution for at least duration d.
type SleepFunc func(d time.Duration)

var (
	// ActualSleep uses actual time to pause the execution.
	ActualSleep SleepFunc = time.Sleep
)

const (
	// time to sleep between retries is an exponential backoff formula:
	//   t(n) = k * m^n
	expRetryN = 3 // how many times we retry the Download
	expRetryK = time.Second * 3
	expRetryM = 2
)

// WithRetries retrieves a response body using the specified downloader. Any
// error returned from d will be retried (and retrieved response bodies will be
// closed on failures). If the retries do not succeed, the last error is returned.
//
// It sleeps in exponentially increasing durations between retries.
func WithRetries(ctx *log.Context, downloaders []Downloader, sf SleepFunc) (io.ReadCloser, error) {
	var downloadErrors error
	for _, d := range downloaders {
		for n := 0; n < expRetryN; n++ {
			ctx := ctx.With("retry", n)
			status, out, err := Download(ctx, d)
			if err == nil {
				return out, nil
			}

			if downloadErrors != nil {
				downloadErrors = errors.Wrapf(downloadErrors, fmt.Sprintf("Attempt %d: %s ", n+1, err.Error()))
			} else {
				downloadErrors = err
			}

			ctx.Log("error", err)

			if out != nil { // we are not going to read this response body
				out.Close()
			}

			// If there is an access issue while downloading using this downloader, use next downloader
			// For ex. User may have set up access to blob using managed identity, but not using public blob access or vice-versa.
			if isAccessIssueHttpStatusCode(status) {
				break
			}

			// status == -1 the value when there was no http request
			if status != -1 && !isTransientHttpStatusCode(status) {
				ctx.Log("info", fmt.Sprintf("downloader %T returned %v, skipping retries", d, status))
				break
			}

			if n != expRetryN-1 {
				// have more retries to go, sleep before retrying
				slp := expRetryK * time.Duration(int(math.Pow(float64(expRetryM), float64(n))))
				ctx.Log("sleep", slp)
				sf(slp)
			}
		}
	}
	return nil, downloadErrors
}

func isTransientHttpStatusCode(statusCode int) bool {
	switch statusCode {
	case
		http.StatusRequestTimeout,      // 408
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true // timeout and too many requests
	default:
		return false
	}
}

func isAccessIssueHttpStatusCode(statusCode int) bool {
	switch statusCode {
	case
		http.StatusUnauthorized, // 401
		http.StatusForbidden,    // 403
		http.StatusNotFound,     // 404
		http.StatusConflict:     // 409
		return true
	default:
		return false
	}
}
