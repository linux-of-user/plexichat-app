package testing

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"plexichat-client/pkg/logging"
)

// TestType represents different types of tests
type TestType string

const (
	TestTypeUnit        TestType = "unit"
	TestTypeIntegration TestType = "integration"
	TestTypeE2E         TestType = "e2e"
	TestTypePerformance TestType = "performance"
	TestTypeSecurity    TestType = "security"
	TestTypeAPI         TestType = "api"
	TestTypeUI          TestType = "ui"
	TestTypeLoad        TestType = "load"
	TestTypeStress      TestType = "stress"
	TestTypeSmoke       TestType = "smoke"
)

// TestStatus represents test execution status
type TestStatus string

const (
	StatusPending TestStatus = "pending"
	StatusRunning TestStatus = "running"
	StatusPassed  TestStatus = "passed"
	StatusFailed  TestStatus = "failed"
	StatusSkipped TestStatus = "skipped"
	StatusError   TestStatus = "error"
)

// TestResult represents the result of a test execution
type TestResult struct {
	Name        string                 `json:"name"`
	Type        TestType               `json:"type"`
	Status      TestStatus             `json:"status"`
	Duration    time.Duration          `json:"duration"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Error       string                 `json:"error,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Assertions  int                    `json:"assertions"`
	Passed      int                    `json:"passed"`
	Failed      int                    `json:"failed"`
	Metadata    map[string]interface{} `json:"metadata"`
	Logs        []string               `json:"logs"`
	Screenshots []string               `json:"screenshots,omitempty"`
}

