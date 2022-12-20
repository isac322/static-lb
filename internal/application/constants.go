package application

type IPMappingTarget string

const (
	IPMappingTargetIngress  IPMappingTarget = "ingress"
	IPMappingTargetExternal IPMappingTarget = "external"
)

const (
	LabelIncludeIngressIPNets  = "static-lb.bhyoo.com/include-ingress-ip-nets"
	LabelIncludeExternalIPNets = "static-lb.bhyoo.com/include-external-ip-nets"
	LabelExcludeIngressIPNets  = "static-lb.bhyoo.com/exclude-ingress-ip-nets"
	LabelExcludeExternalIPNets = "static-lb.bhyoo.com/exclude-external-ip-nets"

	LabelInternalIPMappings = "static-lb.bhyoo.com/internal-ip-mappings"
	LabelExternalIPMappings = "static-lb.bhyoo.com/external-ip-mappings"
)
