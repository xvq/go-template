// Package password defines shared password hashing parameters.
package password

import "github.com/alexedwards/argon2id"

// Params is the project-wide Argon2id configuration. Treat it as read-only.
// Memory is measured in KiB; 19 * 1024 is 19 MiB per calculation.
var Params = argon2id.Params{
	Memory:      19 * 1024,
	Iterations:  2,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}
