// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package noop

import (
	"time"

	"github.com/lapacek-labs/identity-operator/pkg/observability"
)

type Recorder struct{}

var _ observability.Recorder = Recorder{}

func (Recorder) RecordAttempt(attempt observability.Attempt, latency time.Duration) {}

func (Recorder) RecordFanout(fanout observability.Fanout) {
}
