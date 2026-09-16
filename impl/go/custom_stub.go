package twill

// Placeholders for the compile pipeline chapter.

// CustomVariant is a parsed @custom-variant (implemented in a later chapter).
type CustomVariant struct{ Name string }

// CustomUtility is a parsed @utility (implemented in a later chapter).
type CustomUtility struct{ Name string }

// IsCustomVariantCompat is implemented in a later chapter.
func IsCustomVariantCompat(node *AtRule) bool { return false }

// ParseCustomVariant is implemented in a later chapter.
func ParseCustomVariant(node *AtRule) (*CustomVariant, error) {
	return &CustomVariant{Name: node.Params}, nil
}

// ParseUtilityDefinition is implemented in a later chapter.
func ParseUtilityDefinition(node *AtRule) (*CustomUtility, error) {
	return &CustomUtility{Name: node.Params}, nil
}
