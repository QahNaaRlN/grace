package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

// encoding/json accepts case-insensitive field names and null numeric values.
// This boundary requires the exact wire spelling and primitive types instead.
func exactKeys(v any, t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		if v == nil {
			return true
		}
		t = t.Elem()
	}
	if t == reflect.TypeOf(time.Time{}) {
		_, ok := v.(string)
		return ok
	}
	switch t.Kind() {
	case reflect.Struct:
		object, ok := v.(map[string]any)
		if !ok {
			return false
		}
		fields := map[string]reflect.StructField{}
		var collect func(reflect.Type)
		collect = func(typ reflect.Type) {
			for i := 0; i < typ.NumField(); i++ {
				f := typ.Field(i)
				if !f.IsExported() {
					continue
				}
				tag := strings.Split(f.Tag.Get("json"), ",")[0]
				if tag == "-" {
					continue
				}
				if f.Anonymous && tag == "" {
					collect(f.Type)
					continue
				}
				if tag == "" {
					tag = f.Name
				}
				fields[tag] = f
			}
		}
		collect(t)
		for key, value := range object {
			f, ok := fields[key]
			if !ok || !exactKeys(value, f.Type) {
				return false
			}
		}
		for key, f := range fields {
			if !strings.Contains(f.Tag.Get("json"), ",omitempty") {
				if _, ok := object[key]; !ok {
					return false
				}
			}
		}
		return true
	case reflect.Slice, reflect.Array:
		if v == nil && t.Kind() == reflect.Slice {
			return true
		}
		list, ok := v.([]any)
		if !ok {
			return false
		}
		for _, x := range list {
			if !exactKeys(x, t.Elem()) {
				return false
			}
		}
		return true
	case reflect.Map:
		if v == nil {
			return true
		}
		object, ok := v.(map[string]any)
		if !ok {
			return false
		}
		for _, x := range object {
			if !exactKeys(x, t.Elem()) {
				return false
			}
		}
		return true
	case reflect.String:
		_, ok := v.(string)
		return ok
	case reflect.Bool:
		_, ok := v.(bool)
		return ok
	case reflect.Int, reflect.Int64, reflect.Float64:
		_, ok := v.(json.Number)
		return ok
	}
	return false
}
