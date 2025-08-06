package ui

import (
	"context"
	"fmt"
	"image/color"
	"sync"
	"time"

	"plexichat-client/pkg/logging"
)

// ComponentType represents different UI component types
type ComponentType string

const (
	ComponentButton      ComponentType = "button"
	ComponentInput       ComponentType = "input"
	ComponentLabel       ComponentType = "label"
	ComponentList        ComponentType = "list"
	ComponentCard        ComponentType = "card"
	ComponentModal       ComponentType = "modal"
	ComponentTooltip     ComponentType = "tooltip"
	ComponentProgress    ComponentType = "progress"
	ComponentTabs        ComponentType = "tabs"
	ComponentMenu        ComponentType = "menu"
	ComponentTable       ComponentType = "table"
	ComponentTree        ComponentType = "tree"
	ComponentChart       ComponentType = "chart"
	ComponentCalendar    ComponentType = "calendar"
	ComponentColorPicker ComponentType = "color_picker"
)

// ComponentState represents the state of a UI component
type ComponentState string

const (
	StateNormal   ComponentState = "normal"
	StateHover    ComponentState = "hover"
	StateFocus    ComponentState = "focus"
	StatePressed  ComponentState = "pressed"
	StateDisabled ComponentState = "disabled"
	StateActive   ComponentState = "active"
	StateLoading  ComponentState = "loading"
	StateError    ComponentState = "error"
	StateSuccess  ComponentState = "success"
	StateWarning  ComponentState = "warning"
)

// Animation represents UI animations
type UIAnimation struct {
	Type        string        `json:"type"`
	Duration    time.Duration `json:"duration"`
	Delay       time.Duration `json:"delay"`
	Easing      string        `json:"easing"`
	Loop        bool          `json:"loop"`
	Reverse     bool          `json:"reverse"`
	Properties  map[string]interface{} `json:"properties"`
}

// Component represents a base UI component
type Component struct {
	ID          string                 `json:"id"`
	Type        ComponentType          `json:"type"`
	State       ComponentState         `json:"state"`
	Visible     bool                   `json:"visible"`
	Enabled     bool                   `json:"enabled"`
	Position    Position               `json:"position"`
	Size        Size                   `json:"size"`
	Style       ComponentStyle         `json:"style"`
	Properties  map[string]interface{} `json:"properties"`
	Children    []*Component           `json:"children"`
	Parent      *Component             `json:"-"`
	EventHandlers map[string]func(Event) `json:"-"`
	Animations  []*UIAnimation         `json:"animations"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Position represents component position
type Position struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

// Size represents component dimensions
type Size struct {
	Width  float32 `json:"width"`
	Height float32 `json:"height"`
	MinWidth  float32 `json:"min_width"`
	MinHeight float32 `json:"min_height"`
	MaxWidth  float32 `json:"max_width"`
	MaxHeight float32 `json:"max_height"`
}

// ComponentStyle represents component styling
type ComponentStyle struct {
	BackgroundColor color.Color `json:"background_color"`
	ForegroundColor color.Color `json:"foreground_color"`
	BorderColor     color.Color `json:"border_color"`
	BorderWidth     float32     `json:"border_width"`
	BorderRadius    float32     `json:"border_radius"`
	Padding         Padding     `json:"padding"`
	Margin          Margin      `json:"margin"`
	FontFamily      string      `json:"font_family"`
	FontSize        float32     `json:"font_size"`
	FontWeight      string      `json:"font_weight"`
	TextAlign       string      `json:"text_align"`
	Opacity         float32     `json:"opacity"`
	Shadow          Shadow      `json:"shadow"`
	Transform       Transform   `json:"transform"`
}

// Padding represents component padding
type Padding struct {
	Top    float32 `json:"top"`
	Right  float32 `json:"right"`
	Bottom float32 `json:"bottom"`
	Left   float32 `json:"left"`
}

// Margin represents component margin
type Margin struct {
	Top    float32 `json:"top"`
	Right  float32 `json:"right"`
	Bottom float32 `json:"bottom"`
	Left   float32 `json:"left"`
}

// Shadow represents component shadow
type Shadow struct {
	OffsetX float32     `json:"offset_x"`
	OffsetY float32     `json:"offset_y"`
	Blur    float32     `json:"blur"`
	Spread  float32     `json:"spread"`
	Color   color.Color `json:"color"`
}

// Transform represents component transformations
type Transform struct {
	TranslateX float32 `json:"translate_x"`
	TranslateY float32 `json:"translate_y"`
	ScaleX     float32 `json:"scale_x"`
	ScaleY     float32 `json:"scale_y"`
	Rotation   float32 `json:"rotation"`
	SkewX      float32 `json:"skew_x"`
	SkewY      float32 `json:"skew_y"`
}

// Event represents UI events
type Event struct {
	Type      string                 `json:"type"`
	Target    *Component             `json:"target"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Bubbles   bool                   `json:"bubbles"`
	Cancelled bool                   `json:"cancelled"`
}

