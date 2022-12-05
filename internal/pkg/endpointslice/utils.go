package endpointslice

import (
	"fmt"

	"k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
)

func ServiceKeyForSlice(
	endpointSlice *v1.EndpointSlice,
) (types.NamespacedName, error) {
	if endpointSlice == nil {
		return types.NamespacedName{}, fmt.Errorf("nil EndpointSlice")
	}
	serviceName, err := serviceNameForSlice(endpointSlice)
	if err != nil {
		return types.NamespacedName{}, err
	}

	return types.NamespacedName{Namespace: endpointSlice.Namespace, Name: serviceName}, nil
}

func serviceNameForSlice(endpointSlice *v1.EndpointSlice) (string, error) {
	serviceName, ok := endpointSlice.Labels[v1.LabelServiceName]
	if !ok || serviceName == "" {
		return "", fmt.Errorf("endpointSlice missing the %s label", v1.LabelServiceName)
	}
	return serviceName, nil
}
