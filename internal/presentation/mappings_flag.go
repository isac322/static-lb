package presentation

import (
	"errors"
	"fmt"

	"github.com/isac322/static-lb/internal/application"
)

type IPMappingTargets struct {
	value []application.IPMappingTarget
}

func (f *IPMappingTargets) String() string {
	return ""
}

func (f *IPMappingTargets) Mappings() []application.IPMappingTarget {
	return f.value
}

func (f *IPMappingTargets) Set(s string) error {
	switch t := application.IPMappingTarget(s); t {
	case application.IPMappingTargetExternal, application.IPMappingTargetIngress:
		f.value = append(f.value, t)
		return nil
	default:
		return errors.New(fmt.Sprintf("invalid mapping target: %s", s))
	}
}
