package feature

import (
	"errors"
	"fmt"
	"strings"
)

// invalidRefRunes are characters that git forbids anywhere in a ref name.
const invalidRefRunes = " ~^:?*[\\"

// ValidateBranchSlug reports whether slug is acceptable as a branch slug.
// Empty string is valid — it means "no override" and Resolve falls back to
// the feature Identifier. Non-empty slugs must not contain path separators,
// the "feature/" prefix (the system prepends it), or characters/sequences
// that make the resulting ref name invalid per git-check-ref-format(1).
func ValidateBranchSlug(slug string) error {
	if slug == "" {
		return nil
	}
	if strings.HasPrefix(slug, "feature/") {
		return errors.New(`branch_slug must not contain the "feature/" prefix — the system adds it automatically`)
	}
	if strings.Contains(slug, "/") {
		return errors.New("branch_slug must not contain path separators (/)")
	}
	if strings.Contains(slug, "..") {
		return errors.New(`branch_slug contains invalid git-ref sequence (..)`)
	}
	if strings.Contains(slug, "@{") {
		return errors.New(`branch_slug contains invalid git-ref sequence (@{)`)
	}
	if strings.HasPrefix(slug, ".") {
		return errors.New("branch_slug must not start with a dot (.)")
	}
	if strings.HasSuffix(slug, ".") {
		return errors.New("branch_slug must not end with a dot (.)")
	}
	if strings.HasSuffix(slug, ".lock") {
		return errors.New(`branch_slug must not end with ".lock"`)
	}
	for _, c := range slug {
		if c <= 0x1F || c == 0x7F || strings.ContainsRune(invalidRefRunes, c) {
			return fmt.Errorf("branch_slug contains invalid git-ref character %q", c)
		}
	}
	return nil
}
