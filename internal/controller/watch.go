// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/lapacek-labs/identity-operator/api/v1alpha1"
)

const sourceSecretIndexKey = ".spec.secret.sourceRef"

func mapRequestToIdentity(ctx context.Context, k8sClient client.Client, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}
	logger := logf.FromContext(ctx).
		WithValues(
			"source", "Secret",
			"secret", types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name},
			"handler", "mapRequestToPolicy",
		)

	key := secret.Namespace + "/" + secret.Name

	var list v1alpha1.IdentitySyncPolicyList
	if err := k8sClient.List(ctx, &list, client.MatchingFields{
		sourceSecretIndexKey: key,
	}); err != nil {
		logger.Error(err, "Failed to list identity sync policy")
		return nil
	}

	reqs := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		cr := &list.Items[i]
		reqs = append(reqs, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: cr.Namespace,
				Name:      cr.Name,
			},
		})
	}
	logger.V(1).Info("mapped secret to identities", "count", len(reqs))

	return reqs
}

func sourceSecretDataChanged() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSecret, ok1 := e.ObjectOld.(*corev1.Secret)
			newSecret, ok2 := e.ObjectNew.(*corev1.Secret)
			if !ok1 || !ok2 {
				return false
			}
			return secretDataHash(newSecret) != secretDataHash(oldSecret)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func setupIndexers(mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&v1alpha1.IdentitySyncPolicy{},
		sourceSecretIndexKey,
		indexerFunc,
	)
}

func indexerFunc(obj client.Object) []string {
	cr := obj.(*v1alpha1.IdentitySyncPolicy)
	ref := cr.Spec.Secret.SourceRef
	if ref.Name == "" {
		return nil
	}
	if ref.Namespace == "" {
		return nil
	}
	return []string{ref.Namespace + "/" + ref.Name}
}

// secretDataHash a stable hash of Secret.Data.
// The key order is sorted to keep it deterministic.
func secretDataHash(s *corev1.Secret) string {
	h := sha256.New()

	keys := make([]string, 0, len(s.Data))
	for k := range s.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		h.Write([]byte(k))
		h.Write(s.Data[k])
	}

	return hex.EncodeToString(h.Sum(nil))
}
