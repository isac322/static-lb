package application

func (u usecase) mapIPs(nodeIPs NodeIPs) IPStatus {
	var targetIPs IPStatus

	for _, mapping := range u.defaultInternalIPMappings {
		switch mapping {
		case IPMappingTargetExternal:
			targetIPs.ExternalIPs = append(targetIPs.ExternalIPs, nodeIPs.InternalIPs...)
		case IPMappingTargetIngress:
			targetIPs.IngressIPs = append(targetIPs.IngressIPs, nodeIPs.InternalIPs...)
		default:
			break
		}
	}

	for _, mapping := range u.defaultExternalIPMappings {
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
