package controller

import (
	"context"
	"fmt"

	auth0v1alpha1 "github.com/rgracey/auth0-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	finalizerName = "finalizer.auth0.gracey.io"
)

// hasFinalizer returns true if the Client has the finalizer. False otherwise
func (r *ClientReconciler) hasFinalizer(instance *auth0v1alpha1.Client) bool {
	return controllerutil.ContainsFinalizer(instance, finalizerName)
}

// addFinalizer adds the finalizer to the Client if it doesn't already exist
func (r *ClientReconciler) addFinalizer(instance *auth0v1alpha1.Client) error {
	if r.hasFinalizer(instance) {
		return nil
	}

	controllerutil.AddFinalizer(instance, finalizerName)
	return r.Update(context.Background(), instance)
}

// removeFinalizer removes the finalizer from the Client if it exists
func (r *ClientReconciler) removeFinalizer(instance *auth0v1alpha1.Client) error {
	if !r.hasFinalizer(instance) {
		return nil
	}

	controllerutil.RemoveFinalizer(instance, finalizerName)
	return r.Update(context.Background(), instance)
}

// handleFinalizer handles the finalizer logic for the Client
func (r *ClientReconciler) handleFinalizer(ctx context.Context, instance *auth0v1alpha1.Client) error {
	if !r.hasFinalizer(instance) {
		return nil
	}

	if instance.Auth0Id() == "" {
		return r.removeFinalizer(instance)
	}

	// N.B output secret is deleted via owner reference garbage collection

	err := r.Auth0Api.Client.Delete(ctx, instance.Status.Auth0Id)

	// TODO - better handling here if the client doesn't exist?
	if err != nil {
		r.Recorder.Event(instance, "Warning", EventReasonDeleteFailed, err.Error())
		return err
	}

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

	return r.removeFinalizer(instance)
}
