// Package params provides a reflection-based encoder that converts option
// structs into the flat form parameters expected by the Proxmox API
//
// Tag format: `url:"name"` — the wire parameter name. Fields without a
// tag (and unexported fields) are skipped.
//
// Encoding rules:
//   - *T (pointer): encoded iff non-nil; the value is always sent, even
//     when zero — this is how "clear a field" is expressed.
//   - string: skipped when empty.
//   - bool: sent as "1" when true; skipped when false.
//   - int/float: skipped when zero.
//   - []string: comma-joined, skipped when empty.
package params

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Encode converts a struct (or pointer to struct) into Proxmox form
// parameters using the `url` struct tags.
func Encode(v any) (map[string]string, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, fmt.Errorf("params: options must not be nil")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("params: expected struct, got %s", rv.Kind())
	}

	params := map[string]string{}
	rt := rv.Type()

	for i := range rt.NumField() {
		f := rt.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}

		name, ok := splitTag(f.Tag.Get("url"))
		if !ok {
			continue
		}

		fv := rv.Field(i)

		// Pointers are encoded iff non-nil; the pointed-to value is
		// always sent so callers can express "set to zero/empty".
		isPtr := fv.Kind() == reflect.Pointer
		if isPtr {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}

		if err := encodeField(params, name, fv, isPtr); err != nil {
			return nil, fmt.Errorf("params: field %s: %w", f.Name, err)
		}
	}

	return params, nil
}

// encodeField encodes a single (already dereferenced) field value.
func encodeField(params map[string]string, name string, fv reflect.Value, isPtr bool) error {
	switch fv.Kind() { //nolint:exhaustive
	case reflect.String:
		if !isPtr && fv.String() == "" {
			return nil
		}
		params[name] = fv.String()

	case reflect.Bool:
		if !isPtr && !fv.Bool() {
			return nil
		}

		if fv.Bool() {
			params[name] = "1"
		} else {
			params[name] = "0"
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if !isPtr && fv.Int() == 0 {
			return nil
		}
		params[name] = strconv.FormatInt(fv.Int(), 10)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if !isPtr && fv.Uint() == 0 {
			return nil
		}
		params[name] = strconv.FormatUint(fv.Uint(), 10)

	case reflect.Float32, reflect.Float64:
		if !isPtr && fv.Float() == 0 {
			return nil
		}
		params[name] = strconv.FormatFloat(fv.Float(), 'f', -1, 64)

	case reflect.Slice, reflect.Array:
		if fv.Len() == 0 {
			return nil
		}
		parts := make([]string, fv.Len())
		for i := range fv.Len() {
			parts[i] = fmt.Sprintf("%v", fv.Index(i).Interface())
		}
		params[name] = strings.Join(parts, ",")

	default:
		return fmt.Errorf("unsupported type %s", fv.Kind())
	}

	return nil
}

// splitTag extracts the parameter name from a `url:"name"` tag,
// reporting whether the field should be encoded at all.
func splitTag(tag string) (string, bool) {
	if tag == "" || tag == "-" {
		return "", false
	}
	if i := strings.IndexByte(tag, ','); i >= 0 {
		tag = tag[:i]
	}
	return tag, true
}
