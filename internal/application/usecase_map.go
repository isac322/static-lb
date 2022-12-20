package application

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func (u usecase) mapIPs(nodeIPs NodeIPs, svc corev1.Service) IPStatus {
	var targetIPs IPStatus

	for _, mapping := range getMappings(svc, LabelInternalIPMappings, u.defaultInternalIPMappings) {
		switch mapping {
		case IPMappingTargetExternal:
			targetIPs.ExternalIPs = append(targetIPs.ExternalIPs, nodeIPs.InternalIPs...)
		case IPMappingTargetIngress:
			targetIPs.IngressIPs = append(targetIPs.IngressIPs, nodeIPs.InternalIPs...)
		default:
			break
		}
	}

	for _, mapping := range getMappings(svc, LabelExternalIPMappings, u.defaultExternalIPMappings) {
		switch mapping {
		case IPMappingTargetExternal:
			targetIPs.ExternalIPs = append(targetIPs.ExternalIPs, nodeIPs.ExternalIPs...)
		case IPMappingTargetIngress:
			targetIPs.IngressIPs = append(targetIPs.IngressIPs, nodeIPs.ExternalIPs...)
		default:
			break
		}
	}

	return targetIPs
}

func getMappings(svc corev1.Service, annotationName string, defaultVal []IPMappingTarget) []IPMappingTarget {
	annotation, exists := svc.Annotations[annotationName]
	if !exists {
		return defaultVal
	}
	if annotation == "" {
		return nil
	}

	splitted := strings.Split(annotation, ",")
	result := make([]IPMappingTarget, 0, len(splitted))
	for _, s := range splitted {
		switch IPMappingTarget(s) {
		case IPMappingTargetExternal:
			result = append(result, IPMappingTargetExternal)
		case IPMappingTargetIngress:
			result = append(result, IPMappingTargetIngress)
		default:
			continue
		}
	}

	return result
}
