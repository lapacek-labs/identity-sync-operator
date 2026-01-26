// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/lapacek-labs/identity-operator/api/v1alpha1"
)

type identityFixture struct {
	identityName        string
	namespaceName       string
	serviceAccountName  string
	secretName          string
	sourceSecretName    string
	sourceNamespaceName string
	tokenName           string
	tokenValue          string
	targetNamespaces    []string
}

var _ = Describe("IdentitySyncPolicy Controller", func() {
	Context("with valid source secret and namespaces", func() {

		ctx := context.Background()
		testData := &identityFixture{}

		BeforeEach(func() {
			testData = newIdentityFixture()
			seedIdentityFixture(testData)
		})

		It("should successfully reconcile the resource", func() {
			for _, targetNamespace := range testData.targetNamespaces {
				Eventually(func() error {
					saKey := types.NamespacedName{Name: testData.serviceAccountName, Namespace: targetNamespace}
					return k8sClient.Get(ctx, saKey, &corev1.ServiceAccount{})
				}, 5*time.Second, 100*time.Millisecond).Should(Succeed())

				secretKey := types.NamespacedName{Name: testData.secretName, Namespace: targetNamespace}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, &corev1.Secret{})
				}, 5*time.Second, 100*time.Millisecond).Should(Succeed())

				Eventually(func() string {
					s := &corev1.Secret{}
					_ = k8sClient.Get(ctx, secretKey, s)
					return string(s.Data[testData.tokenName])
				}, 5*time.Second, 100*time.Millisecond).Should(Equal(testData.tokenValue))
			}
		})

		It("updates resources when spec.targetNamespaces expands", func() {
			additionalTargetNs := uniqueStr("app-3")
			testData.targetNamespaces = append(testData.targetNamespaces, additionalTargetNs)
			Expect(createNamespace(ctx, additionalTargetNs, k8sClient)).To(Succeed())

			Eventually(func() error {
				identity := &v1alpha1.IdentitySyncPolicy{}
				key := types.NamespacedName{Name: testData.identityName, Namespace: testData.namespaceName}
				if err := k8sClient.Get(ctx, key, identity); err != nil {
					return err
				}
				identity.Spec.TargetNamespaces = testData.targetNamespaces
				return k8sClient.Update(ctx, identity)
			}, 5*time.Second, 100*time.Millisecond).Should(Succeed())

			Eventually(func() error {
				saKey := types.NamespacedName{Name: testData.serviceAccountName, Namespace: additionalTargetNs}
				return k8sClient.Get(ctx, saKey, &corev1.ServiceAccount{})
			}, 5*time.Second, 100*time.Millisecond).Should(Succeed())

			secretKey := types.NamespacedName{Name: testData.secretName, Namespace: additionalTargetNs}
			Eventually(func() error {
				return k8sClient.Get(ctx, secretKey, &corev1.Secret{})
			}, 5*time.Second, 100*time.Millisecond).Should(Succeed())

			Eventually(func() string {
				s := &corev1.Secret{}
				_ = k8sClient.Get(ctx, secretKey, s)
				return string(s.Data[testData.tokenName])
			}, 5*time.Second, 100*time.Millisecond).Should(Equal(testData.tokenValue))
		})

		It("does not recreate target secret when nothing changes", func() {
			var initialRV string
			secretKey := types.NamespacedName{Name: testData.secretName, Namespace: testData.targetNamespaces[0]}
			Eventually(func() error {
				s := &corev1.Secret{}
				if err := k8sClient.Get(ctx, secretKey, s); err != nil {
					return err
				}
				initialRV = s.ResourceVersion
				if initialRV == "" {
					return fmt.Errorf("empty resource version")
				}
				return nil
			}).Should(Succeed())

			Consistently(func() string {
				s := &corev1.Secret{}
				_ = k8sClient.Get(ctx, secretKey, s)
				return s.ResourceVersion
			}, 2*time.Second, 200*time.Millisecond).Should(Equal(initialRV))
		})

		It("updates target Secret when source Secret changes", func() {
			Eventually(func() error {
				source := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testData.sourceSecretName,
						Namespace: testData.sourceNamespaceName,
					},
					Data: map[string][]byte{
						testData.tokenName: []byte("R3G3N3R8T3D"),
					},
				}
				if source.Data == nil {
					source.Data = map[string][]byte{}
				}
				source.Data[testData.tokenName] = []byte("R3G3N3R8T3D")
				return k8sClient.Update(ctx, source)
			}, 5*time.Second, 100*time.Millisecond).Should(Succeed())

			for _, namespace := range testData.targetNamespaces {
				secretKey := types.NamespacedName{
					Name:      testData.secretName,
					Namespace: namespace,
				}
				Eventually(func() string {
					target := &corev1.Secret{}
					_ = k8sClient.Get(ctx, secretKey, target)
					return string(target.Data[testData.tokenName])
				}, 5*time.Second, 100*time.Millisecond).Should(Equal("R3G3N3R8T3D"))
			}
		})
	})
})

func createNamespace(ctx context.Context, name string, c client.Client) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return c.Create(ctx, namespace)
}

func createSecret(ctx context.Context, name, namespace string, data map[string][]byte, c client.Client) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if data != nil {
		secret.Data = data
	}
	return c.Create(ctx, secret)
}

func createIdentity(ctx context.Context, testData identityFixture, c client.Client) error {
	policyCustomResourceDefinition := &v1alpha1.IdentitySyncPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testData.identityName,
			Namespace: testData.namespaceName,
		},
		Spec: v1alpha1.IdentitySyncPolicySpec{
			TargetNamespaces: testData.targetNamespaces,
			ServiceAccount: v1alpha1.ServiceAccount{
				Name: testData.serviceAccountName,
			},
			Secret: v1alpha1.Secret{
				Name: testData.secretName,
				SourceRef: v1alpha1.NamespacedNameRef{
					Name:      testData.sourceSecretName,
					Namespace: testData.sourceNamespaceName,
				},
			},
		},
	}
	return c.Create(ctx, policyCustomResourceDefinition)
}

func newIdentityFixture() *identityFixture {
	return &identityFixture{
		identityName:        uniqueStr("identity-sync-policy"),
		namespaceName:       uniqueStr("namespace"),
		serviceAccountName:  uniqueStr("service-account"),
		secretName:          uniqueStr("secret"),
		sourceSecretName:    uniqueStr("source-secret"),
		sourceNamespaceName: uniqueStr("source-namespace"),
		tokenName:           "token",
		tokenValue:          "t0k3n",
		targetNamespaces:    []string{uniqueStr("app-1"), uniqueStr("app-2")},
	}
}

func seedIdentityFixture(testData *identityFixture) {
	Expect(createNamespace(ctx, testData.namespaceName, k8sClient)).To(Succeed())
	Expect(createNamespace(ctx, testData.sourceNamespaceName, k8sClient)).To(Succeed())
	for _, targetNamespace := range testData.targetNamespaces {
		Expect(createNamespace(ctx, targetNamespace, k8sClient)).To(Succeed())
	}
	tokenData := map[string][]byte{testData.tokenName: []byte(testData.tokenValue)}
	Expect(createSecret(ctx, testData.sourceSecretName, testData.sourceNamespaceName, tokenData, k8sClient)).To(Succeed())
	Expect(createIdentity(ctx, *testData, k8sClient)).To(Succeed())
}

func uniqueStr(name string) string {
	return name + "-" + uuid.NewString()[:8]
}
