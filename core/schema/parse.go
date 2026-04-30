package schema

import (
	"fmt"

	"github.com/scrutineer/scrutineer/core/yaml"
)

// ParseTestStep extracts known fields from a raw map and places the
// remaining entries into the Parameters map.
func ParseTestStep(raw map[string]any) TestStep {
	s := TestStep{
		Parameters: make(map[string]any),
	}

	if v, ok := raw["connector"]; ok {
		if sv, ok := v.(string); ok {
			s.Connector = sv
		}
	}

	if v, ok := raw["action"]; ok {
		if sv, ok := v.(string); ok {
			s.Action = sv
		}
	}

	if v, ok := raw["timeout"]; ok {
		if sv, ok := v.(string); ok {
			s.Timeout = sv
		}
	}

	if v, ok := raw["assert"]; ok {
		if sl, ok := v.([]any); ok {
			for _, item := range sl {
				if m, ok := item.(map[string]any); ok {
					s.Assert = append(s.Assert, m)
				}
			}
		}
	}

	if v, ok := raw["capture"]; ok {
		if m, ok := v.(map[string]any); ok {
			s.Capture = make(map[string]string)
			for k, val := range m {
				if sv, ok := val.(string); ok {
					s.Capture[k] = sv
				}
			}
		}
	}

	// Everything else goes into Parameters.
	for k, v := range raw {
		if !knownStepFields[k] {
			s.Parameters[k] = v
		}
	}

	return s
}

// ParseSuite parses YAML data into a TestSuite, using ParseTestStep for
// each step, and validates the result.
func ParseSuite(data []byte) (*TestSuite, error) {
	// First unmarshal into a raw structure so we can handle TestStep parameters.
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("schema: failed to parse suite YAML: %w", err)
	}

	// Also unmarshal into the typed struct for the simple fields.
	var suite TestSuite
	if err := yaml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("schema: failed to parse suite YAML: %w", err)
	}

	// Re-parse steps from raw data to capture Parameters.
	suite.Setup = parseStepsFromRaw(raw, "setup")
	suite.Teardown = parseStepsFromRaw(raw, "teardown")

	if testsRaw, ok := raw["tests"]; ok {
		if testsList, ok := testsRaw.([]any); ok {
			for i, t := range testsList {
				if tm, ok := t.(map[string]any); ok {
					if i < len(suite.Tests) {
						suite.Tests[i].Steps = parseStepsFromRaw(tm, "steps")
					}
				}
			}
		}
	}

	// Parse interactions if present.
	if interRaw, ok := raw["interactions"]; ok {
		suite.Interactions = parseInteractions(interRaw)
	}

	// Parse fleet provider configs if present.
	if suite.Execution != nil && suite.Execution.Fleet != nil {
		parseFleetProviderConfigs(raw, suite.Execution.Fleet)
	}

	if err := ValidateSuite(&suite); err != nil {
		return nil, err
	}

	return &suite, nil
}

// parseInteractions parses the interactions key from raw YAML, extracting
// test step parameters within each interaction's tests.
func parseInteractions(raw any) []Interaction {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}

	interactions := make([]Interaction, 0, len(list))
	for _, item := range list {
		im, ok := item.(map[string]any)
		if !ok {
			continue
		}

		inter := Interaction{}
		if v, ok := im["name"].(string); ok {
			inter.Name = v
		}
		if v, ok := im["weight"].(int); ok {
			inter.Weight = v
		}
		if v, ok := im["mode"].(string); ok {
			inter.Mode = ExecutionMode(v)
		}

		// Parse tests within the interaction.
		if testsRaw, ok := im["tests"].([]any); ok {
			for _, t := range testsRaw {
				tm, ok := t.(map[string]any)
				if !ok {
					continue
				}

				test := Test{}
				if v, ok := tm["name"].(string); ok {
					test.Name = v
				}
				if v, ok := tm["connector"].(string); ok {
					test.Connector = v
				}
				if v, ok := tm["skip"].(bool); ok {
					test.Skip = v
				}
				if v, ok := tm["weight"].(int); ok {
					test.Weight = v
				}
				if v, ok := tm["tags"].([]any); ok {
					for _, tag := range v {
						if s, ok := tag.(string); ok {
							test.Tags = append(test.Tags, s)
						}
					}
				}
				test.Steps = parseStepsFromRaw(tm, "steps")
				inter.Tests = append(inter.Tests, test)
			}
		}

		interactions = append(interactions, inter)
	}

	return interactions
}

// parseFleetProviderConfigs extracts provider-specific configuration from
// raw YAML into FleetProvider.Config fields. For each provider entry, the
// sub-map keyed by the provider name (e.g., "static", "aws_ec2") is
// extracted into Config.
func parseFleetProviderConfigs(raw map[string]any, fleet *FleetConfig) {
	execRaw, ok := raw["execution"]
	if !ok {
		return
	}
	execMap, ok := execRaw.(map[string]any)
	if !ok {
		return
	}
	fleetRaw, ok := execMap["fleet"]
	if !ok {
		return
	}
	fleetMap, ok := fleetRaw.(map[string]any)
	if !ok {
		return
	}
	providersRaw, ok := fleetMap["providers"]
	if !ok {
		return
	}
	providersList, ok := providersRaw.([]any)
	if !ok {
		return
	}

	for i, p := range providersList {
		if i >= len(fleet.Providers) {
			break
		}
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		provName := fleet.Providers[i].Provider
		if provName == "" {
			continue
		}
		if cfg, ok := pm[provName].(map[string]any); ok {
			fleet.Providers[i].Config = cfg
		}
	}
}

// parseStepsFromRaw extracts a slice of TestStep from a raw map at the given key.
func parseStepsFromRaw(raw map[string]any, key string) []TestStep {
	v, ok := raw[key]
	if !ok {
		return nil
	}
	sl, ok := v.([]any)
	if !ok {
		return nil
	}
	steps := make([]TestStep, 0, len(sl))
	for _, item := range sl {
		if m, ok := item.(map[string]any); ok {
			steps = append(steps, ParseTestStep(m))
		}
	}
	return steps
}

// ParseConfig parses YAML data into a Config and validates the result.
func ParseConfig(data []byte) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("schema: failed to parse config YAML: %w", err)
	}

	if err := ValidateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
