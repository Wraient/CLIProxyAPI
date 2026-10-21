package helps

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ParseServiceRetryAfter preserves a standard service retry deadline without
// applying quota-window grace periods. It must not be used for quota errors.
func ParseServiceRetryAfter(headers http.Header, now time.Time) *time.Duration {
	raw := strings.TrimSpace(headers.Get("Retry-After"))
	if raw == "" {
		return nil
	}
	if seconds, err := strconv.ParseFloat(raw, 64); err == nil {
		if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 || seconds >= float64(math.MaxInt64)/float64(time.Second) {
			return nil
		}
		delay := time.Duration(seconds * float64(time.Second))
		return &delay
	}
	deadline, err := http.ParseTime(raw)
	if err != nil {
		return nil
	}
	delay := deadline.Sub(now)
	if delay < 0 {
		delay = 0
	}
	return &delay
}
