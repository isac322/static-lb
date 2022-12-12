package application

type IPMappingTarget string

const (
	IPMappingTargetIngress  IPMappingTarget = "ingress"
	IPMappingTargetExternal IPMappingTarget = "external"
)
