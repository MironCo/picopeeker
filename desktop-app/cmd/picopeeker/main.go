package main

import (
	"desktop-app/internal/config"
	"desktop-app/internal/serial"
	"desktop-app/internal/ui"
	"desktop-app/internal/util"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var landmarksLabel *widget.Label
var currentModel config.PicoModel = config.Pico2

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("PicoPeeker")

	// Output entry - monospace, read-only, but selectable for copy-paste
	output := widget.NewMultiLineEntry()
	output.Wrapping = fyne.TextWrapOff
	output.TextStyle = fyne.TextStyle{Monospace: true}

	// Port input (shared across tabs)
	portEntry := widget.NewEntry()
	defaultPort := util.FindUSBModemPort()
	portEntry.SetPlaceHolder(defaultPort)
	portEntry.SetText(defaultPort)

	// Pico model selector
	modelSelect := widget.NewSelect([]string{"Pico 1 (RP2040)", "Pico 2 (RP2350)"}, func(selected string) {
		currentModel = config.GetModelFromString(selected)
		regions := config.GetMemoryRegions(currentModel)
		output.SetText(fmt.Sprintf("Model changed to %s\nSRAM: %s | Flash: %s", selected, regions.SRAMSize, regions.FlashSize))
	})
	modelSelect.SetSelected("Pico 2 (RP2350)")

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

		landmarks, err := serial.FetchLandmarks(portName)
		if err != nil {
			landmarksLabel.SetText(fmt.Sprintf("Landmarks: Could not connect - %v", err))
		} else {
			landmarksLabel.SetText(fmt.Sprintf("Landmarks: %s", landmarks))
		}
	})

	// Channel for background operations
	updateChan := make(chan ui.UIUpdate, 10)

	// Goroutine to handle UI updates on main thread
	go func() {
		for update := range updateChan {
			// Must use fyne.Do to safely update UI from goroutine
			text := update.Text
			fyne.Do(func() {
				output.SetText(text)
			})
		}
	}()

	// Build tabs - pass function to get current model
	getModel := func() config.PicoModel { return currentModel }
	readTab := ui.BuildReadMemoryTab(portEntry, output, updateChan, getModel)
	searchTab := ui.BuildSearchMemoryTab(portEntry, output, updateChan, getModel)

	tabs := container.NewAppTabs(readTab, searchTab)

	// Main layout
	portRow := container.NewBorder(nil, nil, widget.NewLabel("Serial Port:"), connectBtn, portEntry)
	modelRow := container.NewBorder(nil, nil, widget.NewLabel("Pico Model:"), nil, modelSelect)

	content := container.NewBorder(
		container.NewVBox(portRow, modelRow, tabs),
		landmarksLabel,
		nil,
		nil,
		container.NewScroll(output),
	)

	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(900, 850))
	myWindow.ShowAndRun()
}
