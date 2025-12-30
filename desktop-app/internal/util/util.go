package util

import "path/filepath"

// findUSBModemPort scans for /dev/tty.usbmodem* devices
func FindUSBModemPort() string {
	matches, err := filepath.Glob("/dev/tty.usbmodem*")
	if err != nil || len(matches) == 0 {
		return "/dev/tty.usbmodem2101" // fallback default
	}
	return matches[0] // return first match
}
