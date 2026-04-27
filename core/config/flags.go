package config

import (
	"flag"
	"fmt"
	"strings"
)

// Flags holds CLI flag values that can override configuration file settings.
type Flags struct {
	ConfigFile  string
	Parallelism int
	Timeout     string
	Format      string // reporter format override (ansi, json)
	Verbose     bool
	Tags        []string // filter tests by tags
}

// tagsValue implements flag.Value for a comma-separated list of tags.
type tagsValue struct {
	tags *[]string
}

func (t *tagsValue) String() string {
	if t.tags == nil {
		return ""
	}
	return strings.Join(*t.tags, ",")
}

func (t *tagsValue) Set(val string) error {
	parts := strings.Split(val, ",")
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			*t.tags = append(*t.tags, trimmed)
		}
	}
	return nil
}

// ParseFlags parses CLI arguments into a Flags struct.
// It uses the standard library flag package.
func ParseFlags(args []string) (*Flags, error) {
	f := &Flags{}

	fs := flag.NewFlagSet("scrutineer", flag.ContinueOnError)

	fs.StringVar(&f.ConfigFile, "config", "", "path to configuration file")
	fs.IntVar(&f.Parallelism, "parallelism", 0, "number of parallel test workers")
	fs.StringVar(&f.Timeout, "timeout", "", "test timeout duration")
	fs.StringVar(&f.Format, "format", "", "reporter format (ansi, json)")
	fs.BoolVar(&f.Verbose, "verbose", false, "enable verbose output")
	fs.Var(&tagsValue{tags: &f.Tags}, "tags", "comma-separated list of test tags")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parsing flags: %w", err)
	}

	return f, nil
}
