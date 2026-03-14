package tailwind

import (
	_ "embed"
)

//go:embed dist.css
var CSS []byte
