package statementextract

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

// This decoder is confined to the two S1.21 response shapes. Legacy decoders
// remain unchanged. Exact round-trip shape rejects missing/case-aliased fields.
func rowScopedDecode(raw []byte, out any) error {
	fail := errors.New("response_invalid")
	if len(raw) > RowScopedResponseLimit || !utf8.Valid(raw) || !rowScopedUnicode(raw) {
		return fail
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 32 {
			return fail
		}
		token, e := d.Token()
		if e != nil || token == nil {
			return fail
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					key, e := d.Token()
					if e != nil {
						return fail
					}
					s, ok := key.(string)
					if !ok || seen[strings.ToLower(s)] {
						return fail
					}
					seen[strings.ToLower(s)] = true
					if walk(depth+1) != nil {
						return fail
					}
				}
			case '[':
				for d.More() {
					if walk(depth+1) != nil {
						return fail
					}
				}
			default:
				return fail
			}
			_, e = d.Token()
			if e != nil {
				return fail
			}
		}
		return nil
	}
	if walk(0) != nil {
		return fail
	}
	if _, e := d.Token(); e != io.EOF {
		return fail
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(out) != nil || dec.Decode(&struct{}{}) != io.EOF {
		return fail
	}
	canonical, e := json.Marshal(out)
	if e != nil {
		return fail
	}
	var got, want any
	if json.Unmarshal(raw, &got) != nil || json.Unmarshal(canonical, &want) != nil || !reflect.DeepEqual(got, want) {
		return fail
	}
	return nil
}
func rowScopedUnicode(raw []byte) bool {
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' {
			continue
		}
		i++
		if i >= len(raw) {
			return false
		}
		if raw[i] != 'u' {
			continue
		}
		if i+4 >= len(raw) {
			return false
		}
		n, e := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
		if e != nil {
			return false
		}
		i += 4
		if n >= 0xdc00 && n <= 0xdfff {
			return false
		}
		if n >= 0xd800 && n <= 0xdbff {
			if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
				return false
			}
			low, e := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
			if e != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 6
		}
	}
	return true
}
