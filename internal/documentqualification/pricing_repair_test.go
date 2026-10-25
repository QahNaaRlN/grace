package documentqualification

import (
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"testing"
	"time"
)

func TestRActualCandidateConservativeReservations(t *testing.T) {
	c := corpus(t)
	raw, err := ReadPublic("../..", "internal/documentqualification/testdata/candidate.json", 16384)
	if err != nil {
		t.Fatal(err)
	}
	member, err := DecodeCandidate(raw)
	if err != nil || member == nil {
		t.Fatal(err)
	}
	m := NewManifest(c, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", member)
	if err = m.Validate(c, m.Source, true); err != nil {
		t.Fatal(err)
	}
	if member.RequestFeeEvidence != assistant.RequestFeeRouterCeilingOnly || member.Policy.RequestFeeMicros != 1000 || member.Policy.ImageFeeMicros != 1 {
		t.Fatal("candidate not accepted R caps")
	}
	p := member.Policy
	p.VerifiedAt = time.Now()
	p.Selection = &assistant.ModelSelection{Profile: assistant.DocumentReading, MaxImages: member.MaxImages, MaxImageSide: member.MaxImageSide}
	maximum, err := p.ReservationMicros(30000, 16000)
	if err != nil || maximum != 35654 {
		t.Fatal(maximum, err)
	}
	var actual, total int64
	for _, v := range c.Cases {
		prompt, schema := assistant.ReceiptDocumentProtocol()
		if v.Golden != nil {
			prompt, schema = statementextract.DocumentPrompt, statementdata.VisualSchema()
		}
		request, err := assistant.DocumentRequest(p, v.Document, prompt, schema, 16000)
		if err != nil {
			t.Fatal(v.ID, err)
		}
		reserve, err := p.ReservationMicros(request.InputTokenUpperBound, request.MaxOutputTokens)
		if err != nil || reserve > MaximumCallMicros {
			t.Fatal(v.ID, reserve, err)
		}
		actual += reserve
		// CurrentDocumentPolicy reserves its full 30k/16k plan for statements.
		if v.Golden != nil {
			total += maximum
		} else {
			total += reserve
		}
	}
	if actual != 247349 || total != 267205 || 8*maximum != 285232 || total > MaximumMicros || 3*8*maximum > MaximumMicros {
		t.Fatal("conservative plan", actual, total, maximum)
	}
	t.Logf("actual request reserves=%d; campaign preflight=%d; eight maximal=%d", actual, total, 8*maximum)
}
