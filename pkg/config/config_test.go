package config

import (
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	os.Setenv("IM_DISCOVERY_ADDR", ":8085")
	discoveryConfig := NewConf().GetDiscoveryConfig()
	t.Logf("discoveryConfig: %v", discoveryConfig)
}
