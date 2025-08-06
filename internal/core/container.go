// Package core provides the dependency injection container and core services
package core

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"plexichat-client/internal/interfaces"
	"plexichat-client/pkg/logging"
)

// Container is a dependency injection container with lifecycle management
type Container struct {
	mu           sync.RWMutex
	services     map[string]*ServiceDefinition
	instances    map[string]interface{}
	singletons   map[string]interface{}
	interceptors []Interceptor
	logger       interfaces.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	started      bool
	shutdownCh   chan struct{}
}

// ServiceDefinition defines how a service should be created and managed
type ServiceDefinition struct {
	Name         string
	Factory      interface{}
	Singleton    bool
	Dependencies []string
	Tags         []string
	Lifecycle    ServiceLifecycle
	Config       map[string]interface{}
	Priority     int
	Timeout      time.Duration
}

// ServiceLifecycle defines the lifecycle hooks for a service
type ServiceLifecycle struct {
	OnCreate    func(instance interface{}) error
	OnStart     func(instance interface{}) error
	OnStop      func(instance interface{}) error
	OnDestroy   func(instance interface{}) error
	HealthCheck func(instance interface{}) error
}

// Interceptor allows modification of service creation and method calls
type Interceptor interface {
	BeforeCreate(name string, def *ServiceDefinition) error
	AfterCreate(name string, instance interface{}) error
	BeforeInvoke(instance interface{}, method string, args []interface{}) error
	AfterInvoke(instance interface{}, method string, result []interface{}) error
}

// NewContainer creates a new dependency injection container
func NewContainer(ctx context.Context) *Container {
	containerCtx, cancel := context.WithCancel(ctx)

	container := &Container{
		services:     make(map[string]*ServiceDefinition),
		instances:    make(map[string]interface{}),
		singletons:   make(map[string]interface{}),
		interceptors: make([]Interceptor, 0),
		logger:       logging.GetLogger("container"),
		ctx:          containerCtx,
		cancel:       cancel,
		shutdownCh:   make(chan struct{}),
	}

	// Register core services
	container.registerCoreServices()

	return container
}

// Register registers a service definition in the container
func (c *Container) Register(name string, factory interface{}, options ...ServiceOption) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return fmt.Errorf("cannot register service '%s': container already started", name)
	}

	def := &ServiceDefinition{
		Name:         name,
		Factory:      factory,
		Singleton:    false,
		Dependencies: make([]string, 0),
		Tags:         make([]string, 0),
		Config:       make(map[string]interface{}),
		Priority:     0,
		Timeout:      30 * time.Second,
	}

	// Apply options
	for _, option := range options {
		option(def)
	}

	// Validate factory
	if err := c.validateFactory(factory); err != nil {
		return fmt.Errorf("invalid factory for service '%s': %w", name, err)
	}

	c.services[name] = def
	c.logger.Debug("Registered service", "name", name, "singleton", def.Singleton)

	return nil
}

// Get retrieves a service instance from the container
func (c *Container) Get(name string) (interface{}, error) {
	c.mu.RLock()
	def, exists := c.services[name]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service '%s' not found", name)
	}

	// Check if singleton instance exists
	if def.Singleton {
		c.mu.RLock()
		if instance, exists := c.singletons[name]; exists {
			c.mu.RUnlock()
			return instance, nil
		}
		c.mu.RUnlock()
	}

	// Create new instance
	instance, err := c.createInstance(def)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance of service '%s': %w", name, err)
	}

	// Store singleton
	if def.Singleton {
		c.mu.Lock()
		c.singletons[name] = instance
		c.mu.Unlock()
	}

	return instance, nil
}

// MustGet retrieves a service instance and panics if not found
func (c *Container) MustGet(name string) interface{} {
	instance, err := c.Get(name)
	if err != nil {
		panic(fmt.Sprintf("failed to get service '%s': %v", name, err))
	}
	return instance
}

