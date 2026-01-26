// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package status

import (
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func mustTime(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, time.UTC)
}

func mustCond(t string, status metav1.ConditionStatus, reason, msg string, gen int64, tt time.Time) metav1.Condition {
	return metav1.Condition{
		Type:               t,
		Status:             status,
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: gen,
		LastTransitionTime: metav1.NewTime(tt),
	}
}

func assertChanged(t *testing.T, cs *ConditionSet, want bool, msg string) {
	t.Helper()
	if got := cs.Changed(); got != want {
		t.Fatalf("%s: expected Changed()=%v, got %v", msg, want, got)
	}
}

func TestConditions_NoZeroEntriesAndExactLen(t *testing.T) {
	t0 := mustTime(2026, time.January, 2, 10, 0)

	cs := NewConditionSet(nil, 1, t0)

	cs.Set("Zed", metav1.ConditionTrue, "Ok", "z")
	cs.Set("Alpha", metav1.ConditionTrue, "Ok", "a")

	got := cs.Conditions()
	if len(got) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(got))
	}
	for i, c := range got {
		if c.Type == "" {
			t.Fatalf("unexpected zero-value condition at index %d: %+v", i, c)
		}
	}
}

func TestChanged_NoOriginal_NoConditions_False(t *testing.T) {
	t0 := mustTime(2026, time.January, 2, 10, 0)

	cs := NewConditionSet(nil, 1, t0)

	assertChanged(t, cs, false, "empty original + empty current")
}

func TestChanged_NoOriginal_NewCondition_True(t *testing.T) {
	t0 := mustTime(2026, time.January, 2, 10, 0)

	cs := NewConditionSet(nil, 1, t0)

	cs.Set("Ready", metav1.ConditionTrue, "Ok", "ok")

	assertChanged(t, cs, true, "no original + set new condition")
}

func TestChanged_OriginalSame_NoTouch_False(t *testing.T) {
	t0 := mustTime(2026, time.January, 2, 10, 0)

	existing := []metav1.Condition{
		mustCond("Ready", metav1.ConditionTrue, "Ok", "ok", 5, t0),
		mustCond("Degraded", metav1.ConditionFalse, "Ok", "ok", 5, t0),
	}

	// IMPORTANT: reconcileTime and generation match existing => should be identical snapshot.
	cs := NewConditionSet(existing, 5, t0)

	assertChanged(t, cs, false, "original == current (no Set calls)")
}

func TestChanged_SetIdentical_NoOp_False(t *testing.T) {
	t1 := mustTime(2025, time.December, 28, 10, 10)
	t2 := mustTime(2025, time.December, 28, 11, 10)

	existing := []metav1.Condition{{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Ok",
		Message:            "ok",
		ObservedGeneration: 5,
		LastTransitionTime: metav1.NewTime(t1),
	}}

	cs := NewConditionSet(existing, 5, t2)

	cs.Set("Ready", metav1.ConditionTrue, "Ok", "ok")

	// No-op Set => should still be unchanged.
	assertChanged(t, cs, false, "Set identical should not mark Changed")
}

func TestChanged_MessageChange_True(t *testing.T) {
	t1 := mustTime(2026, time.January, 2, 10, 0)
	t2 := mustTime(2026, time.January, 2, 11, 0)

	existing := []metav1.Condition{{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Ok",
		Message:            "old",
		ObservedGeneration: 5,
		LastTransitionTime: metav1.NewTime(t1),
	}}

	cs := NewConditionSet(existing, 5, t2)

	cs.Set("Ready", metav1.ConditionTrue, "Ok", "new")

	assertChanged(t, cs, true, "message changed should mark Changed")
}

func TestChanged_AddNewConditionType_True(t *testing.T) {
	t0 := mustTime(2026, time.January, 2, 10, 0)

	existing := []metav1.Condition{
		mustCond("Ready", metav1.ConditionTrue, "Ok", "ok", 5, t0),
	}

	cs := NewConditionSet(existing, 5, t0)

	cs.Set("Degraded", metav1.ConditionFalse, "Ok", "ok")

	assertChanged(t, cs, true, "adding a new condition type should mark Changed")
}

