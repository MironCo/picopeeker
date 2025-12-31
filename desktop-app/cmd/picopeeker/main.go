package main

import (
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

	// Build tabs
	readTab := ui.BuildReadMemoryTab(portEntry, output, updateChan)
	searchTab := ui.BuildSearchMemoryTab(portEntry, output, updateChan)
	monitorTab := ui.BuildMonitorTab(portEntry, output, updateChan)

	tabs := container.NewAppTabs(readTab, searchTab, monitorTab)

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
