// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package v1alpha1

type ConditionType string

const (
	ConditionReady                ConditionType = "Ready"
	ConditionDegraded             ConditionType = "Degraded"
	ConditionReferenceSecretReady ConditionType = "ReferenceSecretReady"
)

type ConditionReason string

const (
	ReasonReconciled      ConditionReason = "Reconciled"
	ReasonReconciling     ConditionReason = "Reconciling"
	ReasonReconcileError  ConditionReason = "ReconcileError"
	ReasonSecretNotFound  ConditionReason = "SecretNotFound"
	ReasonSecretAvailable ConditionReason = "SecretAvailable"
	ReasonSecretGetFailed ConditionReason = "SecretAvailable"

	RBACForbidden ConditionReason = "RBACForbidden"
)
