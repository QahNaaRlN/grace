package statementextract

import (
	_ "embed"
	"encoding/json"
	"errors"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

// The existing public fixture is embedded only into the fixed operator using it.
// No owner document or new corpus entry is introduced.
//
//go:embed testdata/s119-cards-document.json
var rowPilotPublicDocument []byte

func RowPilotPublicDocument() (statementdata.Document, error) {
	var d statementdata.Document
	if json.Unmarshal(rowPilotPublicDocument, &d) != nil || d.Validate() != nil || d.Hash() != ClockCardsDocumentHash {
		return d, errors.New("row_pilot_fixture_invalid")
	}
	return d, nil
}
