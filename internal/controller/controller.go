// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/lapacek-labs/identity-operator/api/v1alpha1"
	"github.com/lapacek-labs/identity-operator/pkg/errclass"
	"github.com/lapacek-labs/identity-operator/pkg/logging"
	"github.com/lapacek-labs/identity-operator/pkg/observability"
	"github.com/lapacek-labs/identity-operator/pkg/result"
	"github.com/lapacek-labs/identity-operator/pkg/status"
)

const ID = "identity-sync-policy"

type reconcileContext struct {
	start       time.Time
	phase       observability.Phase
	decision    result.Decision
	identity    *v1alpha1.IdentitySyncPolicy
	conditions  *status.ConditionSet
	observation *Observation
	currentHash string
}

// Controller reconciles a IdentitySyncPolicy object.
type Controller struct {
	client  client.Client
	scheme  *runtime.Scheme
	limiter *logging.Limiter
	metrics observability.Recorder
}

func NewController(cl client.Client, sch *runtime.Scheme, lim *logging.Limiter, rec observability.Recorder) *Controller {
	return &Controller{client: cl, scheme: sch, limiter: lim, metrics: rec}
}

// SetupWithManager sets up the controller with the Manager.
func (c *Controller) SetupWithManager(mgr controllerruntime.Manager) error {
	if err := setupIndexers(mgr); err != nil {
		return err
	}
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&v1alpha1.IdentitySyncPolicy{}).
		Named("identity-sync-policy").
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(c.mapRequestToIdentity),
			builder.WithPredicates(sourceSecretDataChanged()),
		).
		Complete(c)
}

// +kubebuilder:rbac:groups=identity.lapacek-labs.org,resources=identitysyncpolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=identity.lapacek-labs.org,resources=identitysyncpolicies/status,verbs=get;patch;update
// +kubebuilder:rbac:groups=identity.lapacek-labs.org,resources=identitysyncpolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=serviceaccounts;secrets,verbs=list;get;watch;create;patch;update

// Reconcile is syncing service accounts and secrets in target namespaces.
func (c *Controller) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logf.FromContext(ctx).WithValues(
		"controller", ID,
		"operation", observability.OpReconcile,
		"request", req.NamespacedName,
	)
	ctx = logf.IntoContext(ctx, logger)
	startTime := time.Now()

	identity := &v1alpha1.IdentitySyncPolicy{}
	err := c.client.Get(ctx, req.NamespacedName, identity)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}
		return controllerruntime.Result{}, err
	}

	conditionSet := status.NewConditionSet(identity.Status.Conditions, identity.GetGeneration(), startTime)

	key := types.NamespacedName{
		Name:      identity.Spec.Secret.SourceRef.Name,
		Namespace: identity.Spec.Secret.SourceRef.Namespace,
	}
	secret := &corev1.Secret{}
	secretErr := c.client.Get(ctx, key, secret)
	if secretErr != nil {
		if apierrors.IsNotFound(secretErr) {
			return c.finish(ctx, reconcileContext{
				phase:      observability.PhasePrecondition,
				identity:   identity,
				conditions: conditionSet,
				decision: result.Decision{
					Outcome:      result.OutcomeFailed,
					Reason:       result.ReasonNotFound,
					RequeueAfter: 5 * time.Minute,
					Msg:          "reference secret not found",
				},
				start: startTime,
			})
		}

		_, errReason := errclass.ClassifyError(secretErr, errclass.NotFoundAsTransient)
		reason := mapErrReasonToResultReason(errReason)

		return c.finish(ctx, reconcileContext{
			phase:      observability.PhasePrecondition,
			identity:   identity,
			conditions: conditionSet,
			decision: result.Decision{
				Outcome: result.OutcomeFailed,
				Reason:  reason,
				Err:     secretErr,
				Msg:     "failed reading reference secret",
			},
			start: startTime,
		})
	}
	currentSecretHash := secretDataHash(secret)

	if shouldFastPath(identity, currentSecretHash) {
		return controllerruntime.Result{}, nil
	}

	observation := reconcileIdentity(ctx, c.scheme, c.client, identity, secret)
	decision := DefaultPolicy().Decide(observation)

	switch decision.Outcome {
	case result.OutcomeSuccess:
		decision.Msg = "fanout completed"
	case result.OutcomePartial:
		decision.Msg = "partial fanout failure"
	case result.OutcomeFailed:
		decision.Msg = "fanout failed"
	}

	return c.finish(ctx, reconcileContext{
		phase:       observability.PhaseFanout,
		identity:    identity,
		conditions:  conditionSet,
		currentHash: currentSecretHash,
		observation: observation,
		decision:    decision,
		start:       startTime,
	})
}

