// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package observability

import (
	"github.com/lapacek-labs/identity-operator/pkg/result"
)

type Attempt struct {
	Outcome result.Outcome
	Reason  result.Reason
	Phase   Phase
}
type Phase string

const (
	PhasePrecondition Phase = "precondition"
	PhaseFanout       Phase = "fanout"
)

type Fanout struct {
	Total   int
	Failed  int
	Success int
}
