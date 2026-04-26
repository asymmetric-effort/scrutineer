package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/scrutineer/scrutineer/core/exitcode"
	"github.com/scrutineer/scrutineer/core/telemetry"
)

func cmdLogDump(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: scrutineer log-dump <file>")
		return exitcode.ConfigError
	}

	path := args[0]
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %s\n", path, err)
		return exitcode.ConfigError
	}
	defer f.Close()

	reader := telemetry.NewReader(f)
	defer reader.Close()

	for {
		record, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading record: %s\n", err)
			return exitcode.InternalError
		}

		printRecord(record)
	}

	return exitcode.OK
}

func printRecord(r telemetry.Record) {
	ts := time.Unix(0, r.Timestamp).UTC().Format(time.RFC3339Nano)
	fmt.Printf("[%s] %s", ts, r.EventType)

	if len(r.Tags) > 0 {
		for k, v := range r.Tags {
			fmt.Printf(" %s=%s", k, v)
		}
	}

	if len(r.Detail) > 0 {
		// Try to print as JSON if valid, otherwise raw string
		var m any
		if json.Unmarshal(r.Detail, &m) == nil {
			fmt.Printf(" %s", string(r.Detail))
		} else {
			fmt.Printf(" %s", string(r.Detail))
		}
	}

	fmt.Println()
}
