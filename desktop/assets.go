//go:build !stub

package main

import "embed"

//go:embed all:frontend/dist
var assetsFS embed.FS

const assetsRoot = "frontend/dist"
