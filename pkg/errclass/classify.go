// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package errclass

import (
	"context"
	"errors"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func ClassifyError(err error, notFoundPolicy NotFoundPolicy) (ErrorKind, ErrorReason) {
	if err == nil {
		return "", ""
	}

	// --- Fast-path: context / transport errclass (not Kubernetes StatusError) ---
	// context.DeadlineExceeded is typically an RPC/API timeout -> retry.
	if errors.Is(err, context.DeadlineExceeded) {
		return KindTransient, ReasonTimeout
	}
	// context.Canceled usually means controller shutdown / reconcile aborted.
	if errors.Is(err, context.Canceled) {
		return KindTerminal, ReasonOther
	}

	// --- Kubernetes API typed errclass (StatusError under the hood) ---
	switch {
	// Optimistic concurrency (resourceVersion mismatch) -> retry.
	case apierrors.IsConflict(err):
		return KindConflict, ReasonConflict
	// Create race: someone else already created the object -> retry via requeue.
	case apierrors.IsAlreadyExists(err):
		return KindConflict, ReasonConflict
	// RBAC/auth misconfiguration -> non-retriable (config issue).
	case apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err):
		return KindConfig, ReasonForbidden
	// Validation/schema violations (incl. immutable fields) -> non-retriable.
	case apierrors.IsInvalid(err):
		return KindConfig, ReasonInvalid
	// Missing dependency (or delete race).
	// Depending on policy, treat as either transient (wait for dependency) or config error.
	case apierrors.IsNotFound(err):
		if notFoundPolicy == NotFoundAsTransient {
			return KindTransient, ReasonNotFound
		}
		return KindConfig, ReasonNotFound
	// API server timeouts / throttling -> retry with backoff.
	case apierrors.IsTimeout(err) || apierrors.IsServerTimeout(err) || apierrors.IsTooManyRequests(err):
		return KindTransient, ReasonTimeout
	// Explicit InternalError helper (often redundant with the 5xx fallback below).
	case apierrors.IsInternalError(err):
		return KindTransient, ReasonOther
	}

	// --- Fallback: bucket unknown StatusError by HTTP code (5xx => transient) ---
	// Covers ServiceUnavailable and many "internal" api server / etcd related failures.
	var se *apierrors.StatusError
	if errors.As(err, &se) {
		code := int(se.ErrStatus.Code)
		// Some api servers may return Code=0 -> treat as transient (reliability-first).
		if code == 0 || (code >= http.StatusInternalServerError && code <= 599) {
			return KindTransient, ReasonOther
		}
	}

	// --- Final fallback: default to retry ---
	// Unknown errclass are safest to treat as transient unless explicitly proven terminal.
	return KindTransient, ReasonOther
}
