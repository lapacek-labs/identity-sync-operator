# Identity Sync Operator

## Overview

Identity Sync Operator is a Kubernetes controller that **safely propagates a source Secret into multiple target namespaces** based on a declarative, cluster‑scoped policy.

The operator is designed with **production safety and low operational noise** in mind:

* idempotent reconciliation
* fast‑path no‑op when nothing changed
* minimal status churn
* log‑only‑on‑failure semantics

Typical use cases include:

* distributing identity / credential material (certs, keys, tokens)
* enforcing consistent Secret presence across namespaces
* reducing manual Secret duplication and drift

---

## Mental Model

* **Source Secret**: the single authoritative Secret
* **IdentitySyncPolicy (CR)**: declares *where* the Secret should exist
* **Target Secrets**: copies created/updated in target namespaces

The operator treats the CR as a **contract**:

> *"Ensure that every listed namespace contains an up‑to‑date copy of this Secret."*

---

## Custom Resource: IdentitySyncPolicy

```yaml
apiVersion: identity.lapacek-labs.org/v1alpha1
kind: IdentitySyncPolicy
metadata:
  name: sync-example
spec:
  secret:
    name: source-secret
    sourceRef:
      name: source-secret
      namespace: source-namespace
  serviceAccount:
    name: identity-sync-operator
  targetNamespaces:
    - app-a
    - app-b
    - app-c
```

### Spec Fields

### Spec Fields

| Field                             | Description                                      |
|----------------------------------|--------------------------------------------------|
| `spec.secret.name`               | Name of the target Secret                        |
| `spec.secret.sourceRef.name`     | Name of the source Secret                        |
| `spec.secret.sourceRef.namespace`| Namespace of the source Secret (required)        |
| `spec.serviceAccount.name`       | ServiceAccount used for target namespaces        |
| `spec.targetNamespaces`          | Explicit list of namespaces to sync into (max 50) |


> The CR is **cluster‑scoped**. `sourceRef.namespace` is mandatory.

---

## Reconciliation Behavior

On each reconcile, the operator:

1. Loads the source Secret
2. Computes a stable fingerprint of the source state
3. Ensures each target namespace contains a matching Secret
4. Updates status **only if state changed**

### Fast‑Path Optimization

If:

* `metadata.generation` has not changed
* source Secret fingerprint matches `status.observedSourceSecretHash`

→ reconciliation exits early with **no API writes and no logs**.

---

## Status & Conditions

The operator exposes a small, explicit set of Conditions:

| Condition              | Meaning                                    |
| ---------------------- | ------------------------------------------ |
| `Ready`                | All target namespaces are in sync          |
| `Degraded`             | One or more namespaces failed to reconcile |
| `ReferenceSecretReady` | Source Secret exists and is readable       |

Conditions are:

* transition‑based
* patched only on change
* designed to minimize etcd churn

---

## Failure Modes

| Scenario                | Behavior                                             |
| ----------------------- | ---------------------------------------------------- |
| Source Secret missing   | `ReferenceSecretReady=False`, no fan‑out             |
| RBAC forbidden          | `Degraded=True`, throttled error logs                |
| Transient API error     | Retry via controller-runtime backoff                 |
| Partial fan‑out failure | `Degraded=True`, successful namespaces remain synced |

The operator never deletes the source Secret and never mutates unrelated resources.

---

## Logging Philosophy

* **No logs on success** (fast‑path or clean reconcile)
* Logs emitted only on failures or state transitions
* Errors are fingerprinted and rate‑limited

---

## RBAC & Security

* Minimal RBAC: read source Secret, manage target Secrets
* No cross‑namespace writes outside declared targets
* Clear trust boundary: the CR defines intent, the operator enforces it

---

## Installation (MVP)

```bash
make install   # install CRDs
make deploy    # deploy controller
```

Tested with:

* kind

---

## Local development (kind)

When developing locally with **kind**, the controller image must be explicitly
loaded into the cluster nodes. Otherwise the kubelet will attempt to pull the
image from a remote registry and the controller will fail to start.

For local development it is recommended to use:

```yaml
imagePullPolicy: IfNotPresent
```


Recommended local workflow:

```bash
# Build local image
make docker-build IMG=localhost/identity-operator:dev

# Load image into kind cluster
kind load docker-image localhost/identity-operator:dev --name identity

# Deploy using the same image reference
make deploy IMG=localhost/identity-operator:dev
```

---

## Non‑Goals

The following are **explicitly out of scope** for MVP:

* namespace label selectors
* dynamic fan‑out beyond explicit lists
* secret transformation or templating
* encryption or key management logic

---

## Design Principles

* declarative over imperative
* idempotence first
* predictable failure modes
* low‑noise by default

---

## Roadmap (Post‑MVP)

* namespaceSelector support
* fan‑out governance (`maxFanout`)
* metrics & SLOs
* validating admission webhook
* OLM / OperatorHub packaging

---

## License

MIT
