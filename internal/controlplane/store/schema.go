package store

import _ "embed"

//go:embed migrations/001_init.sql
var schemaSQL string
