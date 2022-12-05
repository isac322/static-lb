package infrastructure

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8sClientNodeRepository struct {
	k8sClient client.Client
}

func NewNodeRepository(cli client.Client) K8sClientNodeRepository {
	return K8sClientNodeRepository{
		k8sClient: cli,
	}
}

func (k K8sClientNodeRepository) GetByName(ctx context.Context, name string) (node corev1.Node, err error) {
	if err = k.k8sClient.Get(ctx, types.NamespacedName{Name: name}, &node); err != nil {
		return corev1.Node{}, err
	}

	return node, nil
}

func (k K8sClientNodeRepository) ListReady(ctx context.Context) ([]corev1.Node, error) {
	var nodeList corev1.NodeList
	if err := k.k8sClient.List(ctx, &nodeList); err != nil {
		return nil, err
	}

	nodes := make([]corev1.Node, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type != corev1.NodeReady || condition.Status != corev1.ConditionTrue {
				continue
			}

			nodes = append(nodes, node)
			break
		}
	}

	return nodes, nil
}
