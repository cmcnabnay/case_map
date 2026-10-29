# Case Map

An interactive world map of where the facts of each case in the lawgpt `cases` table took place.

## Run

```
go build -o case-map ./cmd/server && ./case-map   # http://localhost:3000
```

The server reads cases from Supabase using `SUPABASE_READONLY_DB_URL`. It loads that value from `case_map/.env`, or falls back to `~/lawgpt/app/.env`. The database is never written to.

## Locations

`data/locations.json` maps each case name to a place, coordinates, and a precision level:

| Precision | Meaning |
|---|---|
| `address` | A specific street or address |
| `building` | A named building or site |
| `town` | A town or city |
| `county` | A county |
| `region` | A wider area, such as a lake or part of a city |
| `state` | A state |
| `country` | A whole country |
| `forum` | The opinion doesn't say where the facts happened, so the pin marks the court |

Each location was found by reading the case's source document in `~/lawgpt/documents`. Each entry includes an evidence quote and, where outside knowledge was used, a note saying so. The file is re-read on every request, so edits show up when you refresh the page.

To find cases that have no location yet, along with the document each one comes from:

```
go run ./cmd/pending          # add -json for machine output, -all to include located cases
```
