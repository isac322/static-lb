package application

import (
	"net"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

func TestUsecase_filterTargetIPs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                            string
		targetIPs                       IPStatus
		svc                             corev1.Service
		defaultIncludeIngressIPNetwork  []*net.IPNet
		defaultIncludeExternalIPNetwork []*net.IPNet
		defaultExcludeIngressIPNetwork  []*net.IPNet
		defaultExcludeExternalIPNetwork []*net.IPNet
		expected                        IPStatus
	}{
		{
			name:      "empty",
			targetIPs: IPStatus{},
			svc:       corev1.Service{},
			expected:  IPStatus{},
		},
		{
			name: "filter out",
			targetIPs: IPStatus{
				IngressIPs: []string{
					"10.222.0.1",
					"10.222.2.1",
					"10.222.2.2",
					"2603:c022:8005:302:312::",
					"2603:c022:8005:302:111::",
				},
				ExternalIPs: []string{
					"10.222.0.1",
					"10.222.2.1",
					"10.222.2.2",
					"2603:c022:8005:302:312::",
					"2603:c022:8005:302:111::",
				},
			},
			svc: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						LabelExcludeExternalIPNets: "2603:c022:8005:302:312::/72",
					},
				},
			},
			defaultExcludeIngressIPNetwork: []*net.IPNet{
				{IP: net.IPv4(10, 222, 2, 0), Mask: net.IPv4Mask(255, 255, 255, 0)},
			},
			defaultExcludeExternalIPNetwork: []*net.IPNet{
				{IP: net.ParseIP("2603:c022:8005:302::"), Mask: net.CIDRMask(64, 128)},
			},
			expected: IPStatus{
				IngressIPs: []string{
					"10.222.0.1",
					"2603:c022:8005:302:312::",
					"2603:c022:8005:302:111::",
				},
				ExternalIPs: []string{
					"10.222.0.1",
					"10.222.2.1",
					"10.222.2.2",
					"2603:c022:8005:302:111::",
				},
			},
		},
		{
			name: "select",
			targetIPs: IPStatus{
				IngressIPs: []string{
					"10.222.0.1",
					"10.222.2.1",
					"10.222.2.2",
					"2603:c022:8005:302:312::",
					"2603:c022:8005:302:111::",
				},
				ExternalIPs: []string{
					"10.222.0.1",
					"10.222.2.1",
					"10.222.2.2",
					"2603:c022:8005:302:312::",
					"2603:c022:8005:302:111::",
				},
			},
			svc: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						LabelIncludeExternalIPNets: "2603:c022:8005:302:312::/72",
					},
				},
			},
			defaultIncludeIngressIPNetwork: []*net.IPNet{
				{IP: net.IPv4(10, 222, 2, 0), Mask: net.IPv4Mask(255, 255, 255, 0)},
			},
			defaultIncludeExternalIPNetwork: []*net.IPNet{
				{IP: net.ParseIP("2603:c022:8005:302::"), Mask: net.CIDRMask(64, 128)},
			},
			expected: IPStatus{
				IngressIPs: []string{
					"10.222.2.1",
					"10.222.2.2",
				},
				ExternalIPs: []string{
					"2603:c022:8005:302:312::",
				},
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			u := usecase{
				defaultIncludeIngressIPNetwork:  tc.defaultIncludeIngressIPNetwork,
				defaultIncludeExternalIPNetwork: tc.defaultIncludeExternalIPNetwork,
				defaultExcludeIngressIPNetwork:  tc.defaultExcludeIngressIPNetwork,
				defaultExcludeExternalIPNetwork: tc.defaultExcludeExternalIPNetwork,
			}
			actual := u.filterTargetIPs(tc.targetIPs, tc.svc)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestUsecase_getIPNetFrom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		svc            corev1.Service
		annotationName string
		defaultVal     []*net.IPNet
		expected       []*net.IPNet
	}{
		{
			name:           "empty annotation: use defaultVal",
			svc:            corev1.Service{},
			annotationName: "some-annotation",
			defaultVal:     []*net.IPNet{{IP: net.IPv4zero, Mask: net.IPv4Mask(0, 0, 0, 0)}},
			expected:       []*net.IPNet{{IP: net.IPv4zero, Mask: net.IPv4Mask(0, 0, 0, 0)}},
		},
		{
			name: "[IPv4] get from annotation if exists",
			svc: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"some-annotation": "10.0.0.0/8,172.30.0.0/16",
					},
				},
			},
			annotationName: "some-annotation",
			defaultVal:     []*net.IPNet{{IP: net.IPv4zero, Mask: net.IPv4Mask(0, 0, 0, 0)}},
			expected: []*net.IPNet{
				{IP: net.IPv4(10, 0, 0, 0).To4(), Mask: net.IPv4Mask(255, 0, 0, 0)},
				{IP: net.IPv4(172, 30, 0, 0).To4(), Mask: net.IPv4Mask(255, 255, 0, 0)},
			},
		},
		{
			name: "[IPv6] get from annotation if exists",
			svc: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"some-annotation": "2603:c022:8005:302::/64",
					},
				},
			},
			annotationName: "some-annotation",
			defaultVal:     []*net.IPNet{{IP: net.IPv4zero, Mask: net.IPv4Mask(0, 0, 0, 0)}},
			expected:       []*net.IPNet{{IP: net.ParseIP("2603:c022:8005:302::"), Mask: net.CIDRMask(64, 128)}},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := getIPNetFrom(tc.svc, tc.annotationName, tc.defaultVal)
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}

