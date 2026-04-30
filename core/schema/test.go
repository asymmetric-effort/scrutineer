package schema

// ExecutionMode enumerates how tests or interactions are dispatched.
type ExecutionMode string

const (
	ModeSequential ExecutionMode = "sequential"
	ModeRandom     ExecutionMode = "random"
	ModeConcurrent ExecutionMode = "concurrent"
	ModeWeighted   ExecutionMode = "weighted"
)

// validExecutionModes is the set of recognised execution modes.
var validExecutionModes = map[ExecutionMode]bool{
	ModeSequential: true,
	ModeRandom:     true,
	ModeConcurrent: true,
	ModeWeighted:   true,
}

// Execution configures how tests or interactions are dispatched within a suite.
type Execution struct {
	Mode        ExecutionMode `yaml:"mode"`
	Concurrency int           `yaml:"concurrency"`
	Duration    string        `yaml:"duration"`
	Repeat      int           `yaml:"repeat"`
	Interval    string        `yaml:"interval"`
	Fleet       *FleetConfig  `yaml:"fleet"`
}

// Interaction groups tests into a logical collection (e.g. a user journey).
type Interaction struct {
	Name   string        `yaml:"name"`
	Weight int           `yaml:"weight"`
	Mode   ExecutionMode `yaml:"mode"`
	Tests  []Test        `yaml:"tests"`
}

// FleetConfig configures distributed test execution providers.
type FleetConfig struct {
	Providers []FleetProvider `yaml:"providers"`
}

// FleetProvider configures a single fleet provider instance.
type FleetProvider struct {
	Provider string         `yaml:"provider"`
	Weight   int            `yaml:"weight"`
	TTL      int            `yaml:"ttl"`
	Config   map[string]any `yaml:"-"`
}

// TestSuite represents a complete test file.
type TestSuite struct {
	Suite        string         `yaml:"suite"`
	Tags         []string       `yaml:"tags"`
	Setup        []TestStep     `yaml:"setup"`
	Teardown     []TestStep     `yaml:"teardown"`
	Fixtures     map[string]any `yaml:"fixtures"`
	Tests        []Test         `yaml:"tests"`
	Execution    *Execution     `yaml:"execution"`
	Interactions []Interaction  `yaml:"interactions"`
}

// Test represents a single test case.
type Test struct {
	Name      string     `yaml:"name"`
	Connector string     `yaml:"connector"`
	Tags      []string   `yaml:"tags"`
	Skip      bool       `yaml:"skip"`
	Weight    int        `yaml:"weight"`
	Steps     []TestStep `yaml:"steps"`
}

// TestStep represents a single action within a test.
// Fields not explicitly tagged (method, path, body, command, etc.) are
// collected into the Parameters map via ParseTestStep.
type TestStep struct {
	Connector  string            `yaml:"connector"`
	Action     string            `yaml:"action"`
	Parameters map[string]any    `yaml:"-"`
	Assert     []map[string]any  `yaml:"assert"`
	Capture    map[string]string `yaml:"capture"`
	Timeout    string            `yaml:"timeout"`
}

// knownStepFields is the set of TestStep field names that have explicit
// struct tags and should not be placed into Parameters.
var knownStepFields = map[string]bool{
	"connector": true,
	"action":    true,
	"assert":    true,
	"capture":   true,
	"timeout":   true,
}