func (c *Controller) finish(ctx context.Context, f reconcileContext) (controllerruntime.Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if f.identity == nil {
		return controllerruntime.Result{}, nil
	}

	if f.conditions != nil {
		// --- PRECONDITION -> ReferenceSecretReady (single writer) ---
		switch f.phase {
		case observability.PhasePrecondition:
			if f.decision.Reason == result.ReasonNotFound {
				markSecretNotFound(f.conditions, "Reference secret not found")
			} else {
				markSecretGetFailed(f.conditions, "Reference secret get failed")
			}
		case observability.PhaseFanout:
			markSecretAvailable(f.conditions, "Reference secret available")
		}

		// --- GLOBAL outcome -> Ready/Degraded ---
		switch f.decision.Outcome {
		case result.OutcomeSuccess:
			markReady(f.conditions, "Reconcile completed")
		default:
			msg := f.decision.Msg
			if msg == "" {
				msg = "Reconcile failed"
			}
			markDegraded(f.conditions, msg)
		}
	}

	desiredHash := ""
	if f.decision.Outcome == result.OutcomeSuccess {
		desiredHash = f.currentHash
	}
	statusPatched := false
	if f.conditions != nil {
		patched, err := c.patchStatusIfChanged(ctx, f.identity, f.conditions, desiredHash)
		if err != nil {
			return controllerruntime.Result{}, err
		}
		statusPatched = patched
	}

	if c.metrics != nil {
		c.metrics.RecordAttempt(observability.Attempt{
			Outcome: f.decision.Outcome,
			Reason:  f.decision.Reason,
			Phase:   f.phase,
		}, time.Since(f.start))

		if f.observation != nil {
			c.metrics.RecordFanout(observability.Fanout{
				Total:   f.observation.Total,
				Success: f.observation.Success,
				Failed:  f.observation.Failed,
			})
		}
	}

	logOperationIfAllowed(
		ctx,
		c.limiter,
		f.phase,
		f.identity,
		f.decision,
		f.observation,
		statusPatched,
	)

	return f.decision.Result()
}

func (c *Controller) patchStatusIfChanged(
	ctx context.Context,
	identity *v1alpha1.IdentitySyncPolicy,
	cs *status.ConditionSet,
	desiredHash string,
) (bool, error) {

	condChanged := cs != nil && cs.Changed()

	hashChanged := desiredHash != "" && identity.Status.ObservedSourceSecretHash != desiredHash

	if !condChanged && !hashChanged {
		return false, nil
	}
	base := identity.DeepCopy()

	if hashChanged {
		identity.Status.ObservedSourceSecretHash = desiredHash
	}
	if cs != nil {
		for _, condition := range cs.Conditions() {
			meta.SetStatusCondition(&identity.Status.Conditions, condition)
		}
	}

	if err := c.client.Status().Patch(ctx, identity, client.MergeFrom(base)); err != nil {
		kind, reason := errclass.ClassifyError(err, errclass.NotFoundAsTransient)
		return false, fmt.Errorf("status patch failed (%s/%s): %w", kind, reason, err)
	}
	return true, nil
}

func (c *Controller) mapRequestToIdentity(ctx context.Context, obj client.Object) []reconcile.Request {
	return mapRequestToIdentity(ctx, c.client, obj)
}
