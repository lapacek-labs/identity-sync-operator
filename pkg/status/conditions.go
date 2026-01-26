// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package status

import (
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewConditionSet(conditions []metav1.Condition, generation int64, reconcileTime time.Time) *ConditionSet {
	byType := make(map[string]metav1.Condition, len(conditions))
	for _, c := range conditions {
		byType[c.Type] = c
	}
	return &ConditionSet{
		original:           conditions,
		conditions:         byType,
		reconcileTime:      reconcileTime,
		observedGeneration: generation,
	}
}

type ConditionSet struct {
	original           []metav1.Condition
	conditions         map[string]metav1.Condition
	reconcileTime      time.Time
	observedGeneration int64
}

func (cs *ConditionSet) Conditions() []metav1.Condition {
	out := make([]metav1.Condition, 0, len(cs.conditions))
	for _, condition := range cs.conditions {
		out = append(out, condition)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Type < out[j].Type
	})
	return out
}

func (cs *ConditionSet) Set(condType string, status metav1.ConditionStatus, reason string, message string) {
	next := metav1.Condition{
		Type:               condType,
		Reason:             reason,
		Status:             status,
		Message:            message,
		ObservedGeneration: cs.observedGeneration,
		LastTransitionTime: metav1.NewTime(cs.reconcileTime),
	}
	if prev, found := cs.conditions[condType]; found {
		if prev.Status == next.Status {
			next.LastTransitionTime = prev.LastTransitionTime
		}
		if cs.isSame(prev, next) {
			return
		}
	}
	cs.conditions[condType] = next
}

func (cs *ConditionSet) IsConditionTrue(condType string) bool {
	if condition, ok := cs.conditions[condType]; ok {
		return condition.Status == metav1.ConditionTrue
	}
	return false
}

func (cs *ConditionSet) Changed() bool {
	if len(cs.original) != len(cs.conditions) {
		return true
	}
	for _, prev := range cs.original {
		next, ok := cs.conditions[prev.Type]
		if !ok {
			return true
		}
		if !cs.isSame(prev, next) {
			return true
		}
	}
	return false
}

func (cs *ConditionSet) isSame(prev, next metav1.Condition) bool {
	return prev.Status == next.Status &&
		prev.Reason == next.Reason &&
		prev.Message == next.Message &&
		prev.ObservedGeneration == next.ObservedGeneration
}
