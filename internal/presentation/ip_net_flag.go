package presentation

import (
	"net"
	"strings"
)

type IPNetFilterFlag []*net.IPNet

func (f *IPNetFilterFlag) String() string {
	var ss []string
	for _, ipNet := range *f {
		ss = append(ss, ipNet.String())
	}
	return strings.Join(ss, ",")
}

func (f *IPNetFilterFlag) Set(s string) error {
	splitted := strings.Split(s, ",")

	for _, ip := range splitted {
		_, ipNet, err := net.ParseCIDR(ip)
		if err != nil {
			return err
		}
		if ipNet == nil {
			continue
		}

		*f = append(*f, ipNet)
	}

	return nil
}
