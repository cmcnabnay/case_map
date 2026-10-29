---
name: locate-cases
description: Find which cases in the case map still lack a location, read each case's source document in ~/lawgpt/documents to find where the facts happened, verify the place in Google Maps, and add it to data/locations.json. Use when the user says "update the case map", "locate pending cases", "what cases still need a location", or invokes /locate-cases.
---

# Locate pending cases

Goal: every case in `data/locations.json` gets a pin on the exact site where the facts happened. Work from `~/case_map`, because the tools use paths relative to it.

## 1. List what's pending

```
cd ~/case_map && go run ./cmd/pending -json
```

Each row has `case` (the exact key to use in locations.json), `course`, `document`, and `path` (the resolved source file under `~/lawgpt/documents`) with a `method`. Plain `go run ./cmd/pending` prints the same as text. If it says `0 located`, you ran it from the wrong directory.

Print the pending count, then work through cases one at a time. If the user named specific cases, do only those.

## 2. Find the source document

`path` is relative to `~/lawgpt/documents` (e.g. `torts/142-206`, `contracts/CB pp 87-93.txt`). It may be a PDF, a `.txt`, or a page range of a larger casebook file, so the file may need a glob (`ls ~/lawgpt/documents/<course>/ | grep -i <path>`).

If `path` is empty (`UNRESOLVED`), search `~/lawgpt/documents/<course>/` with Grep for a distinctive party name, using the `document` value as a hint. If you still can't find it, skip to step 4 and say so in `notes`.

If `path` is a page range in a casebook, read only that range and find the section for this case (several cases can share a range). Grep for the party names first.

## 3. Read the document for an exact location

Read the facts section of the case. An exact location is one of:

- **A street address.** Done. Use it as the `address`, and still look it up in Google Maps (above) to get the coordinates.
- **A distinct named establishment**: restaurant, store, mall, stadium, hospital, school, corporate headquarters, named apartment complex, named intersection. Go to the Google Maps lookup below; the street address you read there becomes the entry's `address`.
- **Neither.** A town, county, or state alone is not exact. Go to step 4.

Quote the sentence that supports it; it goes in `evidence`.

### Google Maps lookup (required for every entry)

Every coordinate in `locations.json` comes from Google Maps. Never type coordinates from memory, a guess, or another geocoder. This applies to every precision level: a street address, a building, a court, a town, a county, a state. First decide what the pin should be (the named site; for a procedure case with no facts, the courthouse of the deciding court, e.g. the S.D. Fla. court is the Wilkie D. Ferguson Jr. U.S. Courthouse), then search for exactly that place.

Use Claude in Chrome (invoke the `claude-in-chrome` skill first, load the tools in one ToolSearch call including `browser_batch`, and start with `tabs_context_mcp`; use one tab).

In one `browser_batch` per place:

1. `navigate` to `https://www.google.com/maps/search/<place name>+<town, state>` (a street address works too).
2. `computer wait` about 8 seconds. Reading the URL too early gives no coordinates.
3. `javascript_tool` with `[document.title, document.querySelector('button[data-item-id="address"]')?.innerText].join(' | ')` to read the place name and street address from the result panel.
4. `tabs_context_mcp` to read the tab URL. The pin's coordinates are the `!3d<lat>!4d<lng>` values in the URL's `data=` part, which are better than the `@lat,lng` viewport centre.

