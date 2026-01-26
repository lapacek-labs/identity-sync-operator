// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package controller

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/lapacek-labs/identity-operator/api/v1alpha1"
)

func shouldFastPath(identity *v1alpha1.IdentitySyncPolicy, currentSecretHash string) bool {
	generation := identity.GetGeneration()
	conditions := identity.Status.Conditions
	if !isCurrentAndEqual(conditions, v1alpha1.ConditionReady, metav1.ConditionTrue, generation) {
		return false
	}
	if !isCurrentAndEqual(conditions, v1alpha1.ConditionDegraded, metav1.ConditionFalse, generation) {
		return false
	}
	if !isCurrentAndEqual(conditions, v1alpha1.ConditionReferenceSecretReady, metav1.ConditionTrue, generation) {
		return false
	}
	if identity.Status.ObservedSourceSecretHash != currentSecretHash {
		return false
	}
	return true
}

func isCurrentAndEqual(
	conditions []metav1.Condition,
	condType v1alpha1.ConditionType,
	condStatus metav1.ConditionStatus,
	generation int64,
) bool {
	condition := meta.FindStatusCondition(conditions, string(condType))
	if condition == nil {
		return false
	}
	return condition.Status == condStatus &&
		condition.ObservedGeneration == generation
}
