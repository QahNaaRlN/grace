package receiptdrafts

import (
	"reflect"
	"regexp"

	"github.com/QahNaaRlN/grace/internal/financialevents"
)

var localReceiptClock = regexp.MustCompile(`^(?:[01][0-9]|2[0-3]):[0-5][0-9](?::[0-5][0-9])?$`)

// normalizeExtractedFields projects a provider's local clock onto a date-only
// candidate. It never supplies a timezone or changes the original extraction.
// Only this top-level pointer is replaced; item values are left untouched.
func normalizeExtractedFields(f Fields) Fields {
	if f.IssuedDate != nil && financialevents.ValidCalendarDate(*f.IssuedDate) &&
		f.IssuedAt != nil && localReceiptClock.MatchString(*f.IssuedAt) {
		f.IssuedAt = nil
	}
	return f
}

// Older extracted drafts may still store the provider fields verbatim. Project
// only untouched candidates, without a SQL write, revision change or owner edit.
func projectExtractedCandidate(d *Draft) {
	if d.Status == "extracted" && len(d.Extractions) > 0 &&
		reflect.DeepEqual(d.Fields, d.Extractions[len(d.Extractions)-1].Fields) {
		d.Fields = normalizeExtractedFields(d.Fields)
	}
}
