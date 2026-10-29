// Command pending lists cases that are not yet in data/locations.json,
// along with the source document each one should be read from.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"case-map/internal/data"
)

func main() {
	locPath := flag.String("data", "", "locations file (default: data/locations.json in the case_map root)")
	all := flag.Bool("all", false, "list every case, not just pending ones")
	asJSON := flag.Bool("json", false, "print JSON instead of text")
	flag.Parse()
	if *locPath == "" {
		*locPath = filepath.Join(projectRoot(), "data", "locations.json")
	}

	data.LoadEnv()
	cases, err := data.FetchCases(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	locs, err := data.LoadLocations(*locPath)
	if err != nil {
		log.Fatal(err)
	}
	res, err := data.NewResolver(data.DocumentsRoot())
	if err != nil {
		log.Fatal(err)
	}
	merged, orphans := data.Merge(cases, locs)

	type row struct {
		Case     string `json:"case"`
		Course   string `json:"course"`
		Document string `json:"document"`
		data.Resolution
	}
	var rows []row
	precision := map[string]int{}
	for _, mc := range merged {
		if mc.Location != nil {
			precision[mc.Location.Precision]++
			if !*all {
				continue
			}
		}
		rows = append(rows, row{mc.Name, mc.Course, mc.Document, res.Resolve(mc.Case)})
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rows)
		return
	}
	for _, r := range rows {
		target := r.Path
		if target == "" {
			target = "UNRESOLVED (document: " + r.Document + ")"
		}
		fmt.Printf("%-55s → %s [%s]\n", r.Case, target, r.Method)
	}
	fmt.Printf("\n%d cases, %d located, %d pending\n", len(cases), len(cases)-countPending(merged), countPending(merged))
	keys := make([]string, 0, len(precision))
	for k := range precision {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-9s %d\n", k, precision[k])
	}
	for _, o := range orphans {
		fmt.Printf("warning: %q in locations file matches no case\n", o)
	}
}

func countPending(m []data.MappedCase) int {
	n := 0
	for _, c := range m {
		if c.Location == nil {
			n++
		}
	}
	return n
}

// projectRoot walks up from the working directory to the folder holding
// go.mod, so the command works from anywhere inside case_map.
func projectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}
