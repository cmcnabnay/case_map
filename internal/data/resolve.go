package data

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var courseDirs = map[string]string{
	"Civil Procedure":               "civil_procedure",
	"Contracts":                     "contracts",
	"Torts":                         "torts",
	"Lawyering Skills & Strategies": "lawyering_skills_and_strategies",
}

var (
	extRe    = regexp.MustCompile(`(?i)\.(txt|pdf|docx?|rtf)$`)
	prefixRe = regexp.MustCompile(`^(cb\s+)?(pp\s+)?`)
	spaceRe  = regexp.MustCompile(`\s+`)
	nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)
)

// normalize turns "CB pp. 228–38.txt" into "228-38" so near-miss document
// names in the DB line up with filenames on disk.
func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = extRe.ReplaceAllString(s, "")
	s = strings.NewReplacer("–", "-", "—", "-", "pp.", "pp ").Replace(s)
	s = spaceRe.ReplaceAllString(s, " ")
	s = prefixRe.ReplaceAllString(strings.TrimSpace(s), "")
	return strings.TrimSpace(s)
}

// Resolution says which file holds a case and how it was found.
type Resolution struct {
	Path   string `json:"path"`   // relative to the documents root, "" if unresolved
	Method string `json:"method"` // exact | normalized | name-search | unresolved
}

// Resolver finds the source document for a case.
type Resolver struct {
	root  string
	files []string // relative paths
	text  map[string][]byte
}

func NewResolver(root string) (*Resolver, error) {
	r := &Resolver{root: root, text: map[string][]byte{}}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		r.files = append(r.files, rel)
		return nil
	})
	return r, err
}

func (r *Resolver) Resolve(c Case) Resolution {
	dir := courseDirs[c.Course]
	pick := func(match func(rel string) bool) string {
		var hits []string
		for _, f := range r.files {
			if match(f) {
				hits = append(hits, f)
			}
		}
		// Prefer the case's own course folder when several files match.
		for _, h := range hits {
			if dir != "" && strings.HasPrefix(h, dir+string(filepath.Separator)) {
				return h
			}
		}
		if len(hits) > 0 {
			return hits[0]
		}
		return ""
	}

	if p := pick(func(f string) bool { return filepath.Base(f) == c.Document }); p != "" {
		return Resolution{p, "exact"}
	}
	want := normalize(c.Document)
	if p := pick(func(f string) bool { return normalize(filepath.Base(f)) == want }); p != "" {
		return Resolution{p, "normalized"}
	}
	if p := r.searchName(c, dir); p != "" {
		return Resolution{p, "name-search"}
	}
	return Resolution{"", "unresolved"}
}

// searchName looks for the case caption inside text documents, and for the
// parties' names in filenames (e.g. "Lucy v Zehmer" -> "Lucy v. Zehmer.pdf").
func (r *Resolver) searchName(c Case, dir string) string {
	parts := strings.SplitN(strings.ToLower(c.Name), " v", 2)
	first := nonAlnum.ReplaceAllString(parts[0], "")
	if strings.HasPrefix(first, "inre") {
		first = strings.TrimPrefix(first, "inre")
	}
	if len(first) < 3 {
		return ""
	}
	for _, f := range r.files {
		if strings.Contains(nonAlnum.ReplaceAllString(strings.ToLower(filepath.Base(f)), ""), first) {
			return f
		}
	}
	needle := []byte(strings.ToLower(c.Name))
	var fallback string
	for _, f := range r.files {
		if ext := strings.ToLower(filepath.Ext(f)); ext != "" && ext != ".txt" {
			continue
		}
		if bytes.Contains(bytes.ToLower(r.read(f)), needle) {
			if dir != "" && strings.HasPrefix(f, dir+string(filepath.Separator)) {
				return f
			}
			if fallback == "" {
				fallback = f
			}
		}
	}
	return fallback
}

func (r *Resolver) read(rel string) []byte {
	if b, ok := r.text[rel]; ok {
		return b
	}
	b, _ := os.ReadFile(filepath.Join(r.root, rel))
	r.text[rel] = b
	return b
}
