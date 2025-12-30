package main

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"
)

var landmarksLabel *widget.Label


// findUSBModemPort scans for /dev/tty.usbmodem* devices
func findUSBModemPort() string {
	matches, err := filepath.Glob("/dev/tty.usbmodem*")
	if err != nil || len(matches) == 0 {
		return "/dev/tty.usbmodem2101" // fallback default
	}
	return matches[0] // return first match
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("PicoPeeker")

	// Output entry - monospace, read-only, but selectable for copy-paste
	output := widget.NewMultiLineEntry()
	output.Wrapping = fyne.TextWrapOff
	output.TextStyle = fyne.TextStyle{Monospace: true}
	// Don't disable - just make it read-only by handling key events
	// This keeps text dark and readable while still being selectable

	// Port input (shared across tabs)
	portEntry := widget.NewEntry()
	defaultPort := findUSBModemPort()
	portEntry.SetPlaceHolder(defaultPort)
	portEntry.SetText(defaultPort)

	// Landmarks label (shared across tabs)
	landmarksLabel = widget.NewLabel("Landmarks: Not connected")
	landmarksLabel.TextStyle = fyne.TextStyle{Monospace: true}

	// Connect button (shared)
	connectBtn := widget.NewButton("Get Landmarks", func() {
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

	// Display format selector
	displayFormatSelect := widget.NewSelect([]string{"Bytes (Hex)", "16-bit Words", "32-bit Words", "Float (32-bit)"}, nil)
	displayFormatSelect.SetSelected("Bytes (Hex)")

	// Channel for background operations
	type uiUpdate struct {
		text string
	}
	updateChan := make(chan uiUpdate, 10)

	// Goroutine to handle UI updates on main thread
	go func() {
		for update := range updateChan {
			// Must use fyne.Do to safely update UI from goroutine
			text := update.text
			fyne.Do(func() {
				output.SetText(text)
			})
		}
	}()

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

		// Validate address format (must be hex)
		address = strings.TrimSpace(address)
		if !strings.HasPrefix(address, "0x") && !strings.HasPrefix(address, "0X") {
			output.SetText("Error: Address must start with 0x (e.g., 0x20000000)")
			return
		}
		if _, err := strconv.ParseUint(address[2:], 16, 32); err != nil {
			output.SetText("Error: Invalid hex address. Use format like 0x20000000")
			return
		}

		// Validate length (must be a positive integer)
		lengthVal, err := strconv.Atoi(length)
		if err != nil || lengthVal <= 0 || lengthVal > 4096 {
			output.SetText("Error: Length must be a number between 1 and 4096")
			return
		}

		output.SetText("Reading memory...")

		// Run read in background goroutine to keep UI responsive
		go func() {
			result, err := readMemory(portName, address, length)
			if err != nil {
				updateChan <- uiUpdate{text: fmt.Sprintf("Error: %v", err)}
			} else {
				// Format output based on selected display mode
				displayMode := displayFormatSelect.Selected
				formatted := formatMemoryDump(result, displayMode)
				updateChan <- uiUpdate{text: formatted}
			}
		}()
	})
	readMemoryBtn.Importance = widget.HighImportance

	romBtn := widget.NewButton("ROM", func() {
		addressEntry.SetText("0x00000000")
	})
	flashBtn := widget.NewButton("Flash", func() {
		addressEntry.SetText("0x10000000")
	})
	sramBtn := widget.NewButton("SRAM", func() {
		addressEntry.SetText("0x20000000")
	})
	gpioBtn := widget.NewButton("GPIO", func() {
		addressEntry.SetText("0x40028000")
	})

	// dumpRomBtn := widget.NewButton("Dump ROM to File (16KB)", func() {
	// 	portName := portEntry.Text
	// 	if portName == "" {
	// 		output.SetText("Error: Please enter a port name")
	// 		return
	// 	}

	// 	output.SetText("Dumping 16KB ROM... This will take ~5 seconds...")

	// 	// Show file save dialog
	// 	saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
	// 		if err != nil {
	// 			output.SetText(fmt.Sprintf("Error: %v", err))
	// 			return
	// 		}
	// 		if writer == nil {
	// 			output.SetText("Save cancelled")
	// 			return
	// 		}
	// 		defer writer.Close()

	// 		// Dump ROM in 4KB chunks (max size per READ is 4096)
	// 		romData := make([]byte, 0, 16384)
	// 		for offset := uint32(0); offset < 0x4000; offset += 4096 {
	// 			address := fmt.Sprintf("0x%08x", offset)
	// 			result, err := readMemory(portName, address, "4096")
	// 			if err != nil {
	// 				output.SetText(fmt.Sprintf("Error reading ROM at offset 0x%x: %v", offset, err))
	// 				return
	// 			}

	// 			// Parse hex dump and extract bytes
	// 			bytes := parseHexDump(result)
	// 			if len(bytes) == 0 {
	// 				output.SetText(fmt.Sprintf("Error: Failed to parse hex dump at offset 0x%x", offset))
	// 				return
	// 			}
	// 			romData = append(romData, bytes...)

	// 			output.SetText(fmt.Sprintf("Dumping ROM... %d / 16384 bytes (%.1f%%)",
	// 				len(romData), float64(len(romData))*100.0/16384.0))
	// 		}

	// 		// Write to file
	// 		_, err = writer.Write(romData)
	// 		if err != nil {
	// 			output.SetText(fmt.Sprintf("Error writing file: %v", err))
	// 			return
	// 		}

	// 		output.SetText(fmt.Sprintf("SUCCESS! Dumped %d bytes of ROM to %s\nHappy reverse engineering! 😎",
	// 			len(romData), writer.URI().Name()))
	// 	}, myWindow)

	// 	saveDialog.SetFileName("rp2350_rom.bin")
	// 	saveDialog.Show()
	// })
	// dumpRomBtn.Importance = widget.WarningImportance

	addressRow := container.NewBorder(nil, nil, widget.NewLabel("Address:"), container.NewHBox(decrementBtn, incrementBtn), addressEntry)
	lengthRow := container.NewBorder(nil, nil, widget.NewLabel("Length:"), nil, lengthEntry)
	displayFormatRow := container.NewBorder(nil, nil, widget.NewLabel("Display as:"), nil, displayFormatSelect)

	// Wrap form fields in a card with extra padding
	memorySettingsCard := widget.NewCard("", "", container.NewPadded(container.NewVBox(
		addressRow,
		lengthRow,
		displayFormatRow,
	)))

	readTab := container.NewVBox(
		memorySettingsCard,
		widget.NewCard("", "", container.NewPadded(
			container.NewHBox(widget.NewLabel("Quick Access:"), romBtn, flashBtn, sramBtn, gpioBtn),
		)),
		// dumpRomBtn,
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

	searchRegionSelect := widget.NewSelect([]string{"SRAM", "Flash"}, nil)
	searchRegionSelect.SetSelected("SRAM")

	// Helper function to validate and convert search pattern
	validateAndConvertPattern := func() (string, error) {
		pattern := searchEntry.Text
		searchType := searchTypeSelect.Selected

		if pattern == "" {
			return "", fmt.Errorf("Please enter a search pattern")
		}

		// Convert pattern to hex based on type
		hexPattern := ""
		switch searchType {
		case "Hex Bytes":
			// Validate hex bytes - remove spaces and check if valid hex
			hexPattern = strings.ReplaceAll(strings.ToUpper(pattern), " ", "")
			if len(hexPattern)%2 != 0 {
				return "", fmt.Errorf("Hex pattern must have even number of characters (e.g., DEADBEEF)")
			}
			if len(hexPattern) == 0 {
				return "", fmt.Errorf("Hex pattern cannot be empty")
			}
			// Check if all characters are valid hex
			for _, c := range hexPattern {
				if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')) {
					return "", fmt.Errorf("Invalid hex pattern. Use only 0-9 and A-F (e.g., DEADBEEF)")
				}
			}
		case "ASCII String":
			if pattern == "" {
				return "", fmt.Errorf("ASCII string cannot be empty")
			}
			hexPattern = stringToHex(pattern)
		case "32-bit Int (LE)":
			val, err := strconv.ParseInt(pattern, 10, 32)
			if err != nil {
				return "", fmt.Errorf("Invalid integer: %v", err)
			}
			hexPattern = int32ToHexLE(int32(val))
		}
		return hexPattern, nil
	}

	searchBtn := widget.NewButton("Search Memory", func() {
		portName := portEntry.Text
		if portName == "" {
			output.SetText("Error: Please enter a port name")
			return
		}

		hexPattern, err := validateAndConvertPattern()
		if err != nil {
			output.SetText(fmt.Sprintf("Error: %v", err))
			return
		}

		region := searchRegionSelect.Selected

		// Run search in background goroutine to keep UI responsive
		go func() {
			var result string
			var err error

			if region == "SRAM" {
				updateChan <- uiUpdate{text: "Searching SRAM (this may take 5-10 seconds)..."}
				result, err = searchMemory(portName, hexPattern)
			} else { // Flash
				updateChan <- uiUpdate{text: "Searching Flash (this may take 30-60 seconds for 4MB)..."}
				result, err = searchFlash(portName, hexPattern)
			}

			if err != nil {
				updateChan <- uiUpdate{text: fmt.Sprintf("Error: %v", err)}
			} else {
				updateChan <- uiUpdate{text: result}
			}
		}()
	})
	searchBtn.Importance = widget.HighImportance

	searchTypeRow := container.NewBorder(nil, nil, widget.NewLabel("Search Type:"), nil, searchTypeSelect)
	searchPatternRow := container.NewBorder(nil, nil, widget.NewLabel("Pattern:"), nil, searchEntry)
	searchRegionRow := container.NewBorder(nil, nil, widget.NewLabel("Region:"), nil, searchRegionSelect)

	// Wrap search settings in a card with extra padding
	searchSettingsCard := widget.NewCard("", "", container.NewPadded(container.NewVBox(
		searchTypeRow,
		searchPatternRow,
		searchRegionRow,
	)))

	searchTab := container.NewVBox(
		searchSettingsCard,
		widget.NewCard("", "", container.NewPadded(
			widget.NewLabel("SRAM: Runtime data (520KB) | Flash: String literals (4MB)"),
		)),
		searchBtn,
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
	myWindow.Resize(fyne.NewSize(900, 850))
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

// Parse hex dump output and extract raw bytes
func parseHexDump(dump string) []byte {
	var result []byte
	lines := strings.Split(dump, "\n")

	for _, line := range lines {
		// Look for lines with hex dump format: "00000000: 01 02 03 04 ..."
		if !strings.Contains(line, ":") {
			continue
		}

		// Split on colon
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		// Extract hex bytes (before the ASCII section)
		hexPart := parts[1]
		// Remove ASCII part (after double space)
		if idx := strings.Index(hexPart, "  "); idx != -1 {
			hexPart = hexPart[:idx]
		}

		// Parse hex bytes
		hexBytes := strings.Fields(hexPart)
		for _, hexByte := range hexBytes {
			if len(hexByte) == 2 {
				val, err := strconv.ParseUint(hexByte, 16, 8)
				if err == nil {
					result = append(result, byte(val))
				}
			}
		}
	}

	return result
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

func searchMemory(portName, pattern string) (string, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
	}

	port, err := serial.Open(portName, mode)
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

func searchFlash(portName, pattern string) (string, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
	}

	port, err := serial.Open(portName, mode)
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

// formatMemoryDump reformats the hex dump based on the selected display mode
func formatMemoryDump(rawDump string, displayMode string) string {
	// If not in bytes mode, we need to parse and reformat
	if displayMode == "Bytes (Hex)" {
		return rawDump // Return original format
	}

	// Parse the hex dump to extract bytes
	bytes := parseHexDump(rawDump)
	if len(bytes) == 0 {
		return rawDump // Return original if parsing failed
	}

	// Extract starting address from the dump
	startAddr := extractStartAddress(rawDump)

	// Format based on display mode
	switch displayMode {
	case "16-bit Words":
		return formatAs16BitWords(bytes, startAddr)
	case "32-bit Words":
		return formatAs32BitWords(bytes, startAddr)
	case "Float (32-bit)":
		return formatAsFloats(bytes, startAddr)
	default:
		return rawDump
	}
}

// extractStartAddress extracts the starting address from the hex dump
func extractStartAddress(dump string) uint32 {
	lines := strings.Split(dump, "\n")
	for _, line := range lines {
		// Look for "Address: 0x12345678"
		if strings.Contains(line, "Address:") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "Address:" && i+1 < len(fields) {
					addr := strings.TrimSuffix(fields[i+1], ",")
					var val uint64
					fmt.Sscanf(addr, "0x%x", &val)
					return uint32(val)
				}
			}
		}
	}
	return 0
}

// formatAs16BitWords formats bytes as 16-bit words (little-endian)
func formatAs16BitWords(bytes []byte, startAddr uint32) string {
	var sb strings.Builder
	sb.WriteString("=== 16-bit Word View (Little-Endian) ===\n\n")
	sb.WriteString("Address:  Hex Bytes  Value      Decimal\n")
	sb.WriteString("--------  ---------  ------     --------\n")

	for i := 0; i+1 < len(bytes); i += 2 {
		addr := startAddr + uint32(i)
		val := binary.LittleEndian.Uint16(bytes[i : i+2])
		sb.WriteString(fmt.Sprintf("%08x: %02x %02x     0x%04x     %d\n",
			addr, bytes[i], bytes[i+1], val, int16(val)))
	}

	// Handle odd byte at end if present
	if len(bytes)%2 == 1 {
		i := len(bytes) - 1
		addr := startAddr + uint32(i)
		sb.WriteString(fmt.Sprintf("%08x: %02x        0x%02x       %d (partial)\n",
			addr, bytes[i], bytes[i], int8(bytes[i])))
	}

	sb.WriteString("\n===END===\n")
	return sb.String()
}

// formatAs32BitWords formats bytes as 32-bit words (little-endian)
func formatAs32BitWords(bytes []byte, startAddr uint32) string {
	var sb strings.Builder
	sb.WriteString("=== 32-bit Word View (Little-Endian) ===\n\n")
	sb.WriteString("Address:  Hex Bytes        Value       Decimal (signed)  Decimal (unsigned)\n")
	sb.WriteString("--------  ---------------  ----------  ----------------  ------------------\n")

	for i := 0; i+3 < len(bytes); i += 4 {
		addr := startAddr + uint32(i)
		val := binary.LittleEndian.Uint32(bytes[i : i+4])
		sb.WriteString(fmt.Sprintf("%08x: %02x %02x %02x %02x  0x%08x  %-16d  %d\n",
			addr, bytes[i], bytes[i+1], bytes[i+2], bytes[i+3],
			val, int32(val), val))
	}

	// Handle remaining bytes if not aligned to 4
	remainder := len(bytes) % 4
	if remainder > 0 {
		i := len(bytes) - remainder
		addr := startAddr + uint32(i)
		sb.WriteString(fmt.Sprintf("%08x: ", addr))
		for j := 0; j < remainder; j++ {
			sb.WriteString(fmt.Sprintf("%02x ", bytes[i+j]))
		}
		sb.WriteString("(partial word)\n")
	}

	sb.WriteString("\n===END===\n")
	return sb.String()
}

// formatAsFloats formats bytes as 32-bit floats (little-endian)
func formatAsFloats(bytes []byte, startAddr uint32) string {
	var sb strings.Builder
	sb.WriteString("=== Float View (32-bit, Little-Endian) ===\n\n")
	sb.WriteString("Address:  Hex Bytes        Float Value\n")
	sb.WriteString("--------  ---------------  -----------\n")

	for i := 0; i+3 < len(bytes); i += 4 {
		addr := startAddr + uint32(i)
		bits := binary.LittleEndian.Uint32(bytes[i : i+4])
		floatVal := *(*float32)(unsafe.Pointer(&bits))
		sb.WriteString(fmt.Sprintf("%08x: %02x %02x %02x %02x  %f\n",
			addr, bytes[i], bytes[i+1], bytes[i+2], bytes[i+3], floatVal))
	}

	// Handle remaining bytes if not aligned to 4
	remainder := len(bytes) % 4
	if remainder > 0 {
		i := len(bytes) - remainder
		addr := startAddr + uint32(i)
		sb.WriteString(fmt.Sprintf("%08x: ", addr))
		for j := 0; j < remainder; j++ {
			sb.WriteString(fmt.Sprintf("%02x ", bytes[i+j]))
		}
		sb.WriteString("(partial float)\n")
	}

	sb.WriteString("\n===END===\n")
	return sb.String()
}
