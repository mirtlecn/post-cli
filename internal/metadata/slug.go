package metadata

import (
	"strings"

	sluglib "github.com/gosimple/slug"
)

const fallbackSlugBase = "post"

func GenerateSlugFromTitle(title string) string {
	base := strings.ToLower(strings.TrimSpace(sluglib.Make(title)))
	if base == "" {
		base = fallbackSlugBase
	}

	return base
}