// TestSuite represents a collection of tests
type TestSuite struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         TestType               `json:"type"`
	Tests        []*TestCase            `json:"tests"`
	SetupFunc    func() error           `json:"-"`
	TeardownFunc func() error           `json:"-"`
	BeforeEach   func() error           `json:"-"`
	AfterEach    func() error           `json:"-"`
	Timeout      time.Duration          `json:"timeout"`
	Parallel     bool                   `json:"parallel"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// TestCase represents an individual test
type TestCase struct {
	Name         string                   `json:"name"`
	Description  string                   `json:"description"`
	Type         TestType                 `json:"type"`
	TestFunc     func(*TestContext) error `json:"-"`
	SetupFunc    func() error             `json:"-"`
	TeardownFunc func() error             `json:"-"`
	Timeout      time.Duration            `json:"timeout"`
	Skip         bool                     `json:"skip"`
	SkipReason   string                   `json:"skip_reason,omitempty"`
	Tags         []string                 `json:"tags"`
	Dependencies []string                 `json:"dependencies"`
	Metadata     map[string]interface{}   `json:"metadata"`
	Result       *TestResult              `json:"result,omitempty"`
}

// TestContext provides context and utilities for test execution
type TestContext struct {
	Test       *TestCase
	Suite      *TestSuite
	Logger     *logging.Logger
	assertions int
	passed     int
	failed     int
	logs       []string
	metadata   map[string]interface{}
	mu         sync.RWMutex
}

// TestFramework manages test execution and reporting
type TestFramework struct {
	suites    []*TestSuite
	results   []*TestResult
	config    *TestConfig
	logger    *logging.Logger
	mu        sync.RWMutex
	reporters []TestReporter
	hooks     *TestHooks
}

// TestConfig represents test framework configuration
type TestConfig struct {
	Parallel        bool          `json:"parallel"`
	MaxParallel     int           `json:"max_parallel"`
	Timeout         time.Duration `json:"timeout"`
	FailFast        bool          `json:"fail_fast"`
	Verbose         bool          `json:"verbose"`
	OutputDir       string        `json:"output_dir"`
	ReportFormats   []string      `json:"report_formats"`
	Tags            []string      `json:"tags"`
	ExcludeTags     []string      `json:"exclude_tags"`
	Pattern         string        `json:"pattern"`
	Retry           int           `json:"retry"`
	RetryDelay      time.Duration `json:"retry_delay"`
	CoverageEnabled bool          `json:"coverage_enabled"`
	CoverageDir     string        `json:"coverage_dir"`
}

// TestHooks provides hooks for test lifecycle events
type TestHooks struct {
	BeforeAll   func() error
	AfterAll    func() error
	BeforeSuite func(*TestSuite) error
	AfterSuite  func(*TestSuite, []*TestResult) error
	BeforeTest  func(*TestCase) error
	AfterTest   func(*TestCase, *TestResult) error
}

// TestReporter interface for test result reporting
type TestReporter interface {
	StartSuite(*TestSuite) error
	EndSuite(*TestSuite, []*TestResult) error
	StartTest(*TestCase) error
	EndTest(*TestCase, *TestResult) error
	GenerateReport([]*TestResult) error
}

// NewTestFramework creates a new test framework instance
func NewTestFramework(config *TestConfig) *TestFramework {
	if config == nil {
		config = &TestConfig{
			Parallel:      false,
			MaxParallel:   runtime.NumCPU(),
			Timeout:       30 * time.Minute,
			FailFast:      false,
			Verbose:       false,
			OutputDir:     "test-results",
			ReportFormats: []string{"json", "junit"},
			Retry:         0,
			RetryDelay:    1 * time.Second,
		}
	}

	return &TestFramework{
		suites:    make([]*TestSuite, 0),
		results:   make([]*TestResult, 0),
		config:    config,
		logger:    logging.NewLogger(logging.INFO, nil, true),
		reporters: make([]TestReporter, 0),
		hooks:     &TestHooks{},
	}
}

// AddSuite adds a test suite to the framework
func (tf *TestFramework) AddSuite(suite *TestSuite) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	tf.suites = append(tf.suites, suite)
}

// AddReporter adds a test reporter
func (tf *TestFramework) AddReporter(reporter TestReporter) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	tf.reporters = append(tf.reporters, reporter)
}

// SetHooks sets test lifecycle hooks
func (tf *TestFramework) SetHooks(hooks *TestHooks) {
	tf.hooks = hooks
}

// RunTests executes all test suites
func (tf *TestFramework) RunTests(ctx context.Context) (*TestRunResult, error) {
	tf.logger.Info("Starting test execution...")

	startTime := time.Now()

	// Execute before all hook
	if tf.hooks.BeforeAll != nil {
		if err := tf.hooks.BeforeAll(); err != nil {
			return nil, fmt.Errorf("before all hook failed: %w", err)
		}
	}

	var allResults []*TestResult
	var totalPassed, totalFailed, totalSkipped int

	// Filter suites based on tags and patterns
	filteredSuites := tf.filterSuites()

	// Execute suites
	if tf.config.Parallel {
		allResults = tf.runSuitesParallel(ctx, filteredSuites)
	} else {
		allResults = tf.runSuitesSequential(ctx, filteredSuites)
	}

	// Calculate totals
	for _, result := range allResults {
		switch result.Status {
		case StatusPassed:
			totalPassed++
		case StatusFailed:
			totalFailed++
		case StatusSkipped:
			totalSkipped++
		}
	}

	// Execute after all hook
	if tf.hooks.AfterAll != nil {
		if err := tf.hooks.AfterAll(); err != nil {
			tf.logger.Error("After all hook failed: %v", err)
		}
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Generate reports
	for _, reporter := range tf.reporters {
		if err := reporter.GenerateReport(allResults); err != nil {
			tf.logger.Error("Failed to generate report: %v", err)
		}
	}

	runResult := &TestRunResult{
		StartTime:  startTime,
		EndTime:    endTime,
		Duration:   duration,
		TotalTests: len(allResults),
		Passed:     totalPassed,
		Failed:     totalFailed,
		Skipped:    totalSkipped,
		Results:    allResults,
		Success:    totalFailed == 0,
	}

	tf.logger.Info("Test execution completed: %d passed, %d failed, %d skipped (%.2fs)",
		totalPassed, totalFailed, totalSkipped, duration.Seconds())

	return runResult, nil
}

// RunSuite executes a single test suite
func (tf *TestFramework) RunSuite(ctx context.Context, suite *TestSuite) ([]*TestResult, error) {
	tf.logger.Info("Running test suite: %s", suite.Name)

	// Execute before suite hook
	if tf.hooks.BeforeSuite != nil {
		if err := tf.hooks.BeforeSuite(suite); err != nil {
			return nil, fmt.Errorf("before suite hook failed: %w", err)
		}
	}

	// Notify reporters
	for _, reporter := range tf.reporters {
		if err := reporter.StartSuite(suite); err != nil {
			tf.logger.Error("Reporter start suite failed: %v", err)
		}
	}

	// Execute suite setup
	if suite.SetupFunc != nil {
		if err := suite.SetupFunc(); err != nil {
			return nil, fmt.Errorf("suite setup failed: %w", err)
		}
	}

	var results []*TestResult

	// Filter tests based on configuration
	filteredTests := tf.filterTests(suite.Tests)

	// Execute tests
	if suite.Parallel && tf.config.Parallel {
		results = tf.runTestsParallel(ctx, suite, filteredTests)
	} else {
		results = tf.runTestsSequential(ctx, suite, filteredTests)
	}

	// Execute suite teardown
	if suite.TeardownFunc != nil {
		if err := suite.TeardownFunc(); err != nil {
			tf.logger.Error("Suite teardown failed: %v", err)
		}
	}

	// Execute after suite hook
	if tf.hooks.AfterSuite != nil {
		if err := tf.hooks.AfterSuite(suite, results); err != nil {
			tf.logger.Error("After suite hook failed: %v", err)
		}
	}

	// Notify reporters
	for _, reporter := range tf.reporters {
		if err := reporter.EndSuite(suite, results); err != nil {
			tf.logger.Error("Reporter end suite failed: %v", err)
		}
	}

	return results, nil
}

// RunTest executes a single test case
func (tf *TestFramework) RunTest(ctx context.Context, suite *TestSuite, test *TestCase) *TestResult {
	result := &TestResult{
		Name:      test.Name,
		Type:      test.Type,
		Status:    StatusPending,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
		Logs:      make([]string, 0),
	}

	// Check if test should be skipped
	if test.Skip {
		result.Status = StatusSkipped
		result.Message = test.SkipReason
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// Execute before test hook
	if tf.hooks.BeforeTest != nil {
		if err := tf.hooks.BeforeTest(test); err != nil {
			result.Status = StatusError
			result.Error = fmt.Sprintf("before test hook failed: %v", err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}
	}

	// Notify reporters
	for _, reporter := range tf.reporters {
		if err := reporter.StartTest(test); err != nil {
			tf.logger.Error("Reporter start test failed: %v", err)
		}
	}

	result.Status = StatusRunning

	// Create test context
	testCtx := &TestContext{
		Test:     test,
		Suite:    suite,
		Logger:   tf.logger,
		metadata: make(map[string]interface{}),
		logs:     make([]string, 0),
	}

	// Set timeout
	timeout := test.Timeout
	if timeout == 0 {
		timeout = tf.config.Timeout
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute test with retry logic
	var lastErr error
	maxRetries := tf.config.Retry + 1

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			tf.logger.Info("Retrying test %s (attempt %d/%d)", test.Name, attempt+1, maxRetries)
			time.Sleep(tf.config.RetryDelay)
		}

		// Execute before each hook
		if suite.BeforeEach != nil {
			if err := suite.BeforeEach(); err != nil {
				lastErr = fmt.Errorf("before each hook failed: %w", err)
				continue
			}
		}

		// Execute test setup
		if test.SetupFunc != nil {
			if err := test.SetupFunc(); err != nil {
				lastErr = fmt.Errorf("test setup failed: %w", err)
				continue
			}
		}

		// Execute test function
		if test.TestFunc != nil {
			testCtx := &TestContext{
				Test:   test,
				Suite:  suite,
				Logger: tf.logger,
			}
			lastErr = test.TestFunc(testCtx)
		}

		// Execute test teardown
		if test.TeardownFunc != nil {
			if teardownErr := test.TeardownFunc(); teardownErr != nil {
				tf.logger.Error("Test teardown failed: %v", teardownErr)
			}
		}

		// Execute after each hook
		if suite.AfterEach != nil {
			if afterErr := suite.AfterEach(); afterErr != nil {
				tf.logger.Error("After each hook failed: %v", afterErr)
			}
		}

		// Check if test passed
		if lastErr == nil {
			result.Status = StatusPassed
			break
		}

		// Check for timeout
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				lastErr = fmt.Errorf("test timed out after %v", timeout)
			}
		default:
		}
	}

	// Set final result
	if lastErr != nil {
		result.Status = StatusFailed
		result.Error = lastErr.Error()
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Assertions = testCtx.assertions
	result.Passed = testCtx.passed
	result.Failed = testCtx.failed
	result.Logs = testCtx.logs

	// Copy metadata
	testCtx.mu.RLock()
	for k, v := range testCtx.metadata {
		result.Metadata[k] = v
	}
	testCtx.mu.RUnlock()

	// Execute after test hook
	if tf.hooks.AfterTest != nil {
		if err := tf.hooks.AfterTest(test, result); err != nil {
			tf.logger.Error("After test hook failed: %v", err)
		}
	}

	// Notify reporters
	for _, reporter := range tf.reporters {
		if err := reporter.EndTest(test, result); err != nil {
			tf.logger.Error("Reporter end test failed: %v", err)
		}
	}

	test.Result = result
	return result
}

// filterSuites filters test suites based on configuration
func (tf *TestFramework) filterSuites() []*TestSuite {
	var filtered []*TestSuite

	for _, suite := range tf.suites {
		// Check tags
		if len(tf.config.Tags) > 0 {
			hasTag := false
			for _, tag := range tf.config.Tags {
				for _, suiteTag := range suite.Tags {
					if tag == suiteTag {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		// Check exclude tags
		if len(tf.config.ExcludeTags) > 0 {
			hasExcludeTag := false
			for _, excludeTag := range tf.config.ExcludeTags {
				for _, suiteTag := range suite.Tags {
					if excludeTag == suiteTag {
						hasExcludeTag = true
						break
					}
				}
				if hasExcludeTag {
					break
				}
			}
			if hasExcludeTag {
				continue
			}
		}

		// Check pattern
		if tf.config.Pattern != "" {
			if !strings.Contains(strings.ToLower(suite.Name), strings.ToLower(tf.config.Pattern)) {
				continue
			}
		}

		filtered = append(filtered, suite)
	}

	return filtered
}

// filterTests filters test cases based on configuration
func (tf *TestFramework) filterTests(tests []*TestCase) []*TestCase {
	var filtered []*TestCase

	for _, test := range tests {
		// Check tags
		if len(tf.config.Tags) > 0 {
			hasTag := false
			for _, tag := range tf.config.Tags {
				for _, testTag := range test.Tags {
					if tag == testTag {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		// Check exclude tags
		if len(tf.config.ExcludeTags) > 0 {
			hasExcludeTag := false
			for _, excludeTag := range tf.config.ExcludeTags {
				for _, testTag := range test.Tags {
					if excludeTag == testTag {
						hasExcludeTag = true
						break
					}
				}
				if hasExcludeTag {
					break
				}
			}
			if hasExcludeTag {
				continue
			}
		}

		// Check pattern
		if tf.config.Pattern != "" {
			if !strings.Contains(strings.ToLower(test.Name), strings.ToLower(tf.config.Pattern)) {
				continue
			}
		}

		filtered = append(filtered, test)
	}

	return filtered
}

// runSuitesSequential runs test suites sequentially
func (tf *TestFramework) runSuitesSequential(ctx context.Context, suites []*TestSuite) []*TestResult {
	var allResults []*TestResult

	for _, suite := range suites {
		results, err := tf.RunSuite(ctx, suite)
		if err != nil {
			tf.logger.Error("Suite %s failed: %v", suite.Name, err)
			continue
		}

		allResults = append(allResults, results...)

		// Check fail fast
		if tf.config.FailFast {
			for _, result := range results {
				if result.Status == StatusFailed {
					tf.logger.Info("Stopping execution due to fail fast mode")
					return allResults
				}
			}
		}
	}

	return allResults
}

// runSuitesParallel runs test suites in parallel
func (tf *TestFramework) runSuitesParallel(ctx context.Context, suites []*TestSuite) []*TestResult {
	var allResults []*TestResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, tf.config.MaxParallel)

	for _, suite := range suites {
		wg.Add(1)
		go func(s *TestSuite) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			results, err := tf.RunSuite(ctx, s)
			if err != nil {
				tf.logger.Error("Suite %s failed: %v", s.Name, err)
				return
			}

			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
		}(suite)
	}

	wg.Wait()
	return allResults
}

// runTestsSequential runs tests sequentially
func (tf *TestFramework) runTestsSequential(ctx context.Context, suite *TestSuite, tests []*TestCase) []*TestResult {
	var results []*TestResult

	for _, test := range tests {
		result := tf.RunTest(ctx, suite, test)
		results = append(results, result)

		// Check fail fast
		if tf.config.FailFast && result.Status == StatusFailed {
			tf.logger.Info("Stopping suite execution due to fail fast mode")
			break
		}
	}

	return results
}

// runTestsParallel runs tests in parallel
func (tf *TestFramework) runTestsParallel(ctx context.Context, suite *TestSuite, tests []*TestCase) []*TestResult {
	var results []*TestResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, tf.config.MaxParallel)

	for _, test := range tests {
		wg.Add(1)
		go func(t *TestCase) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := tf.RunTest(ctx, suite, t)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(test)
	}

	wg.Wait()
	return results
}

// TestRunResult represents the overall result of a test run
type TestRunResult struct {
	StartTime  time.Time     `json:"start_time"`
	EndTime    time.Time     `json:"end_time"`
	Duration   time.Duration `json:"duration"`
	TotalTests int           `json:"total_tests"`
	Passed     int           `json:"passed"`
	Failed     int           `json:"failed"`
	Skipped    int           `json:"skipped"`
	Results    []*TestResult `json:"results"`
	Success    bool          `json:"success"`
}

// Assert provides assertion methods for test context
func (tc *TestContext) Assert() *Assertions {
	return &Assertions{ctx: tc}
}

// Log adds a log message to the test context
func (tc *TestContext) Log(format string, args ...interface{}) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	message := fmt.Sprintf(format, args...)
	tc.logs = append(tc.logs, message)
	tc.Logger.Debug("[TEST] %s", message)
}

// SetMetadata sets metadata for the test
func (tc *TestContext) SetMetadata(key string, value interface{}) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.metadata[key] = value
}

// GetMetadata gets metadata from the test
func (tc *TestContext) GetMetadata(key string) (interface{}, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	value, exists := tc.metadata[key]
	return value, exists
}

// Assertions provides assertion methods
type Assertions struct {
	ctx *TestContext
}

// Equal asserts that two values are equal
func (a *Assertions) Equal(expected, actual interface{}, message ...string) {
	a.ctx.assertions++

	if reflect.DeepEqual(expected, actual) {
		a.ctx.passed++
	} else {
		a.ctx.failed++
		msg := fmt.Sprintf("Expected %v, got %v", expected, actual)
		if len(message) > 0 {
			msg = message[0] + ": " + msg
		}
		a.ctx.Log("ASSERTION FAILED: %s", msg)
		panic(fmt.Errorf("assertion failed: %s", msg))
	}
}

// NotEqual asserts that two values are not equal
func (a *Assertions) NotEqual(expected, actual interface{}, message ...string) {
	a.ctx.assertions++

	if !reflect.DeepEqual(expected, actual) {
		a.ctx.passed++
	} else {
		a.ctx.failed++
		msg := fmt.Sprintf("Expected %v to not equal %v", expected, actual)
		if len(message) > 0 {
			msg = message[0] + ": " + msg
		}
		a.ctx.Log("ASSERTION FAILED: %s", msg)
		panic(fmt.Errorf("assertion failed: %s", msg))
	}
}

// True asserts that a value is true
func (a *Assertions) True(value bool, message ...string) {
	a.ctx.assertions++

	if value {
		a.ctx.passed++
	} else {
		a.ctx.failed++
		msg := "Expected true, got false"
		if len(message) > 0 {
			msg = message[0] + ": " + msg
		}
		a.ctx.Log("ASSERTION FAILED: %s", msg)
		panic(fmt.Errorf("assertion failed: %s", msg))
	}
}

// False asserts that a value is false
func (a *Assertions) False(value bool, message ...string) {
	a.ctx.assertions++

	if !value {
		a.ctx.passed++
	} else {
		a.ctx.failed++
		msg := "Expected false, got true"
		if len(message) > 0 {
			msg = message[0] + ": " + msg
		}
		a.ctx.Log("ASSERTION FAILED: %s", msg)
		panic(fmt.Errorf("assertion failed: %s", msg))
	}
}

// Nil asserts that a value is nil
func (a *Assertions) Nil(value interface{}, message ...string) {
	a.ctx.assertions++

	if value == nil || (reflect.ValueOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil()) {
		a.ctx.passed++
	} else {
		a.ctx.failed++
		msg := fmt.Sprintf("Expected nil, got %v", value)
		if len(message) > 0 {
			msg = message[0] + ": " + msg
		}
		a.ctx.Log("ASSERTION FAILED: %s", msg)
		panic(fmt.Errorf("assertion failed: %s", msg))
	}
}

// NotNil asserts that a value is not nil
func (a *Assertions) NotNil(value interface{}, message ...string) {
	a.ctx.assertions++

	if value != nil && !(reflect.ValueOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil()) {
		a.ctx.passed++
	} else {
		a.ctx.failed++
		msg := "Expected non-nil value, got nil"
		if len(message) > 0 {
			msg = message[0] + ": " + msg
		}
		a.ctx.Log("ASSERTION FAILED: %s", msg)
		panic(fmt.Errorf("assertion failed: %s", msg))
	}
}

// Contains asserts that a string contains a substring
func (a *Assertions) Contains(haystack, needle string, message ...string) {
	a.ctx.assertions++

	if strings.Contains(haystack, needle) {
		a.ctx.passed++
	} else {
		a.ctx.failed++
		msg := fmt.Sprintf("Expected '%s' to contain '%s'", haystack, needle)
		if len(message) > 0 {
			msg = message[0] + ": " + msg
		}
		a.ctx.Log("ASSERTION FAILED: %s", msg)
		panic(fmt.Errorf("assertion failed: %s", msg))
	}
}

// NoError asserts that an error is nil
func (a *Assertions) NoError(err error, message ...string) {
	a.ctx.assertions++

	if err == nil {
		a.ctx.passed++
	} else {
		a.ctx.failed++
		msg := fmt.Sprintf("Expected no error, got: %v", err)
		if len(message) > 0 {
			msg = message[0] + ": " + msg
		}
		a.ctx.Log("ASSERTION FAILED: %s", msg)
		panic(fmt.Errorf("assertion failed: %s", msg))
	}
}

// Error asserts that an error is not nil
func (a *Assertions) Error(err error, message ...string) {
	a.ctx.assertions++

	if err != nil {
		a.ctx.passed++
	} else {
		a.ctx.failed++
		msg := "Expected an error, got nil"
		if len(message) > 0 {
			msg = message[0] + ": " + msg
		}
		a.ctx.Log("ASSERTION FAILED: %s", msg)
		panic(fmt.Errorf("assertion failed: %s", msg))
	}
}
