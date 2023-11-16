package controller

import (
	"context"
	"time"

	"github.com/auth0/go-auth0/management"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	auth0v1alpha1 "github.com/rgracey/auth0-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Timeout for eventually assertions
const timeout = time.Second * 7

var _ = Describe("Client controller", func() {
	var key types.NamespacedName

	var client *auth0v1alpha1.Client

	var auth0Client *management.Client

	BeforeEach(func() {
		key = types.NamespacedName{
			Name:      "test-client-" + time.Now().Format("20060102150405"),
			Namespace: "default",
		}
		client = &auth0v1alpha1.Client{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: auth0v1alpha1.ClientSpec{
				Name:        "test-suite-client",
				Type:        "spa",
				Description: "A client created by the test suite",
				Metadata: map[string]string{
					"test": "test",
				},
			},
		}
	})

	Describe("when a client is created", func() {
		AfterEach(func() {
			// Delete the client
			Expect(k8sClient.Delete(context.Background(), client)).To(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, &auth0v1alpha1.Client{})
				return ctrlclient.IgnoreNotFound(err) == nil
			}).WithTimeout(timeout).Should(BeTrue())
		})

		JustBeforeEach(func() {
			// Create he client in the cluster
			Expect(k8sClient.Create(ctx, client)).To(Succeed())

			// Wait for the Auth0 ID to be populated and finalizer to be added
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, key, client); err != nil {
					return false
				}
				return client.Status.Auth0Id != "" && controllerutil.ContainsFinalizer(client, finalizerName)
			}).WithTimeout(timeout).Should(BeTrue())

			// Get the Auth0 client
			var err error
			auth0Client, err = auth0Api.Client.Read(ctx, client.Status.Auth0Id)
			Expect(err).To(BeNil())
			Expect(auth0Client).ToNot(BeNil())
		})

		It("should create a client in Auth0 with the correct values", func() {
			Expect(*auth0Client.Name).To(Equal(client.Spec.Name))
			Expect(*auth0Client.Description).To(Equal(client.Spec.Description))
			Expect(*auth0Client.AppType).To(Equal(client.Spec.Type))
			// Expect(*c.ClientMetadata).To(ConsistOf(client.Spec.Metadata))
		})

		When("a secret is provided", func() {
			const expectedSecret = "ThisIsA48CharacterSecretSoItIsLongEnoughForAuth0"

			Describe("as a literal value", func() {
				BeforeEach(func() {
					client.Spec.ClientSecret = auth0v1alpha1.ClientSecret{
						Literal: expectedSecret,
					}
				})

				It("should create a client in Auth0 with the provided secret", func() {
					Expect(*auth0Client.ClientSecret).To(Equal(expectedSecret))
				})
			})

			Describe("as a secret reference", func() {
				var secret *corev1.Secret

				BeforeEach(func() {
					secret = &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: key.Namespace,
						},
						StringData: map[string]string{
							"test-key": expectedSecret,
						},
					}

					Expect(k8sClient.Create(ctx, secret)).To(Succeed())
				})

				AfterEach(func() {
					Expect(k8sClient.Delete(context.Background(), secret)).To(Succeed())
				})

				When("the secret exists and the key exists", func() {
					BeforeEach(func() {
						client.Spec.ClientSecret = auth0v1alpha1.ClientSecret{
							SecretRef: auth0v1alpha1.SecretRef{
								Name: "test-secret",
								Key:  "test-key",
							},
						}
					})

					It("should create a client in Auth0 with the provided secret", func() {
						Expect(*auth0Client.ClientSecret).To(Equal(expectedSecret))
					})
				})

				When("the key doesn't exist in the secret", func() {
					// Currently causes suite to fail (because it expects the
					// client to always create successfully)

					// BeforeEach(func() {
					// 	client.Spec.ClientSecret = auth0v1alpha1.ClientSecret{
					// 		SecretRef: auth0v1alpha1.SecretRef{
					// 			Name: "test-secret",
					// 			Key:  "non-existent-key",
					// 		},
					// 	}
					// })

					// It("should not create a client in Auth0", func() {
					// 	Consistently(func() bool {
					// 		c := &auth0v1alpha1.Client{}
					// 		err := k8sClient.Get(ctx, key, c)

					// 		if err != nil {
					// 			return false
					// 		}

					// 		return c.Status.Auth0Id == ""
					// 	})
					// })
				})
			})
		})

		When("no secret is provided", func() {
			It("should create a client in Auth0 with a generated secret", func() {
				Expect(*auth0Client.ClientSecret).ToNot(BeEmpty())
			})
		})

		When("an output secret is specified", func() {
			var outputSecretName string
			const outputSecretKey = "test-key"

			BeforeEach(func() {
				outputSecretName = "test-output-secret-" + time.Now().Format("20060102150405")
				client.Spec.ClientSecret = auth0v1alpha1.ClientSecret{
					OutputSecretRef: auth0v1alpha1.SecretRef{
						Name: outputSecretName,
						Key:  outputSecretKey,
					},
				}
			})

			AfterEach(func() {
				// Delete the output secret
				Expect(k8sClient.Delete(context.Background(), &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      outputSecretName,
						Namespace: key.Namespace,
					},
				})).To(Succeed())
			})

			It("should create a secret in the cluster", func() {
				By("Checking that the secret exists in the cluster")
				Eventually(func() bool {
					s := &corev1.Secret{}
					err := k8sClient.Get(
						ctx,
						types.NamespacedName{
							Namespace: key.Namespace,
							Name:      outputSecretName,
						},
						s,
					)

					if err != nil {
						return false
					}

					return string(s.Data[outputSecretKey]) != ""
				}).WithTimeout(timeout).Should(BeTrue())

				By("checking that the secret has an owner reference")
				Eventually(func() bool {
					secret := &corev1.Secret{}
					err := k8sClient.Get(
						ctx,
						types.NamespacedName{
							Namespace: key.Namespace,
							Name:      outputSecretName,
						},
						secret,
					)

					if err != nil {
						return false
					}

					return len(secret.OwnerReferences) > 0
				}).WithTimeout(timeout).Should(BeTrue())
			})

			When("the output secret already exists", func() {
				BeforeEach(func() {
					secret := &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      outputSecretName,
							Namespace: key.Namespace,
						},
						StringData: map[string]string{
							outputSecretKey: "old-value",
						},
					}

					Expect(k8sClient.Create(ctx, secret)).To(Succeed())
				})

				It("should update the secret", func() {
					Eventually(func() bool {
						secret := &corev1.Secret{}
						err := k8sClient.Get(
							ctx,
							types.NamespacedName{
								Namespace: key.Namespace,
								Name:      outputSecretName,
							},
							secret,
						)

						if err != nil {
							return false
						}

						return string(secret.Data[outputSecretKey]) != "old-value"
					}).WithTimeout(timeout).Should(BeTrue())
				})
			})
		})
	})

	Describe("when a client is deleted", func() {
		var client *auth0v1alpha1.Client

		BeforeEach(func() {
			client = &auth0v1alpha1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: auth0v1alpha1.ClientSpec{
					Name: "test-suite-client",
					Type: "spa",
				},
			}
		})

		JustBeforeEach(func() {
			Expect(k8sClient.Create(ctx, client)).To(Succeed())
		})

		It("should delete the client in Auth0", func() {
			Expect(k8sClient.Delete(context.Background(), client)).To(Succeed())

			// Check that the client is deleted in Auth0
			Eventually(func() bool {
				_, err := auth0Api.Client.Read(ctx, client.Status.Auth0Id)
				if err == nil {
					return false
				}

				Expect(err.Error()).To(ContainSubstring("Not Found"))
				return true
			}).WithTimeout(timeout).WithPolling(1 * time.Second).Should(BeTrue())
		})
	})
})
