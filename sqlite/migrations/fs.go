package migrations

import "embed"

//go:embed *.sql
var fs embed.FS

func FS() embed.FS {
	return fs
}
