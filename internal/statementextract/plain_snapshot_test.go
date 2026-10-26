package statementextract

import (
	"encoding/json"
	"testing"
)

func TestPlainSnapshotHashIgnoresJSONBKeyOrderWithoutRounding(t *testing.T) {
	a := json.RawMessage(`{"model":"fixture/plain","selection":{"profile":"plain_document_reading","maxImages":5},"exact":9007199254740993}`)
	b := json.RawMessage(`{"exact":9007199254740993,"selection":{"maxImages":5,"profile":"plain_document_reading"},"model":"fixture/plain"}`)
	ca, e := canonicalPlainSnapshot(a)
	if e != nil {
		t.Fatal(e)
	}
	cb, e := canonicalPlainSnapshot(b)
	if e != nil {
		t.Fatal(e)
	}
	if Hash(a) == Hash(b) || Hash(ca) != Hash(cb) || string(ca) != `{"exact":9007199254740993,"model":"fixture/plain","selection":{"maxImages":5,"profile":"plain_document_reading"}}` {
		t.Fatal("snapshot identity changed across storage")
	}
	changed, _ := canonicalPlainSnapshot(json.RawMessage(`{"exact":9007199254740992,"selection":{"maxImages":5,"profile":"plain_document_reading"},"model":"fixture/plain"}`))
	if Hash(changed) == Hash(ca) {
		t.Fatal("semantic change lost")
	}
	for _, raw := range []string{`null`, `[]`, `{"x":1} {"y":2}`, `{"x":`} {
		if _, e := canonicalPlainSnapshot(json.RawMessage(raw)); e == nil {
			t.Fatal("invalid snapshot admitted")
		}
	}
}
