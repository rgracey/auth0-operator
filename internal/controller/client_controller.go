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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	finalizerName = "finalizer.auth0.gracey.io"
)

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

	// Check if the Client instance is marked to be deleted,
	// and run finalizer logic
	isMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isMarkedToBeDeleted {
		logger.Info("deleting client", "name", instance.Spec.Name)

		if controllerutil.ContainsFinalizer(instance, finalizerName) {
			if instance.Status.Auth0Id == "" {
				logger.Info(
					"Auth0 ID not present. Skipping deletion",
					"name", instance.Spec.Name,
				)

				controllerutil.RemoveFinalizer(instance, finalizerName)
				if err := r.Update(ctx, instance); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{}, nil
			}

			err := r.Auth0Api.Client.Delete(ctx, instance.Status.Auth0Id)

			// TODO - better handling here if the client doesn't exist?
			if err != nil {
				logger.Error(
					err,
					"unable to delete client",
					"name", instance.Spec.Name,
					"Auth0 id", instance.Status.Auth0Id,
				)
				r.Recorder.Event(instance, "Warning", EventReasonDeleteFailed, err.Error())
				return ctrl.Result{}, err
			}

			logger.Info(
				"deleted client",
				"name", instance.Spec.Name,
				"Auth0 id", instance.Status.Auth0Id,
			)

			r.Recorder.Event(
				instance,
				"Normal",
				EventReasonDeleted,
				fmt.Sprintf(
					"Deleted client %s (ID: %s)",
					instance.Spec.Name,
					instance.Status.Auth0Id,
				),
			)

			controllerutil.RemoveFinalizer(instance, finalizerName)
			if err := r.Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}
	} else {
		// Add finalizer if it doesn't exist
		if !controllerutil.ContainsFinalizer(instance, finalizerName) {
			controllerutil.AddFinalizer(instance, finalizerName)
			if err := r.Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	c := &management.Client{
		Name:           &instance.Spec.Name,
		Description:    &instance.Spec.Description,
		AppType:        &instance.Spec.Type,
		Callbacks:      &instance.Spec.CallbackUrls,
		ClientMetadata: &map[string]interface{}{},
	}

	for k, v := range instance.Spec.Metadata {
		(*c.ClientMetadata)[k] = v
	}

	// Create the Client if it doesn't exist
	if instance.Status.Auth0Id == "" {
		logger.Info("creating client", "name", instance.Spec.Name)
		err := r.Auth0Api.Client.Create(ctx, c)

		if err != nil {
			logger.Error(err, "unable to create client", "name", instance.Spec.Name)
			r.Recorder.Event(instance, "Warning", EventReasonCreateFailed, err.Error())
			return ctrl.Result{}, err
		}

		logger.Info("created client", "name", instance.Spec.Name, "Auth0 id", c.GetClientID())

		instance.Status.Auth0Id = c.GetClientID()
		apiErr := r.Status().Update(context.Background(), instance)

		if apiErr != nil {
			logger.Error(apiErr, "unable to update client status", "name", instance.Spec.Name)
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

		return ctrl.Result{}, nil
	}

	// Move Client to the desired state
	err := r.Auth0Api.Client.Update(ctx, instance.Status.Auth0Id, c)

	if err != nil {
		logger.Error(err, "unable to update client", "name", instance.Spec.Name)
		r.Recorder.Event(instance, "Warning", EventReasonUpdateFailed, err.Error())
		r.Status().Update(context.Background(), instance)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&auth0v1alpha1.Client{}).
		Complete(r)
}
