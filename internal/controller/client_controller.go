/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/auth0/go-auth0/management"
	auth0v1alpha1 "github.com/rgracey/auth0-operator/api/v1alpha1"
)

// ClientReconciler reconciles a Client object
type ClientReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Auth0Api *management.Management
}

const (
	EventReasonCreated      = "Created"
	EventReasonCreateFailed = "CreateFailed"
	EventReasonUpdated      = "Updated"
	EventReasonUpdateFailed = "UpdateFailed"
	EventReasonDeleted      = "Deleted"
	EventReasonDeleteFailed = "DeleteFailed"
)

//+kubebuilder:rbac:groups=auth0.gracey.io,resources=clients,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=auth0.gracey.io,resources=clients/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=auth0.gracey.io,resources=clients/finalizers,verbs=update
//+kubebuilder:rbac:groups=auth0.gracey.io,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Client object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *ClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Client instance
	instance := &auth0v1alpha1.Client{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.addFinalizer(instance); err != nil {
		return ctrl.Result{}, err
	}

	// Check if the Client instance is being deleted, and run finalizer logic
	if instance.IsBeingDeleted() {
		logger.Info("deleting client", "name", instance.Spec.Name)
		return ctrl.Result{}, r.handleFinalizer(ctx, instance)
	}

	var clientSecret, err = r.maybeLoadSecretValue(ctx, instance)

	if err != nil {
		return ctrl.Result{}, err
	}

	// callbackUrls must be non-nil
	if instance.Spec.CallbackUrls == nil {
		instance.Spec.CallbackUrls = []string{}
	}

	c := &management.Client{
		Name:           &instance.Spec.Name,
		Description:    &instance.Spec.Description,
		AppType:        &instance.Spec.Type,
		Callbacks:      &instance.Spec.CallbackUrls,
		ClientMetadata: &map[string]interface{}{},
		ClientSecret:   clientSecret,
	}

	for k, v := range instance.Spec.Metadata {
		(*c.ClientMetadata)[k] = v
	}

	// Create the Client if it doesn't exist
	if instance.Auth0Id() == "" {
		logger.Info("creating client", "name", instance.Spec.Name)
		err := r.Auth0Api.Client.Create(ctx, c)

		if err != nil {
			logger.Error(err, "unable to create client", "name", instance.Spec.Name)
			r.Recorder.Event(instance, "Warning", EventReasonCreateFailed, err.Error())
			return ctrl.Result{}, err
		}

		logger.Info("created client", "name", instance.Spec.Name, "Auth0 id", c.GetClientID())

		instance.Status.Auth0Id = c.GetClientID()
		apiErr := r.Status().Update(ctx, instance)

		if apiErr != nil {
			logger.Error(apiErr, "unable to update client status", "name", instance.Spec.Name)

			logger.Info("deleting client", "name", instance.Spec.Name, "Auth0 id", instance.Status.Auth0Id)
			err = r.Auth0Api.Client.Delete(ctx, instance.Status.Auth0Id)

			if err != nil {
				logger.Error(err, "unable to delete client", "name", instance.Spec.Name)
				r.Recorder.Event(instance, "Warning", EventReasonDeleteFailed, err.Error())
			}

			return ctrl.Result{}, apiErr
		}

		r.Recorder.Event(
			instance,
			"Normal",
			EventReasonCreated,
			fmt.Sprintf(
				"Created client %s (ID: %s)",
				instance.Spec.Name,
				instance.Status.Auth0Id,
			),
		)

		return ctrl.Result{Requeue: true}, nil
	}

	c, err = r.Auth0Api.Client.Read(ctx, instance.Status.Auth0Id)

	if err != nil {
		logger.Error(err, "unable to fetch client", "name", instance.Spec.Name)
		r.Recorder.Event(instance, "Warning", EventReasonUpdateFailed, err.Error())
		return ctrl.Result{}, err
	}

	if instance.ShouldOutputSecret() {
		if err := r.upsertOutputSecret(ctx, instance, c.GetClientSecret()); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Auth0 doesn't allow updating these fields
	c.ClientID = nil
	c.SigningKeys = nil
	c.JWTConfiguration.SecretEncoded = nil

	// Move Client to the desired state
	err = r.Auth0Api.Client.Update(ctx, instance.Auth0Id(), c)

	if err != nil {
		logger.Error(err, "unable to update client", "name", instance.Spec.Name)
		r.Recorder.Event(instance, "Warning", EventReasonUpdateFailed, err.Error())
		apiErr := r.Status().Update(context.Background(), instance)

		if apiErr != nil {
			logger.Error(apiErr, "unable to update client status", "name", instance.Spec.Name)
		}

		return ctrl.Result{}, err
	}

	// TODO - Raise Updated event

	return ctrl.Result{}, nil
}

// maybeLoadSecretValue attempts to load a secret from a secret first,
// then a literal value if the secret doesn't exist
func (r *ClientReconciler) maybeLoadSecretValue(
	ctx context.Context,
	instance *auth0v1alpha1.Client,
) (*string, error) {
	if instance.Spec.ClientSecret.SecretRef.Name != "" {
		secretRef := instance.Spec.ClientSecret.SecretRef

		secret := &corev1.Secret{}
		err := r.Get(
			ctx,
			client.ObjectKey{
				Namespace: instance.Namespace,
				Name:      secretRef.Name,
			},
			secret,
		)

		if err != nil {
			return nil, err
		}

		value, ok := secret.Data[secretRef.Key]

		if !ok {
			return nil, fmt.Errorf(
				"clientSecret \"%s\" secretRef didn't contain key \"%s\"",
				secretRef.Name,
				secretRef.Key,
			)
		}

		valueStr := string(value)
		return &valueStr, nil
	}

	if instance.Spec.ClientSecret.Literal != "" {
		return &instance.Spec.ClientSecret.Literal, nil
	}

	return nil, nil
}

// upsertOutputSecret creates or updates the output secret for a client
func (r *ClientReconciler) upsertOutputSecret(
	ctx context.Context,
	instance *auth0v1alpha1.Client,
	clientSecret string,
) error {
	secretRef := instance.Spec.ClientSecret.OutputSecretRef

	secret := &corev1.Secret{
		ObjectMeta: ctrl.ObjectMeta{
			Namespace: instance.Namespace,
			Name:      secretRef.Name,
		},
		StringData: map[string]string{
			secretRef.Key: clientSecret,
		},
	}

	err := r.Get(
		ctx,
		client.ObjectKey{
			Namespace: instance.Namespace,
			Name:      secretRef.Name,
		},
		secret,
	)

	if err == nil {
		secret.StringData = map[string]string{}
		secret.StringData[secretRef.Key] = clientSecret
		return r.Update(ctx, secret)
	}

	if client.IgnoreNotFound(err) == nil {
		if err = ctrl.SetControllerReference(instance, secret, r.Scheme); err != nil {
			return err
		}

		return r.Create(ctx, secret)
	}

	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&auth0v1alpha1.Client{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
