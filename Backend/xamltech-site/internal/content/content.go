// Package content loads the bilingual page content and shared UI strings from
// embedded JSON files. Each page file contains both languages keyed by "en" and
// "ar" so the two stay side by side in review.
package content

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed data
var dataFS embed.FS

// Store is an immutable, in-memory view of the content tree.
type Store struct {
	strings map[string]map[string]any            // lang -> strings
	pages   map[string]map[string]map[string]any // slug -> lang -> content
	order   []string                             // page slugs, stable order
}

// Load reads and parses every content file. It returns an error if any file is
// missing a language or is not valid JSON, so a bad edit fails the boot.
func Load() (*Store, error) {
	s := &Store{
		strings: map[string]map[string]any{},
		pages:   map[string]map[string]map[string]any{},
	}
	if err := s.loadStrings(); err != nil {
		return nil, err
	}
	if err := s.loadPages(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) loadStrings() error {
	raw, err := dataFS.ReadFile("data/strings.json")
	if err != nil {
		return err
	}
	var doc map[string]map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("strings.json: %w", err)
	}
	for _, lang := range []string{"en", "ar"} {
		if _, ok := doc[lang]; !ok {
			return fmt.Errorf("strings.json: missing language %q", lang)
		}
		s.strings[lang] = doc[lang]
	}
	return nil
}

func (s *Store) loadPages() error {
	entries, err := fs.ReadDir(dataFS, "data/pages")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".json")
		raw, err := dataFS.ReadFile("data/pages/" + e.Name())
		if err != nil {
			return err
		}
		var doc map[string]map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("page %s: %w", slug, err)
		}
		for _, lang := range []string{"en", "ar"} {
			if _, ok := doc[lang]; !ok {
				return fmt.Errorf("page %s: missing language %q", slug, lang)
			}
		}
		s.pages[slug] = doc
		s.order = append(s.order, slug)
	}
	sort.Strings(s.order)
	return nil
}

// Strings returns the shared UI strings for a language (never nil).
func (s *Store) Strings(lang string) map[string]any {
	if v, ok := s.strings[lang]; ok {
		return v
	}
	return s.strings["ar"]
}

// Page returns the content map for a slug and language. ok is false when the
// slug is unknown or the language is not present.
func (s *Store) Page(slug, lang string) (map[string]any, bool) {
	doc, ok := s.pages[slug]
	if !ok {
		return nil, false
	}
	c, ok := doc[lang]
	return c, ok
}

// Slugs returns every known page slug in stable order.
func (s *Store) Slugs() []string {
	out := make([]string, len(s.order))
	copy(out, s.order)
	return out
}