// GetByType retrieves all services that implement the given interface type
func (c *Container) GetByType(interfaceType reflect.Type) ([]interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var instances []interface{}

	for name := range c.services {
		instance, err := c.Get(name)
		if err != nil {
			continue
		}

		instanceType := reflect.TypeOf(instance)
		if instanceType.Implements(interfaceType) {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

// GetByTag retrieves all services with the given tag
func (c *Container) GetByTag(tag string) ([]interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var instances []interface{}

	for name, def := range c.services {
		for _, serviceTag := range def.Tags {
			if serviceTag == tag {
				instance, err := c.Get(name)
				if err != nil {
					c.logger.Error("Failed to get tagged service", "name", name, "tag", tag, "error", err)
					continue
				}
				instances = append(instances, instance)
				break
			}
		}
	}

	return instances, nil
}

// Start starts all registered services in dependency order
func (c *Container) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return fmt.Errorf("container already started")
	}

	c.logger.Info("Starting container with services", "count", len(c.services))

	// Sort services by priority and dependencies
	startOrder, err := c.calculateStartOrder()
	if err != nil {
		return fmt.Errorf("failed to calculate start order: %w", err)
	}

	// Start services in order
	for _, name := range startOrder {
		def := c.services[name]

		c.logger.Debug("Starting service", "name", name)

		// Get or create instance
		instance, err := c.Get(name)
		if err != nil {
			return fmt.Errorf("failed to get service '%s': %w", name, err)
		}

		// Call lifecycle hooks
		if def.Lifecycle.OnStart != nil {
			if err := def.Lifecycle.OnStart(instance); err != nil {
				return fmt.Errorf("failed to start service '%s': %w", name, err)
			}
		}

		c.logger.Info("Started service", "name", name)
	}

	c.started = true
	c.logger.Info("Container started successfully")

	// Start health check routine
	go c.healthCheckRoutine()

	return nil
}

// Stop stops all services in reverse dependency order
func (c *Container) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}

	c.logger.Info("Stopping container")

	// Calculate stop order (reverse of start order)
	startOrder, err := c.calculateStartOrder()
	if err != nil {
		c.logger.Error("Failed to calculate stop order", "error", err)
		// Continue with best effort shutdown
	}

	// Reverse the order for shutdown
	for i := len(startOrder) - 1; i >= 0; i-- {
		name := startOrder[i]
		def := c.services[name]

		c.logger.Debug("Stopping service", "name", name)

		// Get instance if it exists
		var instance interface{}
		if def.Singleton {
			if inst, exists := c.singletons[name]; exists {
				instance = inst
			}
		}

		if instance != nil && def.Lifecycle.OnStop != nil {
			if err := def.Lifecycle.OnStop(instance); err != nil {
				c.logger.Error("Failed to stop service", "name", name, "error", err)
			}
		}

		c.logger.Debug("Stopped service", "name", name)
	}

	c.started = false
	close(c.shutdownCh)
	c.cancel()

	c.logger.Info("Container stopped")
	return nil
}

// AddInterceptor adds a service interceptor
func (c *Container) AddInterceptor(interceptor Interceptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interceptors = append(c.interceptors, interceptor)
}

// ServiceOption is a function that configures a service definition
type ServiceOption func(*ServiceDefinition)

// WithSingleton makes the service a singleton
func WithSingleton() ServiceOption {
	return func(def *ServiceDefinition) {
		def.Singleton = true
	}
}

// WithDependencies sets the service dependencies
func WithDependencies(deps ...string) ServiceOption {
	return func(def *ServiceDefinition) {
		def.Dependencies = deps
	}
}

// WithTags sets the service tags
func WithTags(tags ...string) ServiceOption {
	return func(def *ServiceDefinition) {
		def.Tags = tags
	}
}

// WithLifecycle sets the service lifecycle hooks
func WithLifecycle(lifecycle ServiceLifecycle) ServiceOption {
	return func(def *ServiceDefinition) {
		def.Lifecycle = lifecycle
	}
}

// WithConfig sets the service configuration
func WithConfig(config map[string]interface{}) ServiceOption {
	return func(def *ServiceDefinition) {
		def.Config = config
	}
}

// WithPriority sets the service priority (higher numbers start first)
func WithPriority(priority int) ServiceOption {
	return func(def *ServiceDefinition) {
		def.Priority = priority
	}
}

// WithTimeout sets the service creation timeout
func WithTimeout(timeout time.Duration) ServiceOption {
	return func(def *ServiceDefinition) {
		def.Timeout = timeout
	}
}

// Helper methods

// validateFactory validates that the factory function is valid
func (c *Container) validateFactory(factory interface{}) error {
	factoryType := reflect.TypeOf(factory)

	if factoryType.Kind() != reflect.Func {
		return fmt.Errorf("factory must be a function")
	}

	// Factory should return at least one value
	if factoryType.NumOut() == 0 {
		return fmt.Errorf("factory must return at least one value")
	}

	// Last return value should be error (optional)
	if factoryType.NumOut() > 1 {
		lastOut := factoryType.Out(factoryType.NumOut() - 1)
		errorInterface := reflect.TypeOf((*error)(nil)).Elem()
		if !lastOut.Implements(errorInterface) {
			return fmt.Errorf("last return value should be error")
		}
	}

	return nil
}