If the result is a list, not a place (the title is the search text and the address is empty), click the right row with `javascript_tool` (`[...document.querySelectorAll('a.hfpxzc')].find(...).click()`), wait again, and repeat steps 3 and 4. Check that the result is the right kind of place in the right town; if several branches match, use the one that fits the facts (the casebook's town, the year, a street named in the opinion) and say so in `notes`.

If Google Maps finds nothing for the specific place (e.g. a demolished building, a 19th-century station), look for a Maps marker on the site, such as a historical plaque, or else fall back to the town, county or state and say in `notes` what was not found. A coarser pin that came from Maps is better than a precise one that did not.

If Google Maps cannot be reached or shows a consent or login wall, stop and tell the user rather than switching geocoders. Never use a Google Maps API key; the project is a free stack.

## 4. Nothing usable in the document, or Maps finds nothing: research it

Work down this ladder, and only move to the next rung when the current one gives nothing usable:

1. The source document (step 3).
2. Google Maps for the place the document names. If it returns nothing, or the wrong place, don't stop.
3. **The internet, through the browser.** WebSearch and WebFetch are often blocked (403, rate limits, empty pages), so do this research in Chrome: `navigate` to a Google search for the case (`https://www.google.com/search?q=<party names>+<court+year>`), then `get_page_text` or `javascript_tool` to read results and pages. Open the full opinion itself (CourtListener, Justia, Google Scholar, Casetext, state court sites) and extract the facts with `javascript_tool` (e.g. `document.body.innerText.match(/[^.]*(street|avenue|road|located|county|highway)[^.]*\./gi)`). The lower-court opinions, briefs, news articles from the time, Wikipedia and local-history pages, corporate filings and business directories often give an address the casebook excerpt left out. Search for the specific site (the accident intersection, the plant, the store, the party's address, the courthouse). Try several queries with different party names, dates and places before giving up, and say in `notes` what you used. Then look the site up in Google Maps (below) to get the coordinates and address.
4. **Best approximation.** Only if the internet also gives nothing useful, use the most specific place you can support (a town, county, state or the court), taken from Google Maps, and say in `notes` what you searched for and didn't find.

Rules for choosing the site:

- Pin the place where the **events** happened, not the court and not the city center. The `forum` precision (pin on the court) is a last resort for cases where the facts really aren't localized; then search Google Maps for the court's own courthouse (e.g. "Wilkie D. Ferguson Jr. U.S. Courthouse"), and use its address and coordinates.
- A corporate party's headquarters is acceptable when the events had no other place (e.g. a nationwide product case), with `notes` saying the headquarters is assumed.
- Whatever came from outside the document must be called out in `notes` (and the source named), as in existing entries.

## 5. Write the entry to data/locations.json

The key is the case string from step 1, character for character (including curly quotes and the trailing citation). If it doesn't match a DB case exactly, `go run ./cmd/pending` will report it as an orphan warning and the case stays pending.

Entry shape (match existing entries):

```json
"<case name>": {
  "place_name": "Kimberly Village, Davenport, Iowa",
  "address": "4600 W 46th St, Davenport, IA 52806",
  "lat": 41.5501234,
  "lng": -90.6301234,
  "precision": "building",
  "source_file": "civil_procedure/302-06",
  "evidence": "<quoted sentence from the document>",
  "notes": "<what came from outside the document; assumptions; ambiguity>",
  "located_at": "<today, YYYY-MM-DD>"
}
```

`address` is required whenever `place_name` is a named place (a building, business, court, stadium, intersection, etc.): copy the address Google Maps shows. The map popup shows it on a line under the place name. Leave it out only for a town, county, state or country pin, where Maps shows no street address. `lat`/`lng` are the Google Maps values from the lookup, exactly as shown (the `!3d`/`!4d` numbers), never rounded or estimated. Notes must never say a coordinate is approximate or unchecked; if it isn't from Maps, don't write it.

`precision` is one of `address`, `building`, `town`, `county`, `region`, `state`, `country`, `forum` (README has the meanings). Use `address` when the pin is a street address, `building` for a named site. Omit `notes` if there is nothing to say.

Edit safely: load the JSON with a script, add keys, and write it back with `indent=2, ensure_ascii=False`, so the rest of the file doesn't change. Do not hand-edit a 100 KB file with sed. Never write to the lawgpt database.

## 6. Verify

Rerun `go run ./cmd/pending` from `~/case_map`. The pending count should drop by the number added and there must be no new `warning: ... matches no case` lines. Summarize for the user: cases added, each with place, precision, and whether the location came from the document or from research, plus any left pending and why.

## Notes

- Several cases may share one source document; each still gets its own entry and its own evidence.
- Do the work one case at a time and don't guess coordinates. If you can't verify a place, use a coarser precision and say so.
- The server re-reads the file on each request, so a browser refresh at http://localhost:3000 shows new pins.
