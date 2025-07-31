package main

import (
"context"
"os"
"strings"

"fyne.io/fyne/v2"
"fyne.io/fyne/v2/app"
"fyne.io/fyne/v2/container"
"fyne.io/fyne/v2/dialog"
"fyne.io/fyne/v2/widget"

"plexichat-client/pkg/client"
)

// Version information
var (
version   = "1.0.1-gui"
commit    = "unknown"
buildTime = "unknown"
)

// lastServerFile is a file to persist last-used server address
const lastServerFile = "last_server.txt"

func loadLastServer() string {
b, err := os.ReadFile(lastServerFile)
if err != nil {
return ""
}
return strings.TrimSpace(string(b))
}

func saveLastServer(addr string) {
_ = os.WriteFile(lastServerFile, []byte(addr), 0600)
}

func main() {
myApp := app.New()

// Create main window
win := myApp.NewWindow("PlexiChat Client v" + version)
win.Resize(fyne.NewSize(1200, 800))
win.CenterOnScreen()

// Load last server or use default
serverAddr := loadLastServer()
if serverAddr == "" {
serverAddr = "http://localhost:8000"
}

// Create simple login screen
content := createSimpleLoginScreen(win, serverAddr)
win.SetContent(content)

win.ShowAndRun()
}

func createSimpleLoginScreen(win fyne.Window, serverAddr string) fyne.CanvasObject {
// Server URL entry
serverEntry := widget.NewEntry()
serverEntry.SetText(serverAddr)
serverEntry.SetPlaceHolder("PlexiChat Server URL")

// Username entry
usernameEntry := widget.NewEntry()
usernameEntry.SetPlaceHolder("Username")

// Password entry
passwordEntry := widget.NewPasswordEntry()
passwordEntry.SetPlaceHolder("Password")

// Status label
statusLabel := widget.NewLabel("Ready to connect")

// Connect button
connectBtn := widget.NewButton("Connect", func() {
statusLabel.SetText("Connecting...")

// Save server address
saveLastServer(serverEntry.Text)

// Create client and test connection
c := client.NewClient(serverEntry.Text)
ctx := context.Background()

// Test health endpoint
health, err := c.Health(ctx)
if err != nil {
statusLabel.SetText("Connection failed: " + err.Error())
return
}

statusLabel.SetText("Connected! Server: " + health.Version)

// Show success dialog
dialog.ShowInformation("Success", 
"Successfully connected to PlexiChat server!\n\nServer Version: "+health.Version+
"\nStatus: "+health.Status, win)
})

// Test button
testBtn := widget.NewButton("Test Connection", func() {
statusLabel.SetText("Testing connection...")

c := client.NewClient(serverEntry.Text)
ctx := context.Background()

health, err := c.Health(ctx)
if err != nil {
statusLabel.SetText("Test failed: " + err.Error())
dialog.ShowError(err, win)
return
}

statusLabel.SetText("Test successful! Server: " + health.Version)
dialog.ShowInformation("Connection Test", 
"Connection test successful!\n\nServer: "+health.Version+
"\nStatus: "+health.Status+
"\nTimestamp: "+health.Timestamp, win)
})

// Layout
form := container.NewVBox(
widget.NewCard("PlexiChat Client", "Connect to your PlexiChat server", 
container.NewVBox(
widget.NewLabel("Server Configuration:"),
serverEntry,
widget.NewSeparator(),
widget.NewLabel("Authentication:"),
usernameEntry,
passwordEntry,
widget.NewSeparator(),
container.NewHBox(testBtn, connectBtn),
widget.NewSeparator(),
statusLabel,
),
),
)

return container.NewBorder(nil, nil, nil, nil, form)
}
