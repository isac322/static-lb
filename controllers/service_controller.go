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
	"errors"
	"fmt"
	"github.com/moznion/go-optional"

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

const SlicesServiceIndexName = "ServiceName"

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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

	if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
		logger.Info("ignore non-LoadBalancer")
		return ctrl.Result{}, nil
	}

	var slices discoveryv1.EndpointSliceList
	if err = r.List(
		ctx,
		&slices,
		client.MatchingFields{SlicesServiceIndexName: req.NamespacedName.String()},
	); err != nil {
		return ctrl.Result{}, err
	}

	nodes, err := r.getNodes(ctx, slices)
	if err != nil {
		logger.Error(err, "unable to fetch Node from EndpointSlice")
		return ctrl.Result{}, err
	} else if len(nodes) == 0 {
		logger.Info("ignore empty service")
		return ctrl.Result{}, nil
	}

	newSvc, err := AssignIPs(ctx, r.Client, service, nodes)
	if err != nil {
		logger.Error(err, "unable to addresses to Service")
		return ctrl.Result{Requeue: true}, err
	}

	if err = r.Update(ctx, &newSvc); err != nil {
		logger.Error(err, "unable to update Service")
		return ctrl.Result{Requeue: true}, err
	}
	if err = r.Status().Update(ctx, &newSvc); err != nil {
		logger.Error(err, "unable to update status of Service")
		return ctrl.Result{Requeue: true}, err
	}
	logger.Info("updated succeed")
	//logger.Info(
	//	fmt.Sprintf("service_name: %s", service.Name),
	//	"slices",
	//	slices,
	//	"nodes",
	//	nodes,
	//)
	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Set a field indexer so we can retrieve all the endpoints for a given service.
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&discoveryv1.EndpointSlice{},
		SlicesServiceIndexName,
		func(rawObj client.Object) []string {
			epSlice, ok := rawObj.(*discoveryv1.EndpointSlice)
			if epSlice == nil || !ok {
				return nil
			}
			serviceKey, err := r.serviceKeyForSlice(epSlice)
			if err != nil {
			}
			return []string{serviceKey.String()}
		},
	); err != nil {
		return err
	}

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
	serviceName, err := r.serviceKeyForSlice(epSlice)
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

func (r *ServiceReconciler) serviceKeyForSlice(endpointSlice *discoveryv1.EndpointSlice) (types.NamespacedName, error) {
	if endpointSlice == nil {
		return types.NamespacedName{}, fmt.Errorf("nil EndpointSlice")
	}
	serviceName, err := serviceNameForSlice(endpointSlice)
	if err != nil {
		return types.NamespacedName{}, err
	}

	return types.NamespacedName{Namespace: endpointSlice.Namespace, Name: serviceName}, nil
}

func (r *ServiceReconciler) getNodes(ctx context.Context, slices discoveryv1.EndpointSliceList) ([]corev1.Node, error) {
	logger := log.FromContext(ctx)

	nodeMap := map[string]corev1.Node{}

	for _, slice := range slices.Items {
		for _, endpoint := range slice.Endpoints {
			if endpoint.NodeName == nil {
				// TODO
				logger.Error(errors.New("empty NodeName endpoint"), "empty NodeName endpoint")
				continue
			}

			nodeName := *endpoint.NodeName
			if _, exists := nodeMap[nodeName]; exists {
				continue
			}

			var node corev1.Node
			if err := r.Get(ctx, types.NamespacedName{Name: nodeName}, &node); err != nil {
				logger.Error(err, "unable to fetch Node")
				continue
			}
			nodeMap[nodeName] = node
		}
	}

	var result []corev1.Node
	for _, node := range nodeMap {
		result = append(result, node)
	}
	return result, nil
}

func serviceNameForSlice(endpointSlice *discoveryv1.EndpointSlice) (string, error) {
	serviceName, ok := endpointSlice.Labels[discoveryv1.LabelServiceName]
	if !ok || serviceName == "" {
		return "", fmt.Errorf("endpointSlice missing the %s label", discoveryv1.LabelServiceName)
	}
	return serviceName, nil
}

type IPAddressPair struct {
	InternalIP optional.Option[string]
	ExternalIP optional.Option[string]
}

type AssignResult int

const (
	AssignSucceed AssignResult = iota
	AssignNotRequired
	AssignFailed
)

func AssignIPs(
	ctx context.Context,
	cli client.Client,
	svc corev1.Service,
	nodes []corev1.Node,
) (corev1.Service, error) {
	addresses := CollectIPs(nodes)
	if len(addresses) == 0 {
		return corev1.Service{}, errors.New("unable to find node ips")
	}

	var ingIPs []corev1.LoadBalancerIngress
	var extIPs []string
	for _, address := range addresses {
		if address.InternalIP.IsSome() {
			ingIPs = append(ingIPs, corev1.LoadBalancerIngress{IP: address.InternalIP.Unwrap()})
		}
		if address.ExternalIP.IsSome() {
			extIPs = append(extIPs, address.ExternalIP.Unwrap())
		}
	}
	svc.Spec.ExternalIPs = extIPs
	svc.Status.LoadBalancer.Ingress = ingIPs

	return svc, nil
}

func CollectIPs(nodes []corev1.Node) []IPAddressPair {
	if len(nodes) == 0 {
		return nil
	}

	result := make([]IPAddressPair, len(nodes))

	for i, node := range nodes {
		pair := &result[i]

		for _, address := range node.Status.Addresses {
			switch address.Type {
			case corev1.NodeExternalIP:
				pair.ExternalIP = optional.Some(address.Address)
			case corev1.NodeInternalIP:
				pair.InternalIP = optional.Some(address.Address)
			}
		}
	}

	return result
}
