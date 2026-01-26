// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/lapacek-labs/identity-operator/api/v1alpha1"
	"github.com/lapacek-labs/identity-operator/pkg/logging"
	"github.com/lapacek-labs/identity-operator/pkg/observability"
	"github.com/lapacek-labs/identity-operator/pkg/result"
)

func logOperationIfAllowed(
	ctx context.Context,
	limiter *logging.Limiter,
	phase observability.Phase,
	identity *v1alpha1.IdentitySyncPolicy,
	decision result.Decision,
	observation *Observation,
	statusPatched bool,
) {
	if identity == nil {
		return
	}
	if decision.Outcome == result.OutcomeSuccess {
		return
	}

	logger := logf.FromContext(ctx)

	if statusPatched {
		logFailure(logger, phase, identity, decision, observation, "transition")
		return
	}

	if limiter == nil {
		logFailure(logger, phase, identity, decision, observation, "reminder(no-limiter)")
		return
	}

	reasonsKey := ""
	samplesHash := ""
	if observation != nil {

		const maxSamples = 3

		reasonsKey = formatReasons(observation.ErrorReasonCounts())
		sampleKeys := buildSampleKeys(observation.Samples, maxSamples)
		samplesHash = hashStrings(sampleKeys)
	}

	primary := decision.Reason
	if primary == "" {
		primary = result.ReasonUnknown
	}

	now := time.Now()
	interval := reminderInterval(primary)
	fpReminder := fmt.Sprintf("fail|%s|%s|%s", identity.UID, phase, primary)
	fpChange := fmt.Sprintf("chg|%s|%s|%s|%s|%s", identity.UID, phase, decision.Outcome, reasonsKey, samplesHash)

	// Log if either:
	// - reminder interval elapsed, OR
	// - content changed (short throttle so we don't spam on flapping)
	if limiter.Allow(fpReminder, now, interval) || limiter.Allow(fpChange, now, 30*time.Second) {
		logFailure(logger, phase, identity, decision, observation, "reminder")
	}
}

func reminderInterval(r result.Reason) time.Duration {
	switch r {
	case result.ReasonNotFound:
		return 20 * time.Minute
	case result.ReasonForbidden, result.ReasonInvalidSpec:
		return 5 * time.Minute
	case result.ReasonTimeout, result.ReasonAPIServerError, result.ReasonConflict:
		return 2 * time.Minute
	default:
		return 10 * time.Minute
	}
}

func logFailure(
	logger logr.Logger,
	phase observability.Phase,
	identity *v1alpha1.IdentitySyncPolicy,
	decision result.Decision,
	observation *Observation,
	tag string,
) {
	kv := []any{
		"policy", identity.Name,
		"phase", phase,
		"outcome", decision.Outcome,
		"reason", decision.Reason,
		"msg", decision.Msg,
		"tag", tag,
	}

	// Add fanout-only fields only when observation is present.
	if observation != nil {
		kv = append(kv,
			"success", observation.Success,
			"failed", observation.Failed,
			"total", observation.Total,
			"hasTransient", observation.HasTransient,
			"hasPermanent", observation.HasPermanent,
			"reasons", formatReasons(observation.ErrorReasonCounts()),
			"samples", formatSamples(observation.Samples),
		)
	}

	switch decision.Outcome {
	case result.OutcomeFailed:
		err := decision.Err
		if decision.Err == nil {
			err = fmt.Errorf("reconcile failed")
		}
		logger.Error(err, "Reconcile failed", kv...)
	case result.OutcomePartial:
		logger.Info("Reconcile degraded", kv...)
	}
}

func formatReasons(reasons ReasonCounts) string {
	if len(reasons) == 0 {
		return ""
	}

	reasons.SortInPlace()

	var b strings.Builder
	for _, rc := range reasons {
		if rc.Count == 0 {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(string(rc.Reason))
	}
	return b.String()
}

func formatSamples(samples []Sample) []string {
	if len(samples) == 0 {
		return nil
	}
	const maxMsgLen = 120
	out := make([]string, 0, len(samples))
	for _, s := range samples {
		msg := truncate(s.Message, maxMsgLen)
		out = append(out, fmt.Sprintf("ns=%s kind=%s reason=%s msg=%q",
			s.Namespace, s.Kind, s.Reason, msg))
	}
	return out
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func buildSampleKeys(samples []Sample, max int) []string {
	if max <= 0 || len(samples) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(samples))
	keys := make([]string, 0, len(samples))

	for _, s := range samples {
		key := fmt.Sprintf("ns=%s kind=%s reason=%s", s.Namespace, s.Kind, s.Reason)

		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	sort.Strings(keys)

	if len(keys) > max {
		keys = keys[:max]
	}
	return keys
}

// hashStrings returns a stable, order-independent hash of the input.
// Used only for log throttling fingerprints; collisions are acceptable.
func hashStrings(items []string) string {
	h := fnv.New64a()

	// Copy before sorting to avoid mutating caller-owned data.
	cp := append([]string{}, items...)
	sort.Strings(cp)

	for _, s := range cp {
		_, _ = h.Write([]byte(s))
		// Separator to avoid framing collisions (e.g. ["ab","c"] vs ["a","bc"]).
		_, _ = h.Write([]byte{0})
	}

	sum := h.Sum(nil)
	return hex.EncodeToString(sum)
}
