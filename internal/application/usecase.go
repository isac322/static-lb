package application

import (
	"context"
	"net"

	"github.com/isac322/static-lb/internal/pkg/slices"

	corev1 "k8s.io/api/core/v1"
)

type Usecase interface {
	AssignIPs(ctx context.Context, svc corev1.Service) error
}

type usecase struct {
	endpointSliceRepo               EndpointSliceRepository
	nodeRepo                        NodeRepository
	serviceRepo                     ServiceRepository
	defaultInternalIPMappings       []IPMappingTarget
	defaultExternalIPMappings       []IPMappingTarget
	defaultIncludeIngressIPNetwork  []*net.IPNet
	defaultIncludeExternalIPNetwork []*net.IPNet
	defaultExcludeIngressIPNetwork  []*net.IPNet
	defaultExcludeExternalIPNetwork []*net.IPNet
}

func New(
	esr EndpointSliceRepository,
	nr NodeRepository,
	sr ServiceRepository,
	defaultInternalIPMappings []IPMappingTarget,
	defaultExternalIPMappings []IPMappingTarget,
	defaultIncludeIngressIPNetwork []*net.IPNet,
	defaultIncludeExternalIPNetwork []*net.IPNet,
	defaultExcludeIngressIPNetwork []*net.IPNet,
	defaultExcludeExternalIPNetwork []*net.IPNet,
) Usecase {
	return usecase{
		endpointSliceRepo:               esr,
		nodeRepo:                        nr,
		serviceRepo:                     sr,
		defaultInternalIPMappings:       defaultInternalIPMappings,
		defaultExternalIPMappings:       defaultExternalIPMappings,
		defaultIncludeIngressIPNetwork:  defaultIncludeIngressIPNetwork,
		defaultIncludeExternalIPNetwork: defaultIncludeExternalIPNetwork,
		defaultExcludeIngressIPNetwork:  defaultExcludeIngressIPNetwork,
		defaultExcludeExternalIPNetwork: defaultExcludeExternalIPNetwork,
	}
}

func (u usecase) AssignIPs(ctx context.Context, svc corev1.Service) (err error) {
	nodeIPs, err := u.collectNodeIPs(ctx, svc)
	if err != nil {
		return err
	}
	if nodeIPs.IsNone() {
		return nil
	}

	targetIPs := u.mapIPs(nodeIPs.Unwrap())
	targetIPs = u.filterTargetIPs(targetIPs, svc)

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