// ComponentManager manages UI components
type ComponentManager struct {
	components map[string]*Component
	root       *Component
	theme      *ThemeConfig
	logger     *logging.Logger
	mu         sync.RWMutex
}

// NewComponentManager creates a new component manager
func NewComponentManager(theme *ThemeConfig) *ComponentManager {
	return &ComponentManager{
		components: make(map[string]*Component),
		theme:      theme,
		logger:     logging.NewLogger(logging.INFO, nil, true),
	}
}

// CreateComponent creates a new component
func (cm *ComponentManager) CreateComponent(componentType ComponentType, id string) *Component {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	component := &Component{
		ID:            id,
		Type:          componentType,
		State:         StateNormal,
		Visible:       true,
		Enabled:       true,
		Properties:    make(map[string]interface{}),
		EventHandlers: make(map[string]func(Event)),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Apply default styling based on theme
	component.Style = cm.getDefaultStyle(componentType)

	cm.components[id] = component
	cm.logger.Debug("Created component: %s (%s)", id, componentType)

	return component
}

// GetComponent retrieves a component by ID
func (cm *ComponentManager) GetComponent(id string) (*Component, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	component, exists := cm.components[id]
	return component, exists
}

// UpdateComponent updates a component
func (cm *ComponentManager) UpdateComponent(id string, updates map[string]interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	component, exists := cm.components[id]
	if !exists {
		return fmt.Errorf("component %s not found", id)
	}

	// Apply updates
	for key, value := range updates {
		component.Properties[key] = value
	}

	component.UpdatedAt = time.Now()
	cm.logger.Debug("Updated component: %s", id)

	return nil
}

// DeleteComponent removes a component
func (cm *ComponentManager) DeleteComponent(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	component, exists := cm.components[id]
	if !exists {
		return fmt.Errorf("component %s not found", id)
	}

	// Remove from parent
	if component.Parent != nil {
		cm.removeChild(component.Parent, component)
	}

	// Remove children
	for _, child := range component.Children {
		delete(cm.components, child.ID)
	}

	delete(cm.components, id)
	cm.logger.Debug("Deleted component: %s", id)

	return nil
}

// AddChild adds a child component
func (cm *ComponentManager) AddChild(parentID, childID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	parent, exists := cm.components[parentID]
	if !exists {
		return fmt.Errorf("parent component %s not found", parentID)
	}

	child, exists := cm.components[childID]
	if !exists {
		return fmt.Errorf("child component %s not found", childID)
	}

	// Remove from current parent if any
	if child.Parent != nil {
		cm.removeChild(child.Parent, child)
	}

	// Add to new parent
	parent.Children = append(parent.Children, child)
	child.Parent = parent

	cm.logger.Debug("Added child %s to parent %s", childID, parentID)
	return nil
}

// removeChild removes a child from parent
func (cm *ComponentManager) removeChild(parent, child *Component) {
	for i, c := range parent.Children {
		if c.ID == child.ID {
			parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
			break
		}
	}
	child.Parent = nil
}

// getDefaultStyle returns default styling for a component type
func (cm *ComponentManager) getDefaultStyle(componentType ComponentType) ComponentStyle {
	if cm.theme == nil {
		return ComponentStyle{
			Opacity: 1.0,
			Transform: Transform{
				ScaleX: 1.0,
				ScaleY: 1.0,
			},
		}
	}

	style := ComponentStyle{
		FontFamily: cm.theme.Typography.FontFamily,
		FontSize:   cm.theme.Typography.FontSize,
		FontWeight: cm.theme.Typography.FontWeight,
		Opacity:    1.0,
		Transform: Transform{
			ScaleX: 1.0,
			ScaleY: 1.0,
		},
		Padding: Padding{
			Top:    cm.theme.Spacing.Padding.Medium,
			Right:  cm.theme.Spacing.Padding.Medium,
			Bottom: cm.theme.Spacing.Padding.Medium,
			Left:   cm.theme.Spacing.Padding.Medium,
		},
		Margin: Margin{
			Top:    cm.theme.Spacing.Margin.Small,
			Right:  cm.theme.Spacing.Margin.Small,
			Bottom: cm.theme.Spacing.Margin.Small,
			Left:   cm.theme.Spacing.Margin.Small,
		},
		BorderRadius: cm.theme.Spacing.BorderRadius.Medium,
	}

	// Customize based on component type
	switch componentType {
	case ComponentButton:
		style.BackgroundColor = parseColor(cm.theme.Colors.Primary)
		style.ForegroundColor = parseColor(cm.theme.Colors.OnPrimary)
		style.BorderWidth = 0
	case ComponentInput:
		style.BackgroundColor = parseColor(cm.theme.Colors.Surface)
		style.ForegroundColor = parseColor(cm.theme.Colors.OnSurface)
		style.BorderColor = parseColor(cm.theme.Colors.Border)
		style.BorderWidth = 1
	case ComponentCard:
		style.BackgroundColor = parseColor(cm.theme.Colors.Surface)
		style.ForegroundColor = parseColor(cm.theme.Colors.OnSurface)
		style.Shadow = Shadow{
			OffsetX: 0,
			OffsetY: 2,
			Blur:    4,
			Spread:  0,
			Color:   parseColor(cm.theme.Colors.Shadow),
		}
	}

	return style
}

// AnimationEngine handles component animations
type AnimationEngine struct {
	animations map[string]*UIAnimation
	running    map[string]context.CancelFunc
	logger     *logging.Logger
	mu         sync.RWMutex
}

// NewAnimationEngine creates a new animation engine
func NewAnimationEngine() *AnimationEngine {
	return &AnimationEngine{
		animations: make(map[string]*UIAnimation),
		running:    make(map[string]context.CancelFunc),
		logger:     logging.NewLogger(logging.INFO, nil, true),
	}
}

// StartAnimation starts an animation
func (ae *AnimationEngine) StartAnimation(componentID string, animation *UIAnimation) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	// Stop existing animation if running
	if cancel, exists := ae.running[componentID]; exists {
		cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	ae.running[componentID] = cancel
	ae.animations[componentID] = animation

	// Start animation in goroutine
	go ae.runAnimation(ctx, componentID, animation)

	ae.logger.Debug("Started animation for component: %s", componentID)
	return nil
}

// StopAnimation stops an animation
func (ae *AnimationEngine) StopAnimation(componentID string) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	if cancel, exists := ae.running[componentID]; exists {
		cancel()
		delete(ae.running, componentID)
		delete(ae.animations, componentID)
		ae.logger.Debug("Stopped animation for component: %s", componentID)
	}
}

