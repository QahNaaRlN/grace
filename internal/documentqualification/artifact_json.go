package documentqualification

// Bounded research artifacts contain base64 pages, HTTP captures and timestamps.
// Provider content continues to use unchanged DirectJSON/transport limits.
import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

func artifactJSON(raw []byte, target any) error {
	if len(raw) == 0 || len(raw) > ReportLimit || !utf8.Valid(raw) {
		return errors.New("artifact_json_size_or_encoding")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 32 {
			return errors.New("artifact_json_depth")
		}
		t, e := d.Token()
		if e != nil {
			return e
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return e
				}
				key, ok := k.(string)
				if !ok || seen[strings.ToLower(key)] {
					return errors.New("artifact_json_duplicate")
				}
				seen[strings.ToLower(key)] = true
				if e = walk(depth + 1); e != nil {
					return e
				}
			}
		case '[':
			for d.More() {
				if e = walk(depth + 1); e != nil {
					return e
				}
			}
		default:
			return errors.New("artifact_json_delimiter")
		}
		_, e = d.Token()
		return e
	}
	if e := walk(0); e != nil {
		return errors.New("artifact_json_invalid")
	}
	if _, e := d.Token(); e != io.EOF {
		return errors.New("artifact_json_trailing")
	}
	var shape any
	if json.Unmarshal(raw, &shape) != nil || artifactTypedKeys(shape, reflect.TypeOf(target)) != nil {
		return errors.New("artifact_json_exact_shape")
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(target); e != nil {
		return errors.New("artifact_json_shape")
	}
	return nil
}

// encoding/json accepts case aliases for Go structs. The research protocol does
// not: every declared JSON field is explicit, including false, null and zero.
func artifactTypedKeys(value any, typ reflect.Type) error {
	if typ == reflect.TypeOf(time.Time{}) {
		if _, ok := value.(string); !ok {
			return errors.New("timestamp_required")
		}
		return nil
	}
	if typ == reflect.TypeOf([]byte{}) {
		if value == nil {
			return nil
		}
		if _, ok := value.(string); !ok {
			return errors.New("base64_required")
		}
		return nil
	}
	if typ == reflect.TypeOf(json.RawMessage{}) {
		return nil
	}
	if typ.Kind() == reflect.Pointer {
		if value == nil {
			return nil
		}
		return artifactTypedKeys(value, typ.Elem())
	}
	switch typ.Kind() {
	case reflect.Struct:
		m, ok := value.(map[string]any)
		if !ok {
			return errors.New("object_required")
		}
		keys := map[string]reflect.Type{}
		optional := map[string]bool{}
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if !f.IsExported() {
				continue
			}
			tag := strings.Split(f.Tag.Get("json"), ",")[0]
			if tag == "-" {
				continue
			}
			if tag == "" {
				tag = f.Name
			}
			keys[tag] = f.Type
			optional[tag] = strings.Contains(f.Tag.Get("json"), ",omitempty")
		}
		for key := range m {
			if _, ok := keys[key]; !ok {
				return errors.New("unknown_exact_key")
			}
		}
		for key, t := range keys {
			v, ok := m[key]
			if !ok {
				if optional[key] {
					continue
				}
				return errors.New("exact_key_required")
			}
			if e := artifactTypedKeys(v, t); e != nil {
				return e
			}
		}
	case reflect.Slice, reflect.Array:
		if value == nil {
			return nil
		}
		items, ok := value.([]any)
		if !ok {
			return errors.New("array_required")
		}
		if typ.Kind() == reflect.Array && len(items) != typ.Len() {
			return errors.New("array_length")
		}
		for _, v := range items {
			if e := artifactTypedKeys(v, typ.Elem()); e != nil {
				return e
			}
		}
	case reflect.Bool:
		if _, ok := value.(bool); !ok {
			return errors.New("boolean_required")
		}
	case reflect.String:
		if _, ok := value.(string); !ok {
			return errors.New("string_required")
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, ok := value.(float64)
		if !ok || math.Trunc(n) != n {
			return errors.New("integer_required")
		}
	case reflect.Float32, reflect.Float64:
		if _, ok := value.(float64); !ok {
			return errors.New("number_required")
		}
	case reflect.Map:
		if value == nil {
			return nil
		}
		m, ok := value.(map[string]any)
		if !ok {
			return errors.New("map_required")
		}
		for _, v := range m {
			if e := artifactTypedKeys(v, typ.Elem()); e != nil {
				return e
			}
		}
	}
	return nil
}
