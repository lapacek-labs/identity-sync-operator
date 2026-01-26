// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package controller

import (
	"sort"
	"time"

	"github.com/lapacek-labs/identity-operator/pkg/errclass"
	"github.com/lapacek-labs/identity-operator/pkg/result"
)

type Policy struct {
	TransientDelay time.Duration
	PermanentDelay time.Duration
}

func DefaultPolicy() Policy {
	return Policy{
		TransientDelay: 2 * time.Minute,
		PermanentDelay: 10 * time.Minute,
	}
}

func (p Policy) Decide(obs *Observation) result.Decision {
	var outcome result.Outcome
	switch {
	case obs.Total == 0:
		outcome = result.OutcomeSuccess
	case obs.Success == obs.Total:
		outcome = result.OutcomeSuccess
	case obs.Success == 0:
		outcome = result.OutcomeFailed
	default:
		outcome = result.OutcomePartial
	}

	dec := result.Decision{
		Outcome: outcome,
		Reason:  obs.PrimaryReason(),
		Msg:     "",
	}

	if outcome != result.OutcomeSuccess {
		if obs.HasTransient {
			dec.RequeueAfter = p.TransientDelay
		} else {
			dec.RequeueAfter = p.PermanentDelay
		}
	}

	return dec
}

type Observation struct {
	Reasons      map[errclass.ErrorReason]int
	Samples      []Sample
	MaxSample    int
	Success      int
	Failed       int
	Total        int
	HasTransient bool
	HasPermanent bool
}

const (
	minSample = 8
)

func NewObservation(total, maxSample int) *Observation {
	return &Observation{
		MaxSample: maxSample,
		Samples:   make([]Sample, 0, min(minSample, maxSample)),
		Total:     total,
	}
}

func (obs *Observation) ObserveSuccess() {
	obs.Success++
}

func (obs *Observation) ObserveFailure(namespace string, kind errclass.ErrorKind, reason errclass.ErrorReason, err error) {
	obs.Failed++

	if kind == errclass.KindTransient || kind == errclass.KindConflict {
		obs.HasTransient = true
	}
	if kind == errclass.KindTerminal || kind == errclass.KindConfig {
		obs.HasPermanent = true
	}

	if obs.Reasons == nil {
		obs.Reasons = make(map[errclass.ErrorReason]int, len(errclass.AllReasons()))
	}
	obs.Reasons[reason]++

	if len(obs.Samples) < obs.MaxSample {
		message := ""
		if err != nil {
			message = err.Error()
		}
		obs.Samples = append(obs.Samples, Sample{
			Namespace: namespace,
			Message:   message,
			Reason:    reason,
			Kind:      kind,
		})
	}
}

func (obs *Observation) PrimaryReason() result.Reason {
	if len(obs.Reasons) == 0 {
		return result.ReasonUnknown
	}
	reasons := obs.ErrorReasonCounts()
	if len(reasons) == 0 {
		return result.ReasonUnknown
	}
	reasons.SortInPlace()
	primary := reasons[0].Reason
	return mapErrReasonToResultReason(primary)
}

func (obs *Observation) ErrorReasonCounts() ReasonCounts {
	reasons := make(ReasonCounts, 0, len(obs.Reasons))
	for r, c := range obs.Reasons {
		if c == 0 {
			continue
		}
		reasons = append(reasons, ReasonCount{Reason: r, Count: c})
	}
	return reasons
}

func mapErrReasonToResultReason(reason errclass.ErrorReason) result.Reason {
	switch reason {
	case errclass.ReasonNotFound:
		return result.ReasonNotFound
	case errclass.ReasonForbidden:
		return result.ReasonForbidden
	case errclass.ReasonInvalid:
		return result.ReasonInvalidSpec
	case errclass.ReasonConflict:
		return result.ReasonConflict
	case errclass.ReasonTimeout:
		return result.ReasonTimeout
	case errclass.ReasonOther:
		return result.ReasonAPIServerError
	default:
		return result.ReasonUnknown
	}
}

type Sample struct {
	Namespace string
	Message   string
	Reason    errclass.ErrorReason
	Kind      errclass.ErrorKind
}

type ReasonCounts []ReasonCount

type ReasonCount struct {
	Reason errclass.ErrorReason
	Count  int
}

// SortInPlace sorts reasons in deterministic order.
//
// --- Winner selection ---
// 1) Most frequent reason wins (represents what is actually happening).
// 2) Tie-break by explicit priority (actionability/severity), not string order.
// 3) Final fallback: lexicographic order for full determinism.
func (rc ReasonCounts) SortInPlace() {
	sort.Slice(rc, func(i, j int) bool {
		if rc[i].Count != rc[j].Count {
			return rc[i].Count > rc[j].Count
		}
		pi, pj := errReasonPriority(rc[i].Reason), errReasonPriority(rc[j].Reason)
		if pi != pj {
			return pi > pj
		}
		return rc[i].Reason < rc[j].Reason
	})
}

// Higher number = higher priority in ties.
//
// --- Priority rationale ---
// Invalid   -> user must fix spec/config, retries won't help.
// Forbidden -> RBAC/auth misconfig, also non-retriable until fixed.
// NotFound  -> missing dependency/delete race; policy decided kind earlier.
// Conflict  -> optimistic concurrency; retriable noise.
// Timeout   -> throttling/timeouts; retriable, often systemic.
// Other     -> fallback/unknown bucket.
func errReasonPriority(r errclass.ErrorReason) int {
	switch r {
	case errclass.ReasonInvalid:
		return 60 // user must fix spec/config
	case errclass.ReasonForbidden:
		return 50 // RBAC/auth config
	case errclass.ReasonNotFound:
		return 40 // missing dependency (policy decides transient/config earlier)
	case errclass.ReasonConflict:
		return 30 // optimistic concurrency / races
	case errclass.ReasonTimeout:
		return 20 // timeouts/throttling
	case errclass.ReasonOther:
		return 10 // fallback/unknown
	default:
		return 0
	}
}
