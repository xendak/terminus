// Package web embeds the templates and static assets the StopTime
// server serves (docs/spec/architecture.md: assets are local, shipped
// from the binary — no external URLs anywhere).
package web

import "embed"

//go:embed templates static
var FS embed.FS
