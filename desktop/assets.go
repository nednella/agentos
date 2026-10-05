package main

import "embed"

//go:embed all:stubassets
var assetsFS embed.FS

const assetsRoot = "stubassets"
