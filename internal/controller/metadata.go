// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/lapacek-labs/identity-operator/api/v1alpha1"
)

const (
	LabelManagedBy = "app.kubernetes.io/managed-by"
	LabelName      = "app.kubernetes.io/name"

	LabelPolicyName = "identitysyncpolicy.platform.lapacek-labs.org/policy-name"
	LabelPolicyUID  = "identitysyncpolicy.platform.lapacek-labs.org/policy-uid"
)

func ensureManagedMetadata(meta *metav1.ObjectMeta, identity *v1alpha1.IdentitySyncPolicy) {
	if meta == nil || identity == nil {
		return
	}
	if meta.Labels == nil {
		meta.Labels = map[string]string{}
	}

	meta.Labels[LabelName] = ID
	meta.Labels[LabelManagedBy] = ID + "-operator"

	meta.Labels[LabelPolicyName] = identity.Name
	meta.Labels[LabelPolicyUID] = string(identity.UID)
}
