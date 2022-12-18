package presentation

import (
	"fmt"
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
	_, ipNet, err := net.ParseCIDR(s)
	if err != nil {
		return err
	}
	if ipNet == nil {
		return fmt.Errorf("invalid IP network: %s", s)
	}

	*f = append(*f, ipNet)
	return nil
}
