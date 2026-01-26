// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package prom

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/lapacek-labs/identity-operator/pkg/observability"
)

type Recorder struct {
	reconcileTotal    *prometheus.CounterVec
	reconcileDuration *prometheus.HistogramVec

	fanoutTargetsTotal  prometheus.Counter
	fanoutTargetsSynced prometheus.Counter
}

func NewRecorder(registerer prometheus.Registerer) *Recorder {
	r := &Recorder{
		reconcileTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "identity_operator_reconcile_total",
				Help: "Number of completed reconciles by outcome/reason/phase.",
			},
			[]string{"outcome", "reason", "phase"},
		),

		reconcileDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "identity_operator_reconcile_duration_seconds",
				Help:    "Duration of a reconcile in seconds by outcome/reason/phase.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"outcome", "reason", "phase"},
		),

		fanoutTargetsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "identity_operator_fanout_targets_total",
				Help: "Total number of fanout targets processed (sum of targetsTotal over fanout reconciles).",
			},
		),

		fanoutTargetsSynced: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "identity_operator_fanout_targets_synced",
				Help: "Total number of fanout targets successfully synced (sum of targetsSynced over fanout reconciles).",
			},
		),
	}

	registerer.MustRegister(
		r.reconcileTotal,
		r.reconcileDuration,
		r.fanoutTargetsTotal,
		r.fanoutTargetsSynced,
	)

	return r
}

var _ observability.Recorder = (*Recorder)(nil)

func (r *Recorder) RecordAttempt(attempt observability.Attempt, latency time.Duration) {
	outcome := string(attempt.Outcome)
	reason := string(attempt.Reason)
	phase := string(attempt.Phase)

	r.reconcileTotal.WithLabelValues(outcome, reason, phase).Inc()
	r.reconcileDuration.WithLabelValues(outcome, reason, phase).Observe(latency.Seconds())
}

func (r *Recorder) RecordFanout(fanout observability.Fanout) {
	// These are global counters (no labels) on purpose: very stable, low cardinality.
	// If you later need per-outcome/per-reason fanout metrics, add a separate labeled vec.
	r.fanoutTargetsTotal.Add(float64(fanout.Total))
	r.fanoutTargetsSynced.Add(float64(fanout.Success))
}
