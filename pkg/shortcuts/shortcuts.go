package shortcuts

import (
	"fmt"
	"strings"

	"plexichat-client/pkg/logging"
)

// KeyCombination represents a keyboard shortcut
type KeyCombination struct {
	Key       string `json:"key"`        // The main key (e.g., "Enter", "F1", "a")
	Ctrl      bool   `json:"ctrl"`       // Ctrl modifier
	Alt       bool   `json:"alt"`        // Alt modifier
	Shift     bool   `json:"shift"`      // Shift modifier
	Meta      bool   `json:"meta"`       // Meta/Cmd modifier (macOS)
}

// String returns a human-readable representation of the key combination
func (kc *KeyCombination) String() string {
	var parts []string
	
	if kc.Ctrl {
		parts = append(parts, "Ctrl")
	}
	if kc.Alt {
		parts = append(parts, "Alt")
	}
	if kc.Shift {
		parts = append(parts, "Shift")
	}
	if kc.Meta {
		parts = append(parts, "Cmd")
	}
	
	parts = append(parts, kc.Key)
	return strings.Join(parts, "+")
}

// Shortcut represents a keyboard shortcut with its action
type Shortcut struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Combination *KeyCombination `json:"combination"`
	Action      func()          `json:"-"` // Function to execute
	Enabled     bool            `json:"enabled"`
}

// ShortcutCategory represents a category of shortcuts
type ShortcutCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ShortcutManager manages keyboard shortcuts
type ShortcutManager struct {
	shortcuts  map[string]*Shortcut
	categories map[string]*ShortcutCategory
	keyMap     map[string]*Shortcut // Maps key combination strings to shortcuts
	logger     *logging.Logger
	enabled    bool
}

// NewShortcutManager creates a new shortcut manager
func NewShortcutManager() *ShortcutManager {
	sm := &ShortcutManager{
		shortcuts:  make(map[string]*Shortcut),
		categories: make(map[string]*ShortcutCategory),
		keyMap:     make(map[string]*Shortcut),
		logger:     logging.NewLogger(logging.INFO, nil, true),
		enabled:    true,
	}
	
	// Initialize default categories
	sm.initializeCategories()
	
	// Initialize default shortcuts
	sm.initializeDefaultShortcuts()
	
	return sm
}

// initializeCategories sets up default shortcut categories
func (sm *ShortcutManager) initializeCategories() {
	categories := []*ShortcutCategory{
		{ID: "general", Name: "General", Description: "General application shortcuts"},
		{ID: "navigation", Name: "Navigation", Description: "Navigation and movement shortcuts"},
		{ID: "messaging", Name: "Messaging", Description: "Message and chat shortcuts"},
		{ID: "search", Name: "Search", Description: "Search and filtering shortcuts"},
		{ID: "files", Name: "Files", Description: "File management shortcuts"},
		{ID: "settings", Name: "Settings", Description: "Configuration and settings shortcuts"},
	}
	
	for _, category := range categories {
		sm.categories[category.ID] = category
	}
}

