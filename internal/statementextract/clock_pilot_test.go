package statementextract

import (
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"os"
	"reflect"
	"strings"
	"testing"
)

func clockPublicDocument(t *testing.T) statementdata.Document {
	t.Helper()
	b, e := os.ReadFile("testdata/s119-cards-document.json")
	if e != nil {
		t.Fatal(e)
	}
	var d statementdata.Document
	if json.Unmarshal(b, &d) != nil {
		t.Fatal("public document")
	}
	return d
}
func TestClockRequestSingleFactorAndIdentity(t *testing.T) {
	d := clockPublicDocument(t)
	if d.Hash() != ClockCardsDocumentHash {
		t.Fatalf("actual public document hash %s", d.Hash())
	}
	before := d.Hash()
	b, e := BuildClockRequest(d, "base")
	if e != nil {
		t.Fatal(e)
	}
	c, e := BuildClockRequest(d, "candidate")
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(b.Plan, c.Plan) || !reflect.DeepEqual(b.Schema, c.Schema) || b.Messages[1]["content"] != c.Messages[1]["content"] || c.Messages[0]["content"] != b.Messages[0]["content"]+"\n\n"+ClockAppendix || b.Hash == c.Hash || b.PromptSHA != ClockBasePromptSHA || c.PromptSHA != ClockCandidatePromptSHA {
		t.Fatal("extra experimental factor")
	}
	raw, _ := json.Marshal(c.Messages)
	if c.InputBytes != len(raw)+len(c.Schema) || b.InputBytes != InputBytes(b.Plan.Chunks[0]) || d.Hash() != before {
		t.Fatal("bytes/mutation")
	}
	c.Messages[1]["content"] = "MUTATED"
	c.Schema[0] = 'X'
	c.Plan.Chunks[0].Spans[0].Text = "MUTATED"
	again, e := BuildClockRequest(d, "candidate")
	if e != nil || again.Hash == "" || again.Messages[1]["content"] == "MUTATED" || d.Hash() != before {
		t.Fatal("shared state")
	}
	for _, v := range []string{"", "BASE", "v14", "mercury"} {
		if _, e := BuildClockRequest(d, v); e == nil {
			t.Fatal("variant", v)
		}
	}
	t.Logf("PUBLIC_S119_BYTES base=%d candidate=%d documentHash=%s planHash=%s baseRequest=%s candidateRequest=%s", b.InputBytes, again.InputBytes, d.Hash(), Hash(b.Plan), b.Hash, again.Hash)
}
func TestClockRequestExactBounds(t *testing.T) {
	values := make([]string, 2000)
	for i := range values {
		values[i] = "RUB"
	}
	if _, e := BuildClockRequest(testDocument(values...), "candidate"); e != nil {
		t.Fatal("2000", e)
	}
	if _, e := BuildClockRequest(testDocument(append(values, "RUB")...), "candidate"); e == nil {
		t.Fatal("2001")
	}
	// Tune real visible numeric spans without bypassing the redactor or planner.
	var d statementdata.Document
	low, high := 1, 2001
	for low < high {
		n := (low + high) / 2
		xs := make([]string, n)
		for i := range xs {
			xs[i] = "1.00"
		}
		candidate := testDocument(xs...)
		if _, e := BuildClockRequest(candidate, "candidate"); e == nil {
			low = n + 1
		} else {
			high = n
		}
	}
	xs := make([]string, low-1)
	for i := range xs {
		xs[i] = "1.00"
	}
	d = testDocument(xs...)
	q, e := BuildClockRequest(d, "candidate")
	if e != nil {
		t.Fatal(e)
	}
	padding := 90000 - q.InputBytes
	for i := 0; padding > 0; i++ {
		if i >= len(d.Spans) {
			t.Fatal("insufficient tuning spans")
		}
		n := padding
		if n > 11 {
			n = 11
		}
		d.Spans[i].Text = strings.Repeat("1", n+1) + ".00"
		padding -= n
	}
	q, e = BuildClockRequest(d, "candidate")
	if e != nil || q.InputBytes != 90000 {
		t.Fatal("90000", q.InputBytes, e)
	}
	d.Spans[len(d.Spans)-1].Text = "11.00"
	q, e = BuildClockRequest(d, "candidate")
	if e == nil || q.Hash != "" || q.Messages != nil {
		t.Fatal("90001 partial request", q.InputBytes, e)
	}
}
