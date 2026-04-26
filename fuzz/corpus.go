package fuzz

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Corpus manages seed and discovered interesting inputs.
type Corpus struct {
	dir     string
	entries []map[string]any
}

// NewCorpus creates a new Corpus that stores entries in the given directory.
func NewCorpus(dir string) *Corpus {
	return &Corpus{
		dir: dir,
	}
}

// Load reads corpus entries from the directory. If the directory does not
// exist it is created. Corrupt (non-JSON) files are skipped.
func (c *Corpus) Load() error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return fmt.Errorf("creating corpus dir: %w", err)
	}

	dirEntries, err := os.ReadDir(c.dir)
	if err != nil {
		return fmt.Errorf("reading corpus dir: %w", err)
	}

	// Sort for deterministic order.
	sort.Slice(dirEntries, func(i, j int) bool {
		return dirEntries[i].Name() < dirEntries[j].Name()
	})

	c.entries = nil
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(c.dir, de.Name()))
		if err != nil {
			continue // skip unreadable
		}

		var entry map[string]any
		if err := json.Unmarshal(data, &entry); err != nil {
			continue // skip corrupt
		}

		c.entries = append(c.entries, entry)
	}

	return nil
}

// Add adds a new entry to the corpus and writes it to disk.
func (c *Corpus) Add(entry map[string]any) error {
	if err := c.SaveEntry(entry); err != nil {
		return err
	}
	c.entries = append(c.entries, entry)
	return nil
}

// Entries returns all corpus entries.
func (c *Corpus) Entries() []map[string]any {
	return c.entries
}

// SaveEntry writes a single entry to a JSON file in the corpus directory.
func (c *Corpus) SaveEntry(entry map[string]any) error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return fmt.Errorf("creating corpus dir: %w", err)
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling entry: %w", err)
	}

	name := fmt.Sprintf("entry_%d.json", time.Now().UnixNano())
	path := filepath.Join(c.dir, name)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing entry: %w", err)
	}

	return nil
}
