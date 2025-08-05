package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("PlexiChat GUI Test")
	myWindow.Resize(fyne.NewSize(400, 300))

	hello := widget.NewLabel("PlexiChat GUI is working!")
	content := widget.NewVBox(
		hello,
		widget.NewButton("Test Button", func() {
			hello.SetText("Button clicked! GUI is functional.")
		}),
	)

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}
