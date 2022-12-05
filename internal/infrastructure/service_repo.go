package infrastructure

import (
	"context"
	"github.com/isac322/static-lb/internal/application"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8sClientServiceRepository struct {
	k8sClient client.Client
}

func NewServiceRepository(cli client.Client) K8sClientServiceRepository {
	return K8sClientServiceRepository{
		k8sClient: cli,
	}
}

func (k K8sClientServiceRepository) AssignIPs(ctx context.Context, svc corev1.Service, target application.IPs) error {
	newSvc := svc.DeepCopy()
	newSvc.Spec.ExternalIPs = target.ExternalIPs

	if err := k.k8sClient.Update(ctx, newSvc); err != nil {
		return err
	}

	newSvc.Status.LoadBalancer.Ingress = make([]corev1.LoadBalancerIngress, len(target.IngressIPs))
	for i, ip := range target.IngressIPs {
		newSvc.Status.LoadBalancer.Ingress[i].IP = ip
	}

	if err := k.k8sClient.Status().Update(ctx, newSvc); err != nil {
		return err
	}

	return nil
}