// initializeDefaultShortcuts sets up default keyboard shortcuts
func (sm *ShortcutManager) initializeDefaultShortcuts() {
	defaultShortcuts := []*Shortcut{
		// General shortcuts
		{
			ID: "quit", Name: "Quit Application", Description: "Exit the application",
			Category: "general", Combination: &KeyCombination{Key: "q", Ctrl: true}, Enabled: true,
		},
		{
			ID: "help", Name: "Show Help", Description: "Display help information",
			Category: "general", Combination: &KeyCombination{Key: "F1"}, Enabled: true,
		},
		{
			ID: "refresh", Name: "Refresh", Description: "Refresh current view",
			Category: "general", Combination: &KeyCombination{Key: "F5"}, Enabled: true,
		},
		{
			ID: "settings", Name: "Open Settings", Description: "Open application settings",
			Category: "settings", Combination: &KeyCombination{Key: "comma", Ctrl: true}, Enabled: true,
		},
		
		// Navigation shortcuts
		{
			ID: "next_conversation", Name: "Next Conversation", Description: "Switch to next conversation",
			Category: "navigation", Combination: &KeyCombination{Key: "Tab", Ctrl: true}, Enabled: true,
		},
		{
			ID: "prev_conversation", Name: "Previous Conversation", Description: "Switch to previous conversation",
			Category: "navigation", Combination: &KeyCombination{Key: "Tab", Ctrl: true, Shift: true}, Enabled: true,
		},
		{
			ID: "scroll_up", Name: "Scroll Up", Description: "Scroll up in message history",
			Category: "navigation", Combination: &KeyCombination{Key: "Page_Up"}, Enabled: true,
		},
		{
			ID: "scroll_down", Name: "Scroll Down", Description: "Scroll down in message history",
			Category: "navigation", Combination: &KeyCombination{Key: "Page_Down"}, Enabled: true,
		},
		{
			ID: "go_to_top", Name: "Go to Top", Description: "Jump to top of conversation",
			Category: "navigation", Combination: &KeyCombination{Key: "Home", Ctrl: true}, Enabled: true,
		},
		{
			ID: "go_to_bottom", Name: "Go to Bottom", Description: "Jump to bottom of conversation",
			Category: "navigation", Combination: &KeyCombination{Key: "End", Ctrl: true}, Enabled: true,
		},
		
		// Messaging shortcuts
		{
			ID: "send_message", Name: "Send Message", Description: "Send the current message",
			Category: "messaging", Combination: &KeyCombination{Key: "Return"}, Enabled: true,
		},
		{
			ID: "new_line", Name: "New Line", Description: "Insert new line in message",
			Category: "messaging", Combination: &KeyCombination{Key: "Return", Shift: true}, Enabled: true,
		},
		{
			ID: "emoji_picker", Name: "Emoji Picker", Description: "Open emoji picker",
			Category: "messaging", Combination: &KeyCombination{Key: "e", Ctrl: true}, Enabled: true,
		},
		{
			ID: "attach_file", Name: "Attach File", Description: "Attach a file to message",
			Category: "messaging", Combination: &KeyCombination{Key: "u", Ctrl: true}, Enabled: true,
		},
		{
			ID: "edit_last_message", Name: "Edit Last Message", Description: "Edit your last message",
			Category: "messaging", Combination: &KeyCombination{Key: "Up"}, Enabled: true,
		},
		
		// Search shortcuts
		{
			ID: "search", Name: "Search", Description: "Open search dialog",
			Category: "search", Combination: &KeyCombination{Key: "f", Ctrl: true}, Enabled: true,
		},
		{
			ID: "search_next", Name: "Search Next", Description: "Find next search result",
			Category: "search", Combination: &KeyCombination{Key: "F3"}, Enabled: true,
		},
		{
			ID: "search_prev", Name: "Search Previous", Description: "Find previous search result",
			Category: "search", Combination: &KeyCombination{Key: "F3", Shift: true}, Enabled: true,
		},
		{
			ID: "filter_messages", Name: "Filter Messages", Description: "Open message filter",
			Category: "search", Combination: &KeyCombination{Key: "f", Ctrl: true, Shift: true}, Enabled: true,
		},
		
		// File shortcuts
		{
			ID: "open_file_manager", Name: "File Manager", Description: "Open file manager",
			Category: "files", Combination: &KeyCombination{Key: "o", Ctrl: true}, Enabled: true,
		},
		{
			ID: "download_file", Name: "Download File", Description: "Download selected file",
			Category: "files", Combination: &KeyCombination{Key: "d", Ctrl: true}, Enabled: true,
		},
	}
	
	for _, shortcut := range defaultShortcuts {
		sm.RegisterShortcut(shortcut)
	}
}

// RegisterShortcut registers a new keyboard shortcut
func (sm *ShortcutManager) RegisterShortcut(shortcut *Shortcut) error {
	if shortcut.ID == "" {
		return fmt.Errorf("shortcut ID cannot be empty")
	}
	
	if shortcut.Combination == nil {
		return fmt.Errorf("shortcut combination cannot be nil")
	}
	
	// Check for conflicts
	keyStr := shortcut.Combination.String()
	if existing, exists := sm.keyMap[keyStr]; exists {
		return fmt.Errorf("key combination %s already assigned to %s", keyStr, existing.Name)
	}
	
	sm.shortcuts[shortcut.ID] = shortcut
	sm.keyMap[keyStr] = shortcut
	
	sm.logger.Debug("Registered shortcut: %s (%s)", shortcut.Name, keyStr)
	return nil
}

// UnregisterShortcut removes a keyboard shortcut
func (sm *ShortcutManager) UnregisterShortcut(id string) error {
	shortcut, exists := sm.shortcuts[id]
	if !exists {
		return fmt.Errorf("shortcut %s not found", id)
	}
	
	keyStr := shortcut.Combination.String()
	delete(sm.shortcuts, id)
	delete(sm.keyMap, keyStr)
	
	sm.logger.Debug("Unregistered shortcut: %s", shortcut.Name)
	return nil
}