// createInstance creates a new instance using the factory function
func (c *Container) createInstance(def *ServiceDefinition) (interface{}, error) {
	factoryType := reflect.TypeOf(def.Factory)
	factoryValue := reflect.ValueOf(def.Factory)

	// Prepare arguments for factory function
	args := make([]reflect.Value, factoryType.NumIn())

	for i := 0; i < factoryType.NumIn(); i++ {
		argType := factoryType.In(i)

		// Special handling for context
		if argType == reflect.TypeOf((*context.Context)(nil)).Elem() {
			args[i] = reflect.ValueOf(c.ctx)
			continue
		}

		// Special handling for container
		if argType == reflect.TypeOf((*Container)(nil)) {
			args[i] = reflect.ValueOf(c)
			continue
		}

		// Try to resolve dependency by type name
		typeName := argType.String()
		if dep, err := c.Get(typeName); err == nil {
			args[i] = reflect.ValueOf(dep)
			continue
		}

		// Try to resolve by interface
		if argType.Kind() == reflect.Interface {
			instances, err := c.GetByType(argType)
			if err == nil && len(instances) > 0 {
				args[i] = reflect.ValueOf(instances[0])
				continue
			}
		}

		return nil, fmt.Errorf("cannot resolve dependency of type %s", argType)
	}

	// Call interceptors
	for _, interceptor := range c.interceptors {
		if err := interceptor.BeforeCreate(def.Name, def); err != nil {
			return nil, fmt.Errorf("interceptor failed: %w", err)
		}
	}

	// Call factory with timeout
	ctx, cancel := context.WithTimeout(c.ctx, def.Timeout)
	defer cancel()

	resultCh := make(chan []reflect.Value, 1)
	errorCh := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorCh <- fmt.Errorf("factory panicked: %v", r)
			}
		}()

		results := factoryValue.Call(args)
		resultCh <- results
	}()

	var results []reflect.Value
	select {
	case results = <-resultCh:
	case err := <-errorCh:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("factory timeout after %v", def.Timeout)
	}

	// Check for error in results
	if len(results) > 1 {
		if errValue := results[len(results)-1]; !errValue.IsNil() {
			return nil, errValue.Interface().(error)
		}
	}

	instance := results[0].Interface()

	// Call lifecycle hooks
	if def.Lifecycle.OnCreate != nil {
		if err := def.Lifecycle.OnCreate(instance); err != nil {
			return nil, fmt.Errorf("onCreate hook failed: %w", err)
		}
	}

	// Call interceptors
	for _, interceptor := range c.interceptors {
		if err := interceptor.AfterCreate(def.Name, instance); err != nil {
			return nil, fmt.Errorf("interceptor failed: %w", err)
		}
	}

	return instance, nil
}

// calculateStartOrder calculates the order in which services should be started
func (c *Container) calculateStartOrder() ([]string, error) {
	// Topological sort with priority consideration
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	order := make([]string, 0, len(c.services))

	var visit func(string) error
	visit = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("circular dependency detected involving service '%s'", name)
		}
		if visited[name] {
			return nil
		}

		visiting[name] = true

		def := c.services[name]
		for _, dep := range def.Dependencies {
			if _, exists := c.services[dep]; !exists {
				return fmt.Errorf("dependency '%s' not found for service '%s'", dep, name)
			}
			if err := visit(dep); err != nil {
				return err
			}
		}

		visiting[name] = false
		visited[name] = true
		order = append(order, name)

		return nil
	}

	// Visit all services
	for name := range c.services {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return order, nil
}

// healthCheckRoutine runs periodic health checks on services
func (c *Container) healthCheckRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.performHealthChecks()
		case <-c.shutdownCh:
			return
		}
	}
}

// performHealthChecks performs health checks on all services
func (c *Container) performHealthChecks() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for name, def := range c.services {
		if def.Lifecycle.HealthCheck == nil {
			continue
		}

		var instance interface{}
		if def.Singleton {
			if inst, exists := c.singletons[name]; exists {
				instance = inst
			}
		}

		if instance != nil {
			if err := def.Lifecycle.HealthCheck(instance); err != nil {
				c.logger.Error("Health check failed", "service", name, "error", err)
			}
		}
	}
}

// registerCoreServices registers essential core services
func (c *Container) registerCoreServices() {
	// Register the container itself
	c.Register("container", func() *Container { return c }, WithSingleton())

	// Register context
	c.Register("context", func() context.Context { return c.ctx }, WithSingleton())
}
