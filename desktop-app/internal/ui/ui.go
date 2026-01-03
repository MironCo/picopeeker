package ui

import (
	"desktop-app/internal/config"
	"desktop-app/internal/format"
	"desktop-app/internal/serial"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// BuildReadMemoryTab creates the Read Memory tab UI
func BuildReadMemoryTab(portEntry *widget.Entry, output *widget.Entry, updateChan chan UIUpdate, getModel func() config.PicoModel) *container.TabItem {
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
			result, err := serial.ReadMemory(portName, address, length)
			if err != nil {
				updateChan <- UIUpdate{Text: fmt.Sprintf("Error: %v", err)}
			} else {
				// Format output based on selected display mode
				displayMode := displayFormatSelect.Selected
				formatted := format.FormatMemoryDump(result, displayMode)
				updateChan <- UIUpdate{Text: formatted}
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
		readMemoryBtn,
	)

	return container.NewTabItem("Read Memory", readTab)
}

// BuildSearchMemoryTab creates the Search Memory tab UI
func BuildSearchMemoryTab(portEntry *widget.Entry, output *widget.Entry, updateChan chan UIUpdate, getModel func() config.PicoModel) *container.TabItem {
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
			hexPattern = format.StringToHex(pattern)
		case "32-bit Int (LE)":
			val, err := strconv.ParseInt(pattern, 10, 32)
			if err != nil {
				return "", fmt.Errorf("Invalid integer: %v", err)
			}
			hexPattern = format.Int32ToHexLE(int32(val))
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
				updateChan <- UIUpdate{Text: "Searching SRAM (this may take 5-10 seconds)..."}
				result, err = serial.SearchMemory(portName, hexPattern)
			} else { // Flash
				updateChan <- UIUpdate{Text: "Searching Flash (this may take 30-60 seconds for 4MB)..."}
				result, err = serial.SearchFlash(portName, hexPattern)
			}

			if err != nil {
				updateChan <- UIUpdate{Text: fmt.Sprintf("Error: %v", err)}
			} else {
				updateChan <- UIUpdate{Text: result}
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

	// Info label that updates based on selected model
	infoLabel := widget.NewLabel("")
	updateInfoLabel := func() {
		regions := config.GetMemoryRegions(getModel())
		infoLabel.SetText(fmt.Sprintf("SRAM: Runtime data (%s) | Flash: String literals (%s)", regions.SRAMSize, regions.FlashSize))
	}
	updateInfoLabel() // Set initial value

	searchTab := container.NewVBox(
		searchSettingsCard,
		widget.NewCard("", "", container.NewPadded(infoLabel)),
		searchBtn,
	)

	return container.NewTabItem("Search Memory", searchTab)
}

// UIUpdate is a message type for updating the UI from background goroutines
type UIUpdate struct {
	Text string
}
