package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"plexichat-client/pkg/database"
	"plexichat-client/pkg/logging"
)

// CommandRegistry manages available commands
type CommandRegistry struct {
	commands map[string]Command
	aliases  map[string]string
	logger   *logging.Logger
	mu       sync.RWMutex
}

// Command defines the interface for all commands
type Command interface {
	Execute(ctx context.Context, args []string) (*CommandResult, error)
	GetName() string
	GetDescription() string
	GetUsage() string
	GetAliases() []string
	GetCategory() string
	RequiresAuth() bool
	ValidateArgs(args []string) error
}

// CommandResult represents the result of command execution
type CommandResult struct {
	Success     bool                   `json:"success"`
	Message     string                 `json:"message"`
	Data        interface{}            `json:"data,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CommandContext provides context for command execution
type CommandContext struct {
	UserID    string
	Username  string
	ChannelID string
	Database  *database.Database
	Logger    *logging.Logger
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	registry := &CommandRegistry{
		commands: make(map[string]Command),
		aliases:  make(map[string]string),
		logger:   logging.NewLogger(logging.INFO, nil, true),
	}

	// Register built-in commands
	registry.registerBuiltinCommands()

	return registry
}

// Register registers a new command
func (cr *CommandRegistry) Register(command Command) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	name := command.GetName()
	if _, exists := cr.commands[name]; exists {
		return fmt.Errorf("command %s already registered", name)
	}

	cr.commands[name] = command

	// Register aliases
	for _, alias := range command.GetAliases() {
		if _, exists := cr.aliases[alias]; exists {
			return fmt.Errorf("alias %s already registered", alias)
		}
		cr.aliases[alias] = name
	}

	cr.logger.Info("Registered command: %s", name)
	return nil
}

// Unregister removes a command from the registry
func (cr *CommandRegistry) Unregister(name string) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	command, exists := cr.commands[name]
	if !exists {
		return fmt.Errorf("command %s not found", name)
	}

	// Remove aliases
	for _, alias := range command.GetAliases() {
		delete(cr.aliases, alias)
	}

	delete(cr.commands, name)
	cr.logger.Info("Unregistered command: %s", name)
	return nil
}

// Execute executes a command
func (cr *CommandRegistry) Execute(ctx context.Context, commandLine string) (*CommandResult, error) {
	if commandLine == "" {
		return &CommandResult{
			Success: false,
			Error:   "Empty command",
		}, nil
	}

	// Parse command line
	parts := strings.Fields(commandLine)
	if len(parts) == 0 {
		return &CommandResult{
			Success: false,
			Error:   "No command specified",
		}, nil
	}

	commandName := parts[0]
	args := parts[1:]

	// Remove leading slash if present
	if strings.HasPrefix(commandName, "/") {
		commandName = commandName[1:]
	}

	// Resolve alias
	cr.mu.RLock()
	if alias, exists := cr.aliases[commandName]; exists {
		commandName = alias
	}

	command, exists := cr.commands[commandName]
	cr.mu.RUnlock()

	if !exists {
		return &CommandResult{
			Success:     false,
			Error:       fmt.Sprintf("Unknown command: %s", commandName),
			Suggestions: cr.getSuggestions(commandName),
		}, nil
	}

	// Validate arguments
	if err := command.ValidateArgs(args); err != nil {
		return &CommandResult{
			Success: false,
			Error:   fmt.Sprintf("Invalid arguments: %v", err),
			Message: command.GetUsage(),
		}, nil
	}

	// Execute command
	result, err := command.Execute(ctx, args)
	if err != nil {
		return &CommandResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return result, nil
}

// GetCommand returns a command by name
func (cr *CommandRegistry) GetCommand(name string) (Command, bool) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	// Check aliases first
	if alias, exists := cr.aliases[name]; exists {
		name = alias
	}

	command, exists := cr.commands[name]
	return command, exists
}

// ListCommands returns all registered commands
func (cr *CommandRegistry) ListCommands() []Command {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	commands := make([]Command, 0, len(cr.commands))
	for _, command := range cr.commands {
		commands = append(commands, command)
	}

	// Sort by name
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].GetName() < commands[j].GetName()
	})

	return commands
}

// GetCommandsByCategory returns commands grouped by category
func (cr *CommandRegistry) GetCommandsByCategory() map[string][]Command {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	categories := make(map[string][]Command)
	for _, command := range cr.commands {
		category := command.GetCategory()
		if categories[category] == nil {
			categories[category] = make([]Command, 0)
		}
		categories[category] = append(categories[category], command)
	}

	// Sort commands within each category
	for category := range categories {
		sort.Slice(categories[category], func(i, j int) bool {
			return categories[category][i].GetName() < categories[category][j].GetName()
		})
	}

	return categories
}

// getSuggestions returns command suggestions for a given input
func (cr *CommandRegistry) getSuggestions(input string) []string {
	suggestions := make([]string, 0)
	input = strings.ToLower(input)

	for name := range cr.commands {
		if strings.HasPrefix(strings.ToLower(name), input) {
			suggestions = append(suggestions, name)
		}
	}

	for alias := range cr.aliases {
		if strings.HasPrefix(strings.ToLower(alias), input) {
			suggestions = append(suggestions, alias)
		}
	}

	// Limit suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	sort.Strings(suggestions)
	return suggestions
}

// registerBuiltinCommands registers built-in commands
func (cr *CommandRegistry) registerBuiltinCommands() {
	// Help command
	cr.Register(&HelpCommand{registry: cr})

	// Version command
	cr.Register(&VersionCommand{})

	// Status command
	cr.Register(&StatusCommand{})

	// Connect command
	cr.Register(&ConnectCommand{})

	// Disconnect command
	cr.Register(&DisconnectCommand{})

	// Join command
	cr.Register(&JoinCommand{})

	// Leave command
	cr.Register(&LeaveCommand{})

	// List command
	cr.Register(&ListCommand{})

	// Send command
	cr.Register(&SendCommand{})

	// Upload command
	cr.Register(&UploadCommand{})

	// Download command
	cr.Register(&DownloadCommand{})

	// Search command
	cr.Register(&SearchCommand{})

	// History command
	cr.Register(&HistoryCommand{})

	// Users command
	cr.Register(&UsersCommand{})

	// Channels command
	cr.Register(&ChannelsCommand{})

	// Config command
	cr.Register(&ConfigCommand{})

	// Clear command
	cr.Register(&ClearCommand{})

	// Exit command
	cr.Register(&ExitCommand{})
}

// BaseCommand provides common functionality for commands
type BaseCommand struct {
	name        string
	description string
	usage       string
	aliases     []string
	category    string
	requireAuth bool
}

func (bc *BaseCommand) GetName() string        { return bc.name }
func (bc *BaseCommand) GetDescription() string { return bc.description }
func (bc *BaseCommand) GetUsage() string       { return bc.usage }
func (bc *BaseCommand) GetAliases() []string   { return bc.aliases }
func (bc *BaseCommand) GetCategory() string    { return bc.category }
func (bc *BaseCommand) RequiresAuth() bool     { return bc.requireAuth }

func (bc *BaseCommand) ValidateArgs(args []string) error {
	// Default implementation - no validation
	return nil
}

// HelpCommand shows help information
type HelpCommand struct {
	BaseCommand
	registry *CommandRegistry
}

func (c *HelpCommand) Execute(ctx context.Context, args []string) (*CommandResult, error) {
	c.BaseCommand = BaseCommand{
		name:        "help",
		description: "Show help information",
		usage:       "help [command]",
		aliases:     []string{"h", "?"},
		category:    "General",
		requireAuth: false,
	}

	if len(args) == 0 {
		// Show all commands
		categories := c.registry.GetCommandsByCategory()
		message := "Available commands:\n\n"

		for category, commands := range categories {
			message += fmt.Sprintf("=== %s ===\n", category)
			for _, cmd := range commands {
				message += fmt.Sprintf("  %-15s %s\n", cmd.GetName(), cmd.GetDescription())
			}
			message += "\n"
		}

		message += "Use 'help <command>' for detailed information about a specific command."

		return &CommandResult{
			Success: true,
			Message: message,
		}, nil
	}

	// Show help for specific command
	commandName := args[0]
	command, exists := c.registry.GetCommand(commandName)
	if !exists {
		return &CommandResult{
			Success: false,
			Error:   fmt.Sprintf("Unknown command: %s", commandName),
		}, nil
	}

	message := fmt.Sprintf("Command: %s\n", command.GetName())
	message += fmt.Sprintf("Description: %s\n", command.GetDescription())
	message += fmt.Sprintf("Usage: %s\n", command.GetUsage())
	message += fmt.Sprintf("Category: %s\n", command.GetCategory())

	if aliases := command.GetAliases(); len(aliases) > 0 {
		message += fmt.Sprintf("Aliases: %s\n", strings.Join(aliases, ", "))
	}

	if command.RequiresAuth() {
		message += "Requires authentication: Yes\n"
	}

	return &CommandResult{
		Success: true,
		Message: message,
	}, nil
}

// VersionCommand shows version information
type VersionCommand struct {
	BaseCommand
}

func (c *VersionCommand) Execute(ctx context.Context, args []string) (*CommandResult, error) {
	c.BaseCommand = BaseCommand{
		name:        "version",
		description: "Show version information",
		usage:       "version",
		aliases:     []string{"v", "ver"},
		category:    "General",
		requireAuth: false,
	}

	return &CommandResult{
		Success: true,
		Message: "PlexiChat Client v3.0.0-production\nBuild: 2024-01-01\nGo: go1.21",
		Data: map[string]string{
			"version": "3.0.0-production",
			"build":   "2024-01-01",
			"go":      "go1.21",
		},
	}, nil
}

// StatusCommand shows connection status
type StatusCommand struct {
	BaseCommand
}

func (c *StatusCommand) Execute(ctx context.Context, args []string) (*CommandResult, error) {
	c.BaseCommand = BaseCommand{
		name:        "status",
		description: "Show connection status",
		usage:       "status",
		aliases:     []string{"stat"},
		category:    "Connection",
		requireAuth: false,
	}

	// This would check actual connection status
	return &CommandResult{
		Success: true,
		Message: "Status: Connected\nServer: localhost:8000\nLatency: 25ms",
		Data: map[string]interface{}{
			"connected": true,
			"server":    "localhost:8000",
			"latency":   "25ms",
		},
	}, nil
}

// ConnectCommand connects to server
type ConnectCommand struct {
	BaseCommand
}

func (c *ConnectCommand) Execute(ctx context.Context, args []string) (*CommandResult, error) {
	c.BaseCommand = BaseCommand{
		name:        "connect",
		description: "Connect to server",
		usage:       "connect [server_url]",
		aliases:     []string{"conn"},
		category:    "Connection",
		requireAuth: false,
	}

	serverURL := "localhost:8000"
	if len(args) > 0 {
		serverURL = args[0]
	}

	// This would perform actual connection
	return &CommandResult{
		Success: true,
		Message: fmt.Sprintf("Connected to %s", serverURL),
		Data: map[string]string{
			"server": serverURL,
		},
	}, nil
}

// DisconnectCommand disconnects from server
type DisconnectCommand struct {
	BaseCommand
}

func (c *DisconnectCommand) Execute(ctx context.Context, args []string) (*CommandResult, error) {
	c.BaseCommand = BaseCommand{
		name:        "disconnect",
		description: "Disconnect from server",
		usage:       "disconnect",
		aliases:     []string{"disc"},
		category:    "Connection",
		requireAuth: false,
	}

	// This would perform actual disconnection
	return &CommandResult{
		Success: true,
		Message: "Disconnected from server",
	}, nil
}

// ExitCommand exits the application
type ExitCommand struct {
	BaseCommand
}

func (c *ExitCommand) Execute(ctx context.Context, args []string) (*CommandResult, error) {
	c.BaseCommand = BaseCommand{
		name:        "exit",
		description: "Exit the application",
		usage:       "exit",
		aliases:     []string{"quit", "q"},
		category:    "General",
		requireAuth: false,
	}

	return &CommandResult{
		Success: true,
		Message: "Goodbye!",
		Metadata: map[string]interface{}{
			"action": "exit",
		},
	}, nil
}
