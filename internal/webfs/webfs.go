// Package webfs embeds the static frontend so both the main binary and the
// selfcheck smoke test serve the same page without duplicating the
// //go:embed directive (the directive cannot escape its package directory).
package webfs

import (
	"embed"
	"io/fs"
)

//go:embed web
var web embed.FS

// FS returns the embedded web/ filesystem rooted at the directory itself.
func FS() fs.FS {
	sub, err := fs.Sub(web, "web")
	if err != nil {
		panic(err)
	}
	return sub
}
