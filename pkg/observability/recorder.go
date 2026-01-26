// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package observability

import (
	"time"
)

type Recorder interface {
	RecordAttempt(attempt Attempt, latency time.Duration)
	RecordFanout(fanout Fanout)
}
