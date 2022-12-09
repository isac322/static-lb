/*
Copyright 2022.

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

package controllers

import (
	"context"
	"github.com/isac322/static-lb/internal/application"
	"github.com/isac322/static-lb/internal/pkg/endpointslice"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Usecase application.Usecase
}

//+kubebuilder:rbac:groups="",resources=services;endpoints;nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslices,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=services;services/status,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Service object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.WithValues("service", req.NamespacedName)

	service, err := r.getService(ctx, req.NamespacedName)
	if err != nil {
		logger.Error(err, "unable to fetch Service")
		return ctrl.Result{}, err
	}

	if err = r.Usecase.AssignIPs(ctx, service); err != nil {
		logger.Error(err, "unable to assign IPs to Service")
		return ctrl.Result{Requeue: true}, err
	}

	logger.Info("updated IPs")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Watches(
			&source.Kind{Type: &discoveryv1.EndpointSlice{}},
			handler.EnqueueRequestsFromMapFunc(r.findLinkedServiceByEndpointSlice),
		).
		Complete(r)
}

func (r *ServiceReconciler) findLinkedServiceByEndpointSlice(endpointSlice client.Object) []reconcile.Request {
	epSlice, ok := endpointSlice.(*discoveryv1.EndpointSlice)
	if !ok {
		return []reconcile.Request{}
	}
	serviceName, err := endpointslice.ServiceKeyForSlice(epSlice)
	if err != nil {
		return []reconcile.Request{}
	}
	return []reconcile.Request{{NamespacedName: serviceName}}
}

func (r *ServiceReconciler) getService(
	ctx context.Context,
	namespacedName types.NamespacedName,
) (corev1.Service, error) {
	var service corev1.Service
	if err := r.Get(ctx, namespacedName, &service); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return corev1.Service{}, client.IgnoreNotFound(err)
	}
	return service, nil
}
