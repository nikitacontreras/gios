package deploy

import (
	"fmt"
	"sync"
	"time"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/forward"
	"github.com/nikitastrike/gios/pkg/utils"
)

var (
	activeListeners = make(map[uint16]*forward.ConnListener)
	listenerMutex   sync.Mutex
)

// GetFirstDevice returns the first detected iOS device
func GetFirstDevice() (ios.DeviceEntry, error) {
	deviceList, err := ios.ListDevices()
	if err != nil {
		return ios.DeviceEntry{}, err
	}
	if len(deviceList.DeviceList) == 0 {
		return ios.DeviceEntry{}, fmt.Errorf("no iOS devices detected via USB")
	}
	return deviceList.DeviceList[0], nil
}

// StartNativePortForward starts a background port forwarder (iproxy replacement)
func StartNativePortForward(hostPort, devicePort uint16) error {
	listenerMutex.Lock()
	defer listenerMutex.Unlock()

	if _, exists := activeListeners[hostPort]; exists {
		return nil // Already running
	}

	device, err := GetFirstDevice()
	if err != nil {
		return err
	}

	listener, err := forward.Forward(device, hostPort, devicePort)
	if err != nil {
		return fmt.Errorf("failed to start forwarder: %w", err)
	}

	activeListeners[hostPort] = listener
	return nil
}

// StopAllForwarders closes all active port forwarders
func StopAllForwarders() {
	listenerMutex.Lock()
	defer listenerMutex.Unlock()

	for port, listener := range activeListeners {
		listener.Close()
		delete(activeListeners, port)
	}
}

// EnsureUSBTunnelNative replaces the old check and actually starts the tunnel if needed
func EnsureUSBTunnelNative() bool {
	fmt.Printf("%s[gios]%s Creating native USB tunnel (usbmuxd)...\n", utils.ColorCyan, utils.ColorReset)
	
	// Start forwarding 2222 -> 22 for SSH
	err := StartNativePortForward(2222, 22)
	if err != nil {
		fmt.Printf("%s[!] USB Error:%s %v\n", utils.ColorRed, utils.ColorReset, err)
		return false
	}

	// Give a small moment for the listener to bind if it's the first time
	time.Sleep(200 * time.Millisecond)
	fmt.Printf("%s[+] USB Tunnel active:%s localhost:2222 -> Device:22\n", utils.ColorGreen, utils.ColorReset)
	return true
}
