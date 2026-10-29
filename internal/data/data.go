// Package data loads cases from the lawgpt database and merges them with
// the hand-curated locations in data/locations.json.
package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

// Case is one row of the lawgpt cases table (only the columns the map uses).
type Case struct {
	Name        string `json:"case"`
	Course      string `json:"course"`
	Facts       string `json:"facts"`
	Holding     string `json:"holding"`
	Rule        string `json:"rule"`
	Disposition string `json:"disposition"`
	Document    string `json:"document"`
}

// Location is where the facts of a case took place.
type Location struct {
	PlaceName  string  `json:"place_name"`
	Address    string  `json:"address,omitempty"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	Precision  string  `json:"precision"` // address | building | town | county | state | country
	Evidence   string  `json:"evidence,omitempty"`
	SourceFile string  `json:"source_file,omitempty"`
	Notes      string  `json:"notes,omitempty"`
	LocatedAt  string  `json:"located_at,omitempty"`
}

// MappedCase is a case plus its location (nil if not located yet).
type MappedCase struct {
	Case
	Location *Location `json:"location"`
}

// LoadEnv reads case_map/.env, then falls back to the lawgpt app's .env.
// Variables already set in the environment win.
func LoadEnv() {
	_ = godotenv.Load(".env")
	if home, err := os.UserHomeDir(); err == nil {
		_ = godotenv.Load(filepath.Join(home, "lawgpt", "app", ".env"))
	}
}

// DocumentsRoot is where lawgpt stores source documents.
func DocumentsRoot() string {
	if v := os.Getenv("LAWGPT_DOCUMENTS"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "lawgpt", "documents")
}

// FetchCases reads every case from the database using the read-only URL.
func FetchCases(ctx context.Context) ([]Case, error) {
	url := os.Getenv("SUPABASE_READONLY_DB_URL")
	if url == "" {
		return nil, errors.New("SUPABASE_READONLY_DB_URL is not set")
	}
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `
		select "case", coalesce(course,''), coalesce(facts,''), coalesce(holding,''),
		       coalesce(rule,''), coalesce(disposition,''), coalesce(document,'')
		from cases order by course, "case"`)
	if err != nil {
		return nil, fmt.Errorf("query cases: %w", err)
	}
	var cases []Case
	for rows.Next() {
		var c Case
		if err := rows.Scan(&c.Name, &c.Course, &c.Facts, &c.Holding, &c.Rule, &c.Disposition, &c.Document); err != nil {
			return nil, err
		}
		cases = append(cases, c)
	}
	return cases, rows.Err()
}

// LoadLocations reads the locations file. A missing file is an empty map.
func LoadLocations(path string) (map[string]Location, error) {
	locs := map[string]Location{}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return locs, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &locs); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return locs, nil
}

// Merge attaches locations to cases, and also reports location keys that no
// longer match any case (e.g. a case was renamed in the DB).
func Merge(cases []Case, locs map[string]Location) (out []MappedCase, orphans []string) {
	seen := map[string]bool{}
	for _, c := range cases {
		mc := MappedCase{Case: c}
		if l, ok := locs[c.Name]; ok {
			l := l
			mc.Location = &l
			seen[c.Name] = true
		}
		out = append(out, mc)
	}
	for k := range locs {
		if !seen[k] {
			orphans = append(orphans, k)
		}
	}
	sort.Strings(orphans)
	return out, orphans
}
