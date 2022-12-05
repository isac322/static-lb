package application

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ServiceRepository interface {
	AssignIPs(ctx context.Context, svc corev1.Service, target IPs) error
}

type EndpointSliceRepository interface {
	ListLinkedTo(ctx context.Context, svcKey types.NamespacedName) (discoveryv1.EndpointSliceList, error)
}

type NodeRepository interface {
	GetByName(ctx context.Context, name string) (corev1.Node, error)
	ListReady(ctx context.Context) ([]corev1.Node, error)
}