func TestUsecase_parseIPs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ips      []string
		expected []net.IP
	}{
		{
			name:     "nil",
			ips:      nil,
			expected: nil,
		},
		{
			name:     "empty",
			ips:      []string{},
			expected: nil,
		},
		{
			name:     "ignore error",
			ips:      []string{"0.0.0", ":fac0:0"},
			expected: []net.IP{},
		},
		{
			name: "basic",
			ips:  []string{"0.0.0.0", "10.222.0.0", "fcad:31:ca::312"},
			expected: []net.IP{
				net.IPv4zero,
				net.IPv4(10, 222, 0, 0),
				net.ParseIP("fcad:31:ca::312"),
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := parseIPs(tc.ips)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestUsecase_unparseIPs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ips      []net.IP
		expected []string
	}{
		{
			name:     "nil",
			ips:      nil,
			expected: nil,
		},
		{
			name:     "empty",
			ips:      nil,
			expected: nil,
		},
		{
			name:     "basic",
			ips:      []net.IP{net.IPv4zero, net.IPv4(10, 222, 0, 0), net.ParseIP("fcad:31:ca::312")},
			expected: []string{"0.0.0.0", "10.222.0.0", "fcad:31:ca::312"},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := unparseIPs(tc.ips)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestUsecase_filterOutIPs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      []net.IP
		ipNets   []*net.IPNet
		expected []net.IP
	}{
		{
			name:     "both nil",
			src:      nil,
			ipNets:   nil,
			expected: nil,
		},
		{
			name:     "both empty",
			src:      []net.IP{},
			ipNets:   []*net.IPNet{},
			expected: []net.IP{},
		},
		{
			name:     "empty nets same return",
			src:      []net.IP{net.IPv4zero, net.IPv6zero},
			ipNets:   []*net.IPNet{},
			expected: []net.IP{net.IPv4zero, net.IPv6zero},
		},
		{
			name: "all ipv6",
			src: []net.IP{
				net.ParseIP("10.222.0.1"),
				net.ParseIP("10.222.2.1"),
				net.ParseIP("10.222.3.1"),
				net.ParseIP("fcad:31:ca::1:312"),
				net.ParseIP("fcad:31:ca::2:312"),
				net.ParseIP("fcad:31:ca::3:312"),
			},
			ipNets: []*net.IPNet{
				{IP: net.IPv6zero, Mask: net.CIDRMask(0, 128)},
			},
			expected: []net.IP{
				net.ParseIP("10.222.0.1"),
				net.ParseIP("10.222.2.1"),
				net.ParseIP("10.222.3.1"),
			},
		},
		{
			name: "all ipv4",
			src: []net.IP{
				net.ParseIP("10.222.0.1"),
				net.ParseIP("10.222.2.1"),
				net.ParseIP("10.222.3.1"),
				net.ParseIP("fcad:31:ca::1:312"),
				net.ParseIP("fcad:31:ca::2:312"),
				net.ParseIP("fcad:31:ca::3:312"),
			},
			ipNets: []*net.IPNet{
				{IP: net.IPv4zero, Mask: net.IPv4Mask(0, 0, 0, 0)},
			},
			expected: []net.IP{
				net.ParseIP("fcad:31:ca::1:312"),
				net.ParseIP("fcad:31:ca::2:312"),
				net.ParseIP("fcad:31:ca::3:312"),
			},
		},
		{
			name: "mixed version",
			src: []net.IP{
				net.ParseIP("10.222.0.1"),
				net.ParseIP("10.222.2.1"),
				net.ParseIP("fcad:31:ca::1:312"),
				net.ParseIP("fcad:31:ca::2:312"),
			},
			ipNets: []*net.IPNet{
				{IP: net.ParseIP("10.222.2.0"), Mask: net.IPv4Mask(255, 255, 255, 0)},
				{IP: net.ParseIP("fcad:31:ca::1:0"), Mask: net.CIDRMask(112, 128)},
			},
			expected: []net.IP{net.ParseIP("10.222.0.1"), net.ParseIP("fcad:31:ca::2:312")},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := filterOutIPs(tc.src, tc.ipNets)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestUsecase_selectIPs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      []net.IP
		ipNets   []*net.IPNet
		expected []net.IP
	}{
		{
			name:     "both nil",
			src:      nil,
			ipNets:   nil,
			expected: nil,
		},
		{
			name:     "both empty",
			src:      []net.IP{},
			ipNets:   []*net.IPNet{},
			expected: []net.IP{},
		},
		{
			name:     "empty nets same return",
			src:      []net.IP{net.IPv4zero, net.IPv6zero},
			ipNets:   []*net.IPNet{},
			expected: []net.IP{net.IPv4zero, net.IPv6zero},
		},
		{
			name: "all ipv6",
			src: []net.IP{
				net.ParseIP("10.222.0.1"),
				net.ParseIP("10.222.2.1"),
				net.ParseIP("10.222.3.1"),
				net.ParseIP("fcad:31:ca::1:312"),
				net.ParseIP("fcad:31:ca::2:312"),
				net.ParseIP("fcad:31:ca::3:312"),
			},
			ipNets: []*net.IPNet{
				{IP: net.IPv6zero, Mask: net.CIDRMask(0, 128)},
			},
			expected: []net.IP{
				net.ParseIP("fcad:31:ca::1:312"),
				net.ParseIP("fcad:31:ca::2:312"),
				net.ParseIP("fcad:31:ca::3:312"),
			},
		},
		{
			name: "all ipv4",
			src: []net.IP{
				net.ParseIP("10.222.0.1"),
				net.ParseIP("10.222.2.1"),
				net.ParseIP("10.222.3.1"),
				net.ParseIP("fcad:31:ca::1:312"),
				net.ParseIP("fcad:31:ca::2:312"),
				net.ParseIP("fcad:31:ca::3:312"),
			},
			ipNets: []*net.IPNet{
				{IP: net.IPv4zero, Mask: net.IPv4Mask(0, 0, 0, 0)},
			},
			expected: []net.IP{
				net.ParseIP("10.222.0.1"),
				net.ParseIP("10.222.2.1"),
				net.ParseIP("10.222.3.1"),
			},
		},
		{
			name: "mixed version",
			src: []net.IP{
				net.ParseIP("10.222.0.1"),
				net.ParseIP("10.222.2.1"),
				net.ParseIP("10.222.3.1"),
				net.ParseIP("fcad:31:ca::1:312"),
				net.ParseIP("fcad:31:ca::2:312"),
				net.ParseIP("fcad:31:ca::3:312"),
			},
			ipNets: []*net.IPNet{
				{IP: net.ParseIP("10.222.2.0"), Mask: net.IPv4Mask(255, 255, 255, 0)},
				{IP: net.ParseIP("fcad:31:ca::1:0"), Mask: net.CIDRMask(112, 128)},
			},
			expected: []net.IP{net.ParseIP("10.222.2.1"), net.ParseIP("fcad:31:ca::1:312")},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := selectIPs(tc.src, tc.ipNets)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
