package embed

import "embed"

//go:embed migrations/*
var Migrations embed.FS
