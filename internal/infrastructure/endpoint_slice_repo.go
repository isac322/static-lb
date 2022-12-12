package infrastructure

import (
	"context"

	"github.com/isac322/static-lb/internal/pkg/endpointslice"

	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	endpointSliceServiceIndexName = "ServiceName"
)

type K8sClientEndpointSliceRepository struct {
	k8sClient client.Client
}

func NewEndpointSliceRepository(cli client.Client) K8sClientEndpointSliceRepository {
	return K8sClientEndpointSliceRepository{
		k8sClient: cli,
	}
}

func (k K8sClientEndpointSliceRepository) ListLinkedTo(
	ctx context.Context,
	svcKey types.NamespacedName,
) (slices discoveryv1.EndpointSliceList, err error) {
	if err = k.k8sClient.List(
		ctx,
		&slices,
		client.MatchingFields{endpointSliceServiceIndexName: svcKey.String()},
	); err != nil {
		return discoveryv1.EndpointSliceList{}, err
	}

	return slices, nil
}

func (k K8sClientEndpointSliceRepository) RegisterFieldIndex(ctx context.Context, indexer client.FieldIndexer) error {
	return indexer.IndexField(
		ctx,
		&discoveryv1.EndpointSlice{},
		endpointSliceServiceIndexName,
		func(rawObj client.Object) []string {
			epSlice, ok := rawObj.(*discoveryv1.EndpointSlice)
			if epSlice == nil || !ok {
				return nil
			}
			serviceKey, err := endpointslice.ServiceKeyForSlice(epSlice)
			if err != nil {
				return nil
			}
			return []string{serviceKey.String()}
		},
	)
}
