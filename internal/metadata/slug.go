package metadata

import (
	"fmt"
	"strings"

	sluglib "github.com/gosimple/slug"
)

const fallbackSlugBase = "post"

func GenerateSlugFromTitle(title string, unixTime int64) string {
	base := strings.ToLower(strings.TrimSpace(sluglib.Make(title)))
	if base == "" {
		base = fallbackSlugBase
	}

	return fmt.Sprintf("%s-%d", base, unixTime)
}
