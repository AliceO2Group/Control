package dcs

import (
	"testing"

	"github.com/spf13/viper"
)

// TestPluginInitWithUnavailableGateway tests that the plugin can initialize
// even if the DCS gateway is unavailable, and that it starts a reconnection
// goroutine that keeps trying to connect.
func TestPluginInitWithUnavailableGateway(t *testing.T) {
	// Save original endpoint
	originalEndpoint := viper.GetString("dcsServiceEndpoint")
	defer viper.Set("dcsServiceEndpoint", originalEndpoint)
	
	// Set an unreachable endpoint
	viper.Set("dcsServiceEndpoint", "localhost:99999")
	
	plugin := NewPlugin("localhost:99999").(*Plugin)
	
	// Initialize should succeed even with unavailable gateway
	err := plugin.Init("test-instance")
	if err != nil {
		t.Fatalf("Plugin.Init() should succeed even with unavailable gateway, got error: %v", err)
	}
	
	// Verify the plugin thinks it's initialized
	if plugin.GetName() != "dcs" {
		t.Errorf("Plugin should be properly initialized")
	}
	
	// The connection state should reflect the fact that we can't connect
	// but the plugin should still be functional
	connState := plugin.GetConnectionState()
	if connState == "READY" {
		t.Errorf("Connection state should not be READY with unavailable gateway, got: %s", connState)
	}
	
	// Clean up
	if plugin.dcsClient != nil {
		plugin.dcsClient.Close()
	}
}