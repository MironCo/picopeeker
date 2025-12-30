package serial

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	goserial "go.bug.st/serial"
)

func FetchLandmarks(portName string) (string, error) {
	mode := &goserial.Mode{
		BaudRate: 115200,
	}

	port, err := goserial.Open(portName, mode)
	if err != nil {
		return "", fmt.Errorf("failed to open port: %w", err)
	}
	defer port.Close()

	// Clear any initial data
	time.Sleep(200 * time.Millisecond)
	buf := make([]byte, 4096)
	port.SetReadTimeout(100 * time.Millisecond)
	port.Read(buf) // Discard

	// Send LANDMARKS command
	_, err = port.Write([]byte("LANDMARKS\n"))
	if err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	result := ""
	startTime := time.Now()
	for time.Since(startTime) < 2*time.Second {
		port.SetReadTimeout(100 * time.Millisecond)
		n, _ := port.Read(buf)
		if n > 0 {
			result += string(buf[:n])
			// Check if we got the end marker
			if strings.Contains(result, "END_LANDMARKS") {
				break
			}
		}
	}

	// Parse landmarks
	landmarks := ParseLandmarks(result)
	if landmarks == "" {
		return "No landmarks found in response", nil
	}

	return landmarks, nil
}

func ParseLandmarks(data string) string {
	// Look for LANDMARKS section
	if !strings.Contains(data, "LANDMARKS:") {
		return ""
	}

	result := []string{}

	// Extract each landmark
	re := regexp.MustCompile(`(\w+)=(0x[0-9a-fA-F]+)`)
	matches := re.FindAllStringSubmatch(data, -1)

	for _, match := range matches {
		if len(match) == 3 {
			result = append(result, fmt.Sprintf("%s @ %s", match[1], match[2]))
		}
	}

	if len(result) == 0 {
		return "Found LANDMARKS but couldn't parse"
	}

	return strings.Join(result, " | ")
}

func ReadMemory(portName, address, length string) (string, error) {
	mode := &goserial.Mode{
		BaudRate: 115200,
	}

	port, err := goserial.Open(portName, mode)
	if err != nil {
		return "", fmt.Errorf("failed to open port: %w", err)
	}
	defer port.Close()

	// Clear any buffered data before sending command
	time.Sleep(100 * time.Millisecond)
	clearBuf := make([]byte, 4096)
	port.SetReadTimeout(50 * time.Millisecond)
	port.Read(clearBuf) // Discard any leftover data

	// Send command
	command := fmt.Sprintf("READ:%s:%s\n", address, length)
	_, err = port.Write([]byte(command))
	if err != nil {
		return "", fmt.Errorf("failed to write: %w", err)
	}

	// Read until we see ===END===
	result := ""
	buf := make([]byte, 1024)
	startTime := time.Now()

	for {
		// Timeout after 5 seconds
		if time.Since(startTime) > 5*time.Second {
			if result == "" {
				return "", fmt.Errorf("timeout: no response from Pico")
			}
			return result, nil
		}

		port.SetReadTimeout(100 * time.Millisecond)
		n, _ := port.Read(buf)

		if n > 0 {
			result += string(buf[:n])

			// Check if we got the end marker
			if strings.Contains(result, "===END===") {
				return result, nil
			}
		}
	}
}

func SearchMemory(portName, pattern string) (string, error) {
	mode := &goserial.Mode{
		BaudRate: 115200,
	}

	port, err := goserial.Open(portName, mode)
	if err != nil {
		return "", fmt.Errorf("failed to open port: %w", err)
	}
	defer port.Close()

	// Clear any buffered data before sending command
	time.Sleep(100 * time.Millisecond)
	clearBuf := make([]byte, 4096)
	port.SetReadTimeout(50 * time.Millisecond)
	port.Read(clearBuf) // Discard any leftover data

	// Send command
	command := fmt.Sprintf("SEARCH:%s\n", pattern)
	_, err = port.Write([]byte(command))
	if err != nil {
		return "", fmt.Errorf("failed to write: %w", err)
	}

	// Read until we see ===END===
	// Search takes longer, so we use a longer timeout
	result := ""
	buf := make([]byte, 1024)
	startTime := time.Now()

	for {
		// Timeout after 30 seconds (search can be slow)
		if time.Since(startTime) > 30*time.Second {
			if result == "" {
				return "", fmt.Errorf("timeout: no response from Pico")
			}
			return result, nil
		}

		port.SetReadTimeout(100 * time.Millisecond)
		n, _ := port.Read(buf)

		if n > 0 {
			result += string(buf[:n])

			// Check if we got the end marker
			if strings.Contains(result, "===END===") {
				return result, nil
			}
		}
	}
}

func SearchFlash(portName, pattern string) (string, error) {
	mode := &goserial.Mode{
		BaudRate: 115200,
	}

	port, err := goserial.Open(portName, mode)
	if err != nil {
		return "", fmt.Errorf("failed to open port: %w", err)
	}
	defer port.Close()

	// Clear any buffered data before sending command
	time.Sleep(100 * time.Millisecond)
	clearBuf := make([]byte, 4096)
	port.SetReadTimeout(50 * time.Millisecond)
	port.Read(clearBuf) // Discard any leftover data

	// Send SEARCHFLASH command
	command := fmt.Sprintf("SEARCHFLASH:%s\n", pattern)
	_, err = port.Write([]byte(command))
	if err != nil {
		return "", fmt.Errorf("failed to write: %w", err)
	}

	// Read until we see ===END===
	// Flash search can take a LONG time (4MB), so use a very long timeout
	result := ""
	buf := make([]byte, 1024)
	startTime := time.Now()

	for {
		// Timeout after 120 seconds (Flash is much larger than SRAM)
		if time.Since(startTime) > 120*time.Second {
			if result == "" {
				return "", fmt.Errorf("timeout: no response from Pico")
			}
			return result, nil
		}

		port.SetReadTimeout(100 * time.Millisecond)
		n, _ := port.Read(buf)

		if n > 0 {
			result += string(buf[:n])

			// Check if we got the end marker
			if strings.Contains(result, "===END===") {
				return result, nil
			}
		}
	}
}