// runAnimation executes an animation
func (ae *AnimationEngine) runAnimation(ctx context.Context, componentID string, animation *UIAnimation) {
	defer func() {
		ae.mu.Lock()
		delete(ae.running, componentID)
		delete(ae.animations, componentID)
		ae.mu.Unlock()
	}()

	// Wait for delay
	if animation.Delay > 0 {
		select {
		case <-time.After(animation.Delay):
		case <-ctx.Done():
			return
		}
	}

	// Simple animation implementation
	// In a real implementation, this would interpolate values over time
	ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
	defer ticker.Stop()

	startTime := time.Now()
	duration := animation.Duration

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime)
			progress := float64(elapsed) / float64(duration)

			if progress >= 1.0 {
				// Animation complete
				if animation.Loop {
					startTime = time.Now()
					continue
				}
				return
			}

			// Apply easing function
			easedProgress := ae.applyEasing(progress, animation.Easing)

			// Update component properties based on animation
			ae.updateComponentForAnimation(componentID, animation, easedProgress)

		case <-ctx.Done():
			return
		}
	}
}

// applyEasing applies easing function to progress
func (ae *AnimationEngine) applyEasing(progress float64, easing string) float64 {
	switch easing {
	case "ease-in":
		return progress * progress
	case "ease-out":
		return 1 - (1-progress)*(1-progress)
	case "ease-in-out":
		if progress < 0.5 {
			return 2 * progress * progress
		}
		return 1 - 2*(1-progress)*(1-progress)
	default: // linear
		return progress
	}
}

// updateComponentForAnimation updates component properties during animation
func (ae *AnimationEngine) updateComponentForAnimation(componentID string, animation *UIAnimation, progress float64) {
	// This would update the actual component properties
	// For now, just log the progress
	ae.logger.Debug("Animation progress for %s: %.2f", componentID, progress)
}
