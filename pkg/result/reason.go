// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package result

type Reason string

const (
	ReasonAPIServerError Reason = "APIServerError"
	ReasonPartialFailure Reason = "PartialFailure"
	ReasonInvalidSpec    Reason = "InvalidSpec"
	ReasonForbidden      Reason = "Forbidden"
	ReasonConflict       Reason = "Conflict"
	ReasonNotFound       Reason = "NotFound"
	ReasonTimeout        Reason = "Timeout"
	ReasonUnknown        Reason = "Unknown"
)
