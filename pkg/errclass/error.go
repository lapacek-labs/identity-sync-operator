// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package errclass

type NotFoundPolicy int

const (
	NotFoundAsConfig NotFoundPolicy = iota
	NotFoundAsTransient
)

type ErrorKind string

const (
	KindTransient ErrorKind = "Transient"
	KindTerminal  ErrorKind = "Terminal"
	KindConflict  ErrorKind = "Conflict"
	KindConfig    ErrorKind = "Config"
)

type ErrorReason string

const (
	ReasonForbidden ErrorReason = "Forbidden"
	ReasonConflict  ErrorReason = "Conflict"
	ReasonNotFound  ErrorReason = "NotFound"
	ReasonTimeout   ErrorReason = "Timeout"
	ReasonInvalid   ErrorReason = "Invalid"
	ReasonOther     ErrorReason = "Other"
)

func AllReasons() []ErrorReason {
	return []ErrorReason{
		ReasonForbidden,
		ReasonConflict,
		ReasonNotFound,
		ReasonTimeout,
		ReasonInvalid,
		ReasonOther,
	}
}
