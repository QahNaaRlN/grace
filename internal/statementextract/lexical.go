package statementextract

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const ClassNumericFragment = "numeric_fragment"
const ClassOther = "other"

var numericFragmentShape = regexp.MustCompile(`^[+-]?(?:[0-9]+|[0-9]{1,3}(?: [0-9]{3})+)(?:[.,][0-9]{1,2})?$|^[+-]$`)

// Classify shape before redaction. This is neither a money value nor a field
// role; overflow, currency and reconciliation remain in the financial core.
func lexicalClass(text string) string {
	text = strings.NewReplacer("−", "-", "\u00a0", " ", "\u202f", " ").Replace(text)
	if utf8.RuneCountInString(text) <= 80 && numericFragmentShape.MatchString(text) {
		return ClassNumericFragment
	}
	return ClassOther
}
