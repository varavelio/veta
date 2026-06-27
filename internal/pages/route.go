package pages

import (
	"fmt"

	"github.com/varavelio/veta/internal/permalink"
)

// normalizePermalink converts a user permalink into a canonical permalink and
// relative output file path.
func normalizePermalink(rawPermalink string) (string, string, error) {
	normalizedPermalink, outputPath, err := permalink.Normalize(rawPermalink)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", ErrPermalinkInvalid, err)
	}

	return normalizedPermalink, outputPath, nil
}
