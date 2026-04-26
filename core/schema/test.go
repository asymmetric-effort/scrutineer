package schema

// TestSuite represents a complete test file.
type TestSuite struct {
	Suite    string         `yaml:"suite"`
	Tags     []string       `yaml:"tags"`
	Setup    []TestStep     `yaml:"setup"`
	Teardown []TestStep     `yaml:"teardown"`
	Fixtures map[string]any `yaml:"fixtures"`
	Tests    []Test         `yaml:"tests"`
}

// Test represents a single test case.
type Test struct {
	Name      string     `yaml:"name"`
	Connector string     `yaml:"connector"`
	Tags      []string   `yaml:"tags"`
	Skip      bool       `yaml:"skip"`
	Steps     []TestStep `yaml:"steps"`
}

// TestStep represents a single action within a test.
// Fields not explicitly tagged (method, path, body, command, etc.) are
// collected into the Parameters map via ParseTestStep.
type TestStep struct {
	Connector  string           `yaml:"connector"`
	Action     string           `yaml:"action"`
	Parameters map[string]any   `yaml:"-"`
	Assert     []map[string]any `yaml:"assert"`
	Capture    map[string]string `yaml:"capture"`
	Timeout    string           `yaml:"timeout"`
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
