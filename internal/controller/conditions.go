// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/lapacek-labs/identity-operator/api/v1alpha1"
	"github.com/lapacek-labs/identity-operator/pkg/status"
)

func markReady(cs *status.ConditionSet, message string) {
	cs.Set(string(v1alpha1.ConditionReady), metav1.ConditionTrue, string(v1alpha1.ReasonReconciled), message)
	cs.Set(string(v1alpha1.ConditionDegraded), metav1.ConditionFalse, string(v1alpha1.ReasonReconciled), message)
}

func markDegraded(cs *status.ConditionSet, message string) {
	cs.Set(string(v1alpha1.ConditionReady), metav1.ConditionFalse, string(v1alpha1.ReasonReconcileError), message)
	cs.Set(string(v1alpha1.ConditionDegraded), metav1.ConditionTrue, string(v1alpha1.ReasonReconcileError), message)
}

func markSecretAvailable(cs *status.ConditionSet, message string) {
	cs.Set(string(v1alpha1.ConditionReferenceSecretReady), metav1.ConditionTrue, string(v1alpha1.ReasonSecretAvailable), message)
}

func markSecretNotFound(cs *status.ConditionSet, message string) {
	cs.Set(string(v1alpha1.ConditionReferenceSecretReady), metav1.ConditionFalse, string(v1alpha1.ReasonSecretNotFound), message)
}

func markSecretGetFailed(cs *status.ConditionSet, message string) {
	cs.Set(string(v1alpha1.ConditionReferenceSecretReady), metav1.ConditionFalse, string(v1alpha1.ReasonSecretGetFailed), message)
}
