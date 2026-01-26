// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package result

type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomePartial Outcome = "partial"
	OutcomeFailed  Outcome = "failed"
)
