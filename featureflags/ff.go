package featureflags

import (
	"os"
	"strings"
)

const (
	UseTelemetry = "USE_TELEMETRY"
)

type FeatureFlag interface {
	Enabled() bool
}

func New(flag string, config interface{}) FeatureFlag {
	switch flag {
	case UseTelemetry:
		return &useTelemetry{}
	}

	return nil
}

type useTelemetry struct{}

func (ut *useTelemetry) Enabled() bool {
	val, ok := os.LookupEnv(UseTelemetry)
	if !ok {
		return false
	}

	return strings.EqualFold(val, "true")
}
