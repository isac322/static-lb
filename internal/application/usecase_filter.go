package application

import (
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func (u usecase) filterTargetIPs(targetIPs IPStatus, svc corev1.Service) IPStatus {
	ingressIPs := parseIPs(targetIPs.IngressIPs)
	externalIPs := parseIPs(targetIPs.ExternalIPs)

	ingressIPs = filterOutIPs(
		ingressIPs,
		getIPNetFrom(svc, LabelExcludeIngressIPNets, u.defaultExcludeIngressIPNetwork),
	)
	externalIPs = filterOutIPs(
		externalIPs,
		getIPNetFrom(svc, LabelExcludeExternalIPNets, u.defaultExcludeExternalIPNetwork),
	)

	ingressIPs = selectIPs(
		ingressIPs,
		getIPNetFrom(svc, LabelIncludeIngressIPNets, u.defaultIncludeIngressIPNetwork),
	)
	externalIPs = selectIPs(
		externalIPs,
		getIPNetFrom(svc, LabelIncludeExternalIPNets, u.defaultIncludeExternalIPNetwork),
	)

	return IPStatus{
		IngressIPs:  unparseIPs(ingressIPs),
		ExternalIPs: unparseIPs(externalIPs),
	}
}

func getIPNetFrom(svc corev1.Service, annotationName string, defaultVal []*net.IPNet) []*net.IPNet {
	val, exists := svc.Annotations[annotationName]
	if !exists || val == "" {
		return defaultVal
	}

	splitted := strings.Split(strings.TrimSpace(val), ",")
	ipv4Nets := make([]*net.IPNet, 0, len(splitted))
	for _, s := range splitted {
		_, ipNet, err := net.ParseCIDR(s)
		if err != nil {
			continue
		}

		ipv4Nets = append(ipv4Nets, ipNet)
	}

	if len(ipv4Nets) == 0 {
		return defaultVal
	}
	return ipv4Nets
}

func parseIPs(ips []string) []net.IP {
	if len(ips) == 0 {
		return nil
	}

	parsedIPs := make([]net.IP, 0, len(ips))
	for _, ip := range ips {
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			continue
		}
		parsedIPs = append(parsedIPs, parsedIP)
	}
	return parsedIPs
}

func unparseIPs(ips []net.IP) []string {
	if len(ips) == 0 {
		return nil
	}

	result := make([]string, 0, len(ips))
	for _, ip := range ips {
		result = append(result, ip.String())
	}
	return result
}

func filterOutIPs(src []net.IP, ipNets []*net.IPNet) (result []net.IP) {
	if len(ipNets) == 0 {
		return src
	}

outer:
	for _, ip := range src {
		for _, ipNet := range ipNets {
			if ipNet.Contains(ip) {
				continue outer
			}
		}
		result = append(result, ip)
	}

	return result
}

func selectIPs(src []net.IP, ipNets []*net.IPNet) (result []net.IP) {
	if len(ipNets) == 0 {
		return src
	}

	for _, ip := range src {
		for _, ipNet := range ipNets {
			if ipNet.Contains(ip) {
				result = append(result, ip)
				break
			}
		}
	}

	return result
}
