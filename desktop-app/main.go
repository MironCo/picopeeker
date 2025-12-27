package main

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"
)

var landmarksLabel *widget.Label

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("PicoPeeker")

	// Output label - monospace for hex dump
	output := widget.NewLabel("")
	output.Wrapping = fyne.TextWrapOff
	output.TextStyle = fyne.TextStyle{Monospace: true}

	// Port input (shared across tabs)
	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("/dev/tty.usbmodem101")
	portEntry.SetText("/dev/tty.usbmodem101")

	// Landmarks label (shared across tabs)
	landmarksLabel = widget.NewLabel("Landmarks: Not connected")
	landmarksLabel.TextStyle = fyne.TextStyle{Monospace: true}

	// Connect button (shared)
	connectBtn := widget.NewButton("Connect & Get Landmarks", func() {
		portName := portEntry.Text
		if portName == "" {
			output.SetText("Error: Please enter a port name")
			return
		}

		landmarks, err := fetchLandmarks(portName)
		if err != nil {
			landmarksLabel.SetText(fmt.Sprintf("Landmarks: Could not connect - %v", err))
		} else {
			landmarksLabel.SetText(fmt.Sprintf("Landmarks: %s", landmarks))
		}
	})

	// === READ MEMORY TAB ===
	addressEntry := widget.NewEntry()
	addressEntry.SetPlaceHolder("0x20000000")
	addressEntry.SetText("0x20000000")

	decrementBtn := widget.NewButton("-", func() {
		address := addressEntry.Text
		if address == "" {
			return
		}
		val := uint64(0)
		fmt.Sscanf(address, "0x%x", &val)
		if val >= 256 {
			val -= 256
		}
		addressEntry.SetText(fmt.Sprintf("0x%08x", val))
	})

	incrementBtn := widget.NewButton("+", func() {
		address := addressEntry.Text
		if address == "" {
			return
		}
		val := uint64(0)
		fmt.Sscanf(address, "0x%x", &val)
		val += 256
		addressEntry.SetText(fmt.Sprintf("0x%08x", val))
	})

	lengthEntry := widget.NewEntry()
	lengthEntry.SetPlaceHolder("256")
	lengthEntry.SetText("256")

	readMemoryBtn := widget.NewButton("Read Memory", func() {
		portName := portEntry.Text
		address := addressEntry.Text
		length := lengthEntry.Text

		if portName == "" {
			output.SetText("Error: Please enter a port name")
			return
		}
		if address == "" {
			output.SetText("Error: Please enter an address")
			return
		}
		if length == "" {
			output.SetText("Error: Please enter a length")
			return
		}

		output.SetText("Reading memory...")

		result, err := readMemory(portName, address, length)
		if err != nil {
			output.SetText(fmt.Sprintf("Error: %v", err))
			return
		}

		output.SetText(result)
	})
	readMemoryBtn.Importance = widget.HighImportance

	sramBtn := widget.NewButton("SRAM", func() {
		addressEntry.SetText("0x20000000")
	})
	flashBtn := widget.NewButton("Flash", func() {
		addressEntry.SetText("0x10000000")
	})
	gpioBtn := widget.NewButton("GPIO", func() {
		addressEntry.SetText("0x40028000")
	})

	addressRow := container.NewBorder(nil, nil, widget.NewLabel("Address:"), container.NewHBox(decrementBtn, incrementBtn), addressEntry)
	lengthRow := container.NewBorder(nil, nil, widget.NewLabel("Length:"), nil, lengthEntry)

	readTab := container.NewVBox(
		addressRow,
		lengthRow,
		container.NewHBox(widget.NewLabel("Quick:"), sramBtn, flashBtn, gpioBtn),
		readMemoryBtn,
	)

	// === SEARCH MEMORY TAB ===
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Enter pattern...")
	searchEntry.SetText("")

	searchTypeSelect := widget.NewSelect([]string{"Hex Bytes", "ASCII String", "32-bit Int (LE)"}, func(value string) {
		switch value {
		case "Hex Bytes":
			searchEntry.SetPlaceHolder("DEADBEEF")
		case "ASCII String":
			searchEntry.SetPlaceHolder("hello world")
		case "32-bit Int (LE)":
			searchEntry.SetPlaceHolder("42")
		}
	})
	searchTypeSelect.SetSelected("Hex Bytes")

	searchMemoryBtn := widget.NewButton("Search SRAM", func() {
		portName := portEntry.Text
		pattern := searchEntry.Text
		searchType := searchTypeSelect.Selected

		if portName == "" {
			output.SetText("Error: Please enter a port name")
			return
		}
		if pattern == "" {
			output.SetText("Error: Please enter a search pattern")
			return
		}

		// Convert pattern to hex based on type
		hexPattern := ""
		switch searchType {
		case "Hex Bytes":
			hexPattern = pattern
		case "ASCII String":
			hexPattern = stringToHex(pattern)
		case "32-bit Int (LE)":
			val, err := strconv.ParseInt(pattern, 10, 32)
			if err != nil {
				output.SetText(fmt.Sprintf("Error: Invalid integer: %v", err))
				return
			}
			hexPattern = int32ToHexLE(int32(val))
		}

		output.SetText("Searching SRAM (this may take 5-10 seconds)...")

		result, err := searchMemory(portName, hexPattern)
		if err != nil {
			output.SetText(fmt.Sprintf("Error: %v", err))
			return
		}

		output.SetText(result)
	})
	searchMemoryBtn.Importance = widget.HighImportance

	searchTypeRow := container.NewBorder(nil, nil, widget.NewLabel("Search Type:"), nil, searchTypeSelect)
	searchPatternRow := container.NewBorder(nil, nil, widget.NewLabel("Pattern:"), nil, searchEntry)

	searchTab := container.NewVBox(
		searchTypeRow,
		searchPatternRow,
		widget.NewLabel("Searches all 520KB of SRAM for the pattern"),
		searchMemoryBtn,
	)

	// Create tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Read Memory", readTab),
		container.NewTabItem("Search Memory", searchTab),
	)

	// Main layout
	portRow := container.NewBorder(nil, nil, widget.NewLabel("Serial Port:"), connectBtn, portEntry)

	content := container.NewBorder(
		container.NewVBox(portRow, tabs),
		landmarksLabel,
		nil,
		nil,
		container.NewScroll(output),
	)

	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(900, 700))
	myWindow.ShowAndRun()
}

// Helper function to convert string to hex
func stringToHex(s string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		result += fmt.Sprintf("%02X", s[i])
	}
	return result
}

// Helper function to convert int32 to hex (little-endian)
func int32ToHexLE(val int32) string {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(val))
	return fmt.Sprintf("%02X%02X%02X%02X", buf[0], buf[1], buf[2], buf[3])
}

func fetchLandmarks(portName string) (string, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
	}

	port, err := serial.Open(portName, mode)
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
	landmarks := parseLandmarks(result)
	if landmarks == "" {
		return "No landmarks found in response", nil
	}

	return landmarks, nil
}

func parseLandmarks(data string) string {
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

func readMemory(portName, address, length string) (string, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return "", fmt.Errorf("failed to open port: %w", err)
	}
	defer port.Close()

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

func searchMemory(portName, pattern string) (string, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return "", fmt.Errorf("failed to open port: %w", err)
	}
	defer port.Close()

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