// HandleKeyPress processes a key press and executes the associated action
func (sm *ShortcutManager) HandleKeyPress(key string, ctrl, alt, shift, meta bool) bool {
	if !sm.enabled {
		return false
	}
	
	combination := &KeyCombination{
		Key:   key,
		Ctrl:  ctrl,
		Alt:   alt,
		Shift: shift,
		Meta:  meta,
	}
	
	keyStr := combination.String()
	shortcut, exists := sm.keyMap[keyStr]
	
	if !exists || !shortcut.Enabled || shortcut.Action == nil {
		return false
	}
	
	sm.logger.Debug("Executing shortcut: %s (%s)", shortcut.Name, keyStr)
	
	// Execute the action in a goroutine to prevent blocking
	go func() {
		defer func() {
			if r := recover(); r != nil {
				sm.logger.Error("Shortcut action panicked: %v", r)
			}
		}()
		shortcut.Action()
	}()
	
	return true
}

// GetShortcut returns a shortcut by ID
func (sm *ShortcutManager) GetShortcut(id string) (*Shortcut, bool) {
	shortcut, exists := sm.shortcuts[id]
	return shortcut, exists
}

// GetShortcutsByCategory returns all shortcuts in a category
func (sm *ShortcutManager) GetShortcutsByCategory(categoryID string) []*Shortcut {
	var shortcuts []*Shortcut
	
	for _, shortcut := range sm.shortcuts {
		if shortcut.Category == categoryID {
			shortcuts = append(shortcuts, shortcut)
		}
	}
	
	return shortcuts
}

// GetAllShortcuts returns all registered shortcuts
func (sm *ShortcutManager) GetAllShortcuts() map[string]*Shortcut {
	// Return a copy to prevent external modification
	shortcuts := make(map[string]*Shortcut)
	for k, v := range sm.shortcuts {
		shortcuts[k] = v
	}
	return shortcuts
}

// GetCategories returns all shortcut categories
func (sm *ShortcutManager) GetCategories() map[string]*ShortcutCategory {
	// Return a copy to prevent external modification
	categories := make(map[string]*ShortcutCategory)
	for k, v := range sm.categories {
		categories[k] = v
	}
	return categories
}

// SetShortcutEnabled enables or disables a shortcut
func (sm *ShortcutManager) SetShortcutEnabled(id string, enabled bool) error {
	shortcut, exists := sm.shortcuts[id]
	if !exists {
		return fmt.Errorf("shortcut %s not found", id)
	}
	
	shortcut.Enabled = enabled
	sm.logger.Debug("Shortcut %s %s", shortcut.Name, map[bool]string{true: "enabled", false: "disabled"}[enabled])
	return nil
}

// SetEnabled enables or disables the entire shortcut system
func (sm *ShortcutManager) SetEnabled(enabled bool) {
	sm.enabled = enabled
	sm.logger.Info("Shortcut system %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

// IsEnabled returns whether the shortcut system is enabled
func (sm *ShortcutManager) IsEnabled() bool {
	return sm.enabled
}

// UpdateShortcut updates an existing shortcut's key combination
func (sm *ShortcutManager) UpdateShortcut(id string, newCombination *KeyCombination) error {
	shortcut, exists := sm.shortcuts[id]
	if !exists {
		return fmt.Errorf("shortcut %s not found", id)
	}
	
	// Remove old key mapping
	oldKeyStr := shortcut.Combination.String()
	delete(sm.keyMap, oldKeyStr)
	
	// Check for conflicts with new combination
	newKeyStr := newCombination.String()
	if existing, exists := sm.keyMap[newKeyStr]; exists {
		// Restore old mapping
		sm.keyMap[oldKeyStr] = shortcut
		return fmt.Errorf("key combination %s already assigned to %s", newKeyStr, existing.Name)
	}
	
	// Update shortcut
	shortcut.Combination = newCombination
	sm.keyMap[newKeyStr] = shortcut
	
	sm.logger.Debug("Updated shortcut %s: %s -> %s", shortcut.Name, oldKeyStr, newKeyStr)
	return nil
}

// GetHelpText returns formatted help text for all shortcuts
func (sm *ShortcutManager) GetHelpText() string {
	var help strings.Builder
	
	help.WriteString("Keyboard Shortcuts:\n")
	help.WriteString(strings.Repeat("=", 50) + "\n\n")
	
	for _, category := range sm.categories {
		shortcuts := sm.GetShortcutsByCategory(category.ID)
		if len(shortcuts) == 0 {
			continue
		}
		
		help.WriteString(fmt.Sprintf("%s:\n", category.Name))
		help.WriteString(strings.Repeat("-", len(category.Name)+1) + "\n")
		
		for _, shortcut := range shortcuts {
			if shortcut.Enabled {
				help.WriteString(fmt.Sprintf("  %-20s %s\n", 
					shortcut.Combination.String(), shortcut.Description))
			}
		}
		help.WriteString("\n")
	}
	
	return help.String()
}
