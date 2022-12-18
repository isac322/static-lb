package application

import (
	"context"

	"github.com/isac322/static-lb/internal/pkg/slices"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Usecase interface {
	AssignIPs(ctx context.Context, svc corev1.Service) error
}

type usecase struct {
	endpointSliceRepo         EndpointSliceRepository
	nodeRepo                  NodeRepository
	serviceRepo               ServiceRepository
	defaultInternalIPMappings []IPMappingTarget
	defaultExternalIPMappings []IPMappingTarget
}

func New(
	esr EndpointSliceRepository,
	nr NodeRepository,
	sr ServiceRepository,
	defaultInternalIPMappings []IPMappingTarget,
	defaultExternalIPMappings []IPMappingTarget,
) Usecase {
	return usecase{
		endpointSliceRepo:         esr,
		nodeRepo:                  nr,
		serviceRepo:               sr,
		defaultInternalIPMappings: defaultInternalIPMappings,
		defaultExternalIPMappings: defaultExternalIPMappings,
	}
}

func (u usecase) AssignIPs(ctx context.Context, svc corev1.Service) (err error) {
	var endpointIPs NodeIPs

	switch {
	case svc.Spec.Type != corev1.ServiceTypeLoadBalancer:
		// reset ips
		break
	case svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal:
		if endpointIPs, err = u.getIPsFromEndpointSlice(ctx, svc); err != nil {
			return err
		}
	case svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeCluster:
		endpoints, err := u.getServingEndpointsFromSlice(ctx, svc)
		if err != nil {
			return err
		}
		if len(endpoints) == 0 {
			break
		}

		if endpointIPs, err = u.getIPsFromAllNodes(ctx); err != nil {
			return err
		}
	default:
		// Ignore others
		return nil
	}

	var targetIPs IPStatus
	for _, mapping := range u.defaultInternalIPMappings {
		switch mapping {
		case IPMappingTargetExternal:
			targetIPs.ExternalIPs = append(targetIPs.ExternalIPs, endpointIPs.InternalIPs...)
		case IPMappingTargetIngress:
			targetIPs.IngressIPs = append(targetIPs.IngressIPs, endpointIPs.InternalIPs...)
		default:
			break
		}
	}
	for _, mapping := range u.defaultExternalIPMappings {
		switch mapping {
		case IPMappingTargetExternal:
			targetIPs.ExternalIPs = append(targetIPs.ExternalIPs, endpointIPs.ExternalIPs...)
		case IPMappingTargetIngress:
			targetIPs.IngressIPs = append(targetIPs.IngressIPs, endpointIPs.ExternalIPs...)
		default:
			break
		}
	}

	if u.isSynced(svc, targetIPs) {
		return nil
	}

	return u.serviceRepo.AssignIPs(ctx, svc, targetIPs)
}

func (u usecase) isSynced(svc corev1.Service, targetIPs IPStatus) bool {
	origIngressIPs := make([]string, len(svc.Status.LoadBalancer.Ingress))
	for i, ingress := range svc.Status.LoadBalancer.Ingress {
		origIngressIPs[i] = ingress.IP
	}
	return slices.Match(targetIPs.ExternalIPs, svc.Spec.ExternalIPs) &&
		slices.Match(targetIPs.IngressIPs, origIngressIPs)
}

func (u usecase) getIPsFromAllNodes(ctx context.Context) (NodeIPs, error) {
	nodes, err := u.nodeRepo.ListReady(ctx)
	if err != nil {
		return NodeIPs{}, err
	}

	return u.extractIPsFrom(nodes), nil
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

func (u usecase) getServingEndpointsFromSlice(ctx context.Context, svc corev1.Service) ([]discoveryv1.Endpoint, error) {
	endpointSliceList, err := u.endpointSliceRepo.ListLinkedTo(ctx, types.NamespacedName{
		Namespace: svc.Namespace,
		Name:      svc.Name,
	})
	if err != nil {
		return nil, err
	}

	var endpoints []discoveryv1.Endpoint
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
