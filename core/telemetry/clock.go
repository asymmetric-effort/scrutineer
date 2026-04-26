// Package telemetry provides structured binary logging using a custom TLV
// (type-length-value) format with nanosecond-precision timestamps.
package telemetry

import "time"

// NowNano returns the current time as a monotonic nanosecond timestamp.
func NowNano() int64 {
	return time.Now().UnixNano()
}
