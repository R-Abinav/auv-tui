// main.go
package main

import (
	"github.com/rivo/tview"
)

func main() {
	// Create the main application object.
	app := tview.NewApplication()

	// --- UI Components ---

	// A text view for logs and command output.
	logView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	logView.SetBorder(true).SetTitle("Output / Logs")

	// The main menu list.
	menu := tview.NewList().
		AddItem("Connect to Jetson", "Establish an SSH connection", '1', nil).
		AddItem("Start ROS Core", "Run roscore in a background session", '2', nil).
		AddItem("Start Rosserial", "Run the rosserial client", '3', nil).
		AddItem("Run a Script", "Select and run a ROS node", '4', nil).
		AddItem("Quit", "Exit the application", 'q', func() {
			app.Stop()
		})
	menu.SetBorder(true).SetTitle("AUV Control")

	// --- Layout ---

	// A Flexbox layout to arrange the components.
	flex := tview.NewFlex().
		AddItem(menu, 0, 1, true). // Left item: menu, 1 part of the width, with focus.
		AddItem(logView, 0, 3, false) // Right item: logView, 3 parts of the width.

	// --- Application Start ---

	// Set the root of the application and run it.
	if err := app.SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}