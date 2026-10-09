// Package agentos holds the files at the root of the repository that the app ships with.
package agentos

import _ "embed"

// Changelog is the release notes release-please writes, newest release first.
//
//go:embed CHANGELOG.md
var Changelog string