func TestConditions_SortedByType(t *testing.T) {
	t0 := mustTime(2026, time.January, 2, 10, 0)

	cs := NewConditionSet(nil, 1, t0)
	cs.Set("Zed", metav1.ConditionTrue, "Ok", "z")
	cs.Set("Alpha", metav1.ConditionTrue, "Ok", "a")

	got := cs.Conditions()
	if got[0].Type != "Alpha" || got[1].Type != "Zed" {
		t.Fatalf("expected sorted [Alpha Zed], got [%s %s]", got[0].Type, got[1].Type)
	}
}

func TestSet_NoOpWhenIdentical(t *testing.T) {
	t1 := mustTime(2025, time.December, 28, 10, 10)
	t2 := mustTime(2025, time.December, 28, 11, 10)

	existing := []metav1.Condition{{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Ok",
		Message:            "ok",
		ObservedGeneration: 5,
		LastTransitionTime: metav1.NewTime(t1),
	}}

	cs := NewConditionSet(existing, 5, t2)

	cs.Set("Ready", metav1.ConditionTrue, "Ok", "ok")

	got := cs.conditions["Ready"]
	if got.Type == "" {
		t.Fatalf("unexpected zero-value condition: %+v", got)
	}
	if got.LastTransitionTime.Time != t1 {
		t.Fatalf("expected LTT unchanged=%v, got %v", t1, got.LastTransitionTime.Time)
	}
}

func TestSet_StatusSame_MessageChanges_PreservesLTT(t *testing.T) {
	t1 := mustTime(2026, time.January, 2, 10, 0)
	t2 := mustTime(2026, time.January, 2, 11, 0)

	existing := []metav1.Condition{{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Ok",
		Message:            "old",
		ObservedGeneration: 5,
		LastTransitionTime: metav1.NewTime(t1),
	}}

	cs := NewConditionSet(existing, 5, t2)

	cs.Set("Ready", metav1.ConditionTrue, "Ok", "new")

	got := cs.conditions["Ready"]
	if got.Message != "new" {
		t.Fatalf("expected message updated, got %q", got.Message)
	}
	if got.LastTransitionTime.Time != t1 {
		t.Fatalf("expected LTT preserved=%v, got %v", t1, got.LastTransitionTime.Time)
	}
}

func TestSet_StatusChanges_UpdatesLTT(t *testing.T) {
	t1 := mustTime(2026, time.January, 2, 10, 0)
	t2 := mustTime(2026, time.January, 2, 11, 0)

	existing := []metav1.Condition{{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             "Err",
		Message:            "bad",
		ObservedGeneration: 5,
		LastTransitionTime: metav1.NewTime(t1),
	}}

	cs := NewConditionSet(existing, 5, t2)

	cs.Set("Ready", metav1.ConditionTrue, "Ok", "good")

	got := cs.conditions["Ready"]
	if got.Status != metav1.ConditionTrue {
		t.Fatalf("expected status True, got %s", got.Status)
	}
	if got.LastTransitionTime.Time != t2 {
		t.Fatalf("expected LTT updated=%v, got %v", t2, got.LastTransitionTime.Time)
	}
}

func TestSet_ObservedGenerationChanges_UpdatesGen_PreservesLTT(t *testing.T) {
	t1 := mustTime(2026, time.January, 2, 10, 0)
	t2 := mustTime(2026, time.January, 2, 11, 0)

	existing := []metav1.Condition{{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Ok",
		Message:            "ok",
		ObservedGeneration: 4,
		LastTransitionTime: metav1.NewTime(t1),
	}}

	cs := NewConditionSet(existing, 5, t2)

	cs.Set("Ready", metav1.ConditionTrue, "Ok", "ok")

	got := cs.conditions["Ready"]
	if got.ObservedGeneration != 5 {
		t.Fatalf("expected observedGeneration=5, got %d", got.ObservedGeneration)
	}
	if got.LastTransitionTime.Time != t1 {
		t.Fatalf("expected LTT preserved=%v, got %v", t1, got.LastTransitionTime.Time)
	}
}

func TestConditions_DebugDump_SortedDeterministic(t *testing.T) {
	t0 := mustTime(2026, time.January, 2, 10, 0)

	cs := NewConditionSet(nil, 1, t0)

	cs.Set("Zed", metav1.ConditionTrue, "Ok", "z")
	cs.Set("Alpha", metav1.ConditionTrue, "Ok", "a")

	got := cs.Conditions()
	out := fmt.Sprintf("%s,%s", got[0].Type, got[1].Type)
	if out != "Alpha,Zed" {
		t.Fatalf("expected deterministic order Alpha,Zed; got %s", out)
	}
}
