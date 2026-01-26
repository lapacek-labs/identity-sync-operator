// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/lapacek-labs/identity-operator/api/v1alpha1"
)

func cond(t string, s metav1.ConditionStatus, gen int64) metav1.Condition {
	return metav1.Condition{
		Type:               t,
		Status:             s,
		Reason:             "Any",
		Message:            "Any",
		ObservedGeneration: gen,
	}
}

func identityWith(gen int64, observedHash string, conditions ...metav1.Condition) *v1alpha1.IdentitySyncPolicy {
	identity := &v1alpha1.IdentitySyncPolicy{}
	identity.SetGeneration(gen)
	identity.Status.ObservedSourceSecretHash = observedHash
	identity.Status.Conditions = conditions
	return identity
}

func TestShouldFastPath(t *testing.T) {
	const hashA = "hashA"
	const hashB = "hashB"

	tests := []struct {
		name        string
		identity    *v1alpha1.IdentitySyncPolicy
		currentHash string
		want        bool
	}{
		{
			name: "true_when_ready_and_prereqs_ok_for_current_generation",
			identity: identityWith(7, hashA,
				cond("Ready", metav1.ConditionTrue, 7),
				cond("Degraded", metav1.ConditionFalse, 7),
				cond("ReferenceSecretReady", metav1.ConditionTrue, 7),
			),
			currentHash: hashA,
			want:        true,
		},
		{
			name: "false_when_ready_missing",
			identity: identityWith(7, hashA,
				cond("Degraded", metav1.ConditionFalse, 7),
				cond("ReferenceSecretReady", metav1.ConditionTrue, 7),
			),
			currentHash: hashA,
			want:        false,
		},
		{
			name: "false_when_degraded_true",
			identity: identityWith(8, hashA,
				cond("Ready", metav1.ConditionTrue, 8),
				cond("Degraded", metav1.ConditionTrue, 8),
				cond("ReferenceSecretReady", metav1.ConditionTrue, 8),
			),
			currentHash: hashA,
			want:        false,
		},
		{
			name: "false_when_secret_not_found_true",
			identity: identityWith(7, hashA,
				cond("Ready", metav1.ConditionTrue, 7),
				cond("Degraded", metav1.ConditionFalse, 7),
				cond("ReferenceSecretReady", metav1.ConditionFalse, 7),
			),
			currentHash: hashA,
			want:        false,
		},
		{
			name: "false_when_ready_is_stale_generation",
			identity: identityWith(7, hashA,
				cond("Ready", metav1.ConditionTrue, 6),
				cond("Degraded", metav1.ConditionFalse, 7),
				cond("ReferenceSecretReady", metav1.ConditionTrue, 7),
			),
			currentHash: hashA,
			want:        false,
		},
		{
			name: "false_when_any_prereq_is_stale_generation",
			identity: identityWith(7, hashA,
				cond("Ready", metav1.ConditionTrue, 7),
				cond("Degraded", metav1.ConditionFalse, 6),
				cond("ReferenceSecretReady", metav1.ConditionTrue, 7),
			),
			currentHash: hashA,
			want:        false,
		},
		{
			name: "false_when_conditions_present_but_statuses_not_expected",
			identity: identityWith(7, hashA,
				cond("Ready", metav1.ConditionFalse, 7),
				cond("Degraded", metav1.ConditionFalse, 7),
				cond("ReferenceSecretReady", metav1.ConditionTrue, 7),
			),
			want: false,
		},
		{
			name:        "false_when_conditions_empty",
			identity:    identityWith(7, hashB),
			currentHash: hashB,
			want:        false,
		},
		{
			name: "false_when_hash_mismatch",
			identity: identityWith(7, hashA,
				cond("Ready", metav1.ConditionTrue, 7),
				cond("Degraded", metav1.ConditionFalse, 7),
				cond("ReferenceSecretReady", metav1.ConditionTrue, 7),
			),
			currentHash: hashB,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldFastPath(tt.identity, tt.currentHash)
			if got != tt.want {
				t.Fatalf("shouldFastPath()=%v, want %v", got, tt.want)
			}
		})
	}
}
