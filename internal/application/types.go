package application

type IPStatus struct {
	IngressIPs  []string
	ExternalIPs []string
}

type NodeIPs struct {
	InternalIPs []string
	ExternalIPs []string
}
