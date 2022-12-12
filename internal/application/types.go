package application

type IPStatus struct {
	IngressIPs  []string
	ExternalIPs []string
}

type EndpointIPs struct {
	InternalIPs []string
	ExternalIPs []string
}
