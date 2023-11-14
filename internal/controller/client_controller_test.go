package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	auth0v1alpha1 "github.com/rgracey/auth0-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Client controller", func() {
	key := types.NamespacedName{
		Name:      "test-client",
		Namespace: "default",
	}

	Context("When a client is created", func() {
		It("Should create a client in Auth0", func() {
			client := &auth0v1alpha1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: auth0v1alpha1.ClientSpec{
					Name:        "test-suite-client",
					Type:        "spa",
					Description: "A client created by the test suite",
				},
			}

			By("Creating a client in Auth0")
			Expect(k8sClient.Create(ctx, client)).To(Succeed())

			By("Updating the status to hold the Auth0 ID")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, client)
				if err != nil {
					return false
				}
				return client.Status.Auth0Id != ""
			}).WithTimeout(3 * time.Second).Should(BeTrue())

			By("Checking that the client exists in Auth0")
			c, err := auth0Api.Client.Read(ctx, client.Status.Auth0Id)
			Expect(err).To(BeNil())
			Expect(c).ToNot(BeNil())
			Expect(*c.Name).To(Equal(client.Spec.Name))
			Expect(*c.Description).To(Equal(client.Spec.Description))
			Expect(*c.AppType).To(Equal(client.Spec.Type))
			Expect(*c.ClientSecret).ToNot(BeEmpty())

			By("Deleting the client")
			Expect(k8sClient.Delete(ctx, client)).To(Succeed())

			By("Checking that the client is deleted in Auth0")
			Eventually(func() bool {
				_, err := auth0Api.Client.Read(ctx, client.Status.Auth0Id)
				if err == nil {
					return false
				}

				Expect(err.Error()).To(ContainSubstring("Not Found"))
				return true
			}).WithTimeout(3 * time.Second).Should(BeTrue())
		})
	})
})
