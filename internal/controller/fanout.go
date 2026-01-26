// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/lapacek-labs/identity-operator/api/v1alpha1"
	"github.com/lapacek-labs/identity-operator/pkg/errclass"
)

func reconcileIdentity(
	ctx context.Context,
	k8sScheme *runtime.Scheme,
	k8sClient client.Client,
	identity *v1alpha1.IdentitySyncPolicy,
	secret *corev1.Secret,
) *Observation {
	const maxSample = 50
	observation := NewObservation(len(identity.Spec.TargetNamespaces), maxSample)
	targetNamespaces := identity.Spec.TargetNamespaces
	for _, namespace := range targetNamespaces {
		if fanoutErr := reconcileNamespace(ctx, k8sScheme, k8sClient, identity, namespace, secret); fanoutErr != nil {
			kind, reason := errclass.ClassifyError(fanoutErr, errclass.NotFoundAsTransient)
			observation.ObserveFailure(namespace, kind, reason, fanoutErr)
			continue
		}
		observation.ObserveSuccess()
	}
	return observation
}

func reconcileNamespace(
	ctx context.Context,
	k8sScheme *runtime.Scheme,
	k8sClient client.Client,
	identity *v1alpha1.IdentitySyncPolicy,
	namespace string,
	sourceSecret *corev1.Secret,
) error {
	if err := ensureServiceAccount(ctx, k8sScheme, k8sClient, identity, namespace); err != nil {
		return err
	}
	if err := ensureSecret(ctx, k8sScheme, k8sClient, identity, namespace, sourceSecret); err != nil {
		return err
	}
	return nil
}

func ensureServiceAccount(
	ctx context.Context,
	k8sScheme *runtime.Scheme,
	k8sClient client.Client,
	identity *v1alpha1.IdentitySyncPolicy,
	namespace string,
) error {
	targetServiceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      identity.Spec.ServiceAccount.Name,
		},
	}
	_, err := controllerutil.CreateOrPatch(ctx, k8sClient, targetServiceAccount, func() error {
		ensureManagedMetadata(&targetServiceAccount.ObjectMeta, identity)
		if err := controllerutil.SetControllerReference(identity, targetServiceAccount, k8sScheme); err != nil {
			return err
		}
		return nil
	})
	return err
}

func ensureSecret(
	ctx context.Context,
	k8sScheme *runtime.Scheme,
	k8sClient client.Client,
	identity *v1alpha1.IdentitySyncPolicy,
	namespace string,
	sourceSecret *corev1.Secret,
) error {
	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      identity.Spec.Secret.Name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
	}
	_, err := controllerutil.CreateOrPatch(ctx, k8sClient, targetSecret, func() error {
		ensureManagedMetadata(&targetSecret.ObjectMeta, identity)
		if err := controllerutil.SetControllerReference(identity, targetSecret, k8sScheme); err != nil {
			return err
		}
		targetSecret.Data = sourceSecret.Data
		targetSecret.Type = sourceSecret.Type
		return nil
	})
	return err
}
