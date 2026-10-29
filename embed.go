// Package casemap embeds the static map page into the server binary.
package casemap

import "embed"

//go:embed public
var Public embed.FS
