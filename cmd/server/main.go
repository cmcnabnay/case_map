// Command server serves the case map on localhost.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"time"

	casemap "case-map"
	"case-map/internal/data"
)

func main() {
	addr := flag.String("addr", "localhost:3000", "listen address")
	locPath := flag.String("data", "data/locations.json", "locations file")
	flag.Parse()
	data.LoadEnv()

	static, err := fs.Sub(casemap.Public, "public")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(static))
	mux.HandleFunc("GET /api/cases", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		cases, err := data.FetchCases(ctx)
		if err != nil {
			log.Printf("fetch cases: %v", err)
			http.Error(w, "could not load cases", http.StatusBadGateway)
			return
		}
		// Re-read on every request so edits to the file show up on refresh.
		locs, err := data.LoadLocations(*locPath)
		if err != nil {
			log.Printf("load locations: %v", err)
			http.Error(w, "could not load locations", http.StatusInternalServerError)
			return
		}
		merged, _ := data.Merge(cases, locs)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(merged)
	})

	log.Printf("case map on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
