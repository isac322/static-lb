package application

import (
	"context"

	"github.com/isac322/static-lb/internal/pkg/optional"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (u usecase) collectNodeIPs(ctx context.Context, svc corev1.Service) (optional.Option[NodeIPs], error) {
	switch {
	case svc.Spec.Type != corev1.ServiceTypeLoadBalancer:
		// reset ips
		return optional.Some(NodeIPs{}), nil

	case svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal:
		nodeIPs, err := u.getIPsFromEndpointSlice(ctx, svc)
		if err != nil {
			return optional.None[NodeIPs](), err
		}
		return optional.Some(nodeIPs), nil

	case svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeCluster:
		endpoints, err := u.getServingEndpointsFromSlice(ctx, svc)
		if err != nil {
			return optional.None[NodeIPs](), err
		}
		if len(endpoints) == 0 {
			return optional.Some(NodeIPs{}), nil
		}

		nodeIPs, err := u.getIPsFromAllNodes(ctx)
		if err != nil {
			return optional.None[NodeIPs](), err
		}
		return optional.Some(nodeIPs), nil

	default:
		// Ignore others
		return optional.None[NodeIPs](), nil
	}
}

func (u usecase) getIPsFromEndpointSlice(ctx context.Context, svc corev1.Service) (NodeIPs, error) {
	endpoints, err := u.getServingEndpointsFromSlice(ctx, svc)
	if err != nil {
		return NodeIPs{}, err
	}

	nodes := make([]corev1.Node, 0, len(endpoints))
	for _, endpoint := range endpoints {
		node, err := u.nodeRepo.GetByName(ctx, *endpoint.NodeName)
		if err != nil {
			return NodeIPs{}, err
		}
		nodes = append(nodes, node)
	}

	return u.extractIPsFrom(nodes), err
}

func (u usecase) getServingEndpointsFromSlice(ctx context.Context, svc corev1.Service) ([]v1.Endpoint, error) {
	endpointSliceList, err := u.endpointSliceRepo.ListLinkedTo(ctx, types.NamespacedName{
		Namespace: svc.Namespace,
		Name:      svc.Name,
	})
	if err != nil {
		return nil, err
	}

	var endpoints []v1.Endpoint
	for _, endpointSlice := range endpointSliceList.Items {
		for _, endpoint := range endpointSlice.Endpoints {
			if endpoint.Conditions.Serving == nil || !*endpoint.Conditions.Serving || endpoint.NodeName == nil {
				continue
			}

			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints, nil
}

func (u usecase) getIPsFromAllNodes(ctx context.Context) (NodeIPs, error) {
	nodes, err := u.nodeRepo.ListReady(ctx)
	if err != nil {
		return NodeIPs{}, err
	}

	return u.extractIPsFrom(nodes), nil
}

func (u usecase) extractIPsFrom(nodes []corev1.Node) (result NodeIPs) {
	for _, node := range nodes {
		for _, address := range node.Status.Addresses {
			switch address.Type {
			case corev1.NodeInternalIP:
				result.InternalIPs = append(result.InternalIPs, address.Address)
			case corev1.NodeExternalIP:
				result.ExternalIPs = append(result.ExternalIPs, address.Address)
			}
		}
	}
	return result
}
