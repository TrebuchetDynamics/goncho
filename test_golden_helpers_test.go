package goncho

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

type gonchoJSONDiff struct {
	Path    string
	Message string
}

func (d gonchoJSONDiff) Error() string {
	if d.Path == "" {
		return d.Message
	}
	return d.Path + ": " + d.Message
}

func marshalStableJSON(v any) ([]byte, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	return raw, nil
}

func compareGoldenJSON(wantRaw, gotRaw []byte) error {
	want, err := normalizeGoldenJSON(wantRaw)
	if err != nil {
		return fmt.Errorf("decode golden JSON: %w", err)
	}
	got, err := normalizeGoldenJSON(gotRaw)
	if err != nil {
		return fmt.Errorf("decode actual JSON: %w", err)
	}
	if reflect.DeepEqual(want, got) {
		return nil
	}
	if diff := firstJSONDiff("$", want, got); diff != nil {
		return *diff
	}
	return gonchoJSONDiff{Path: "$", Message: "JSON values differ"}
}

func normalizeGoldenJSON(raw []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("extra JSON values")
	}
	return normalizeJSONNumbers(v), nil
}

func normalizeJSONNumbers(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, v := range t {
			out[k] = normalizeJSONNumbers(v)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, v := range t {
			out[i] = normalizeJSONNumbers(v)
		}
		return out
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t.String()
	default:
		return v
	}
}

func firstJSONDiff(path string, want, got any) *gonchoJSONDiff {
	if reflect.DeepEqual(want, got) {
		return nil
	}
	switch wantTyped := want.(type) {
	case map[string]any:
		gotTyped, ok := got.(map[string]any)
		if !ok {
			return &gonchoJSONDiff{Path: path, Message: fmt.Sprintf("type mismatch: want object, got %T", got)}
		}
		keys := make([]string, 0, len(wantTyped)+len(gotTyped))
		seen := map[string]struct{}{}
		for key := range wantTyped {
			keys = append(keys, key)
			seen[key] = struct{}{}
		}
		for key := range gotTyped {
			if _, ok := seen[key]; ok {
				continue
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			nextPath := path + "." + key
			wantValue, wantOK := wantTyped[key]
			gotValue, gotOK := gotTyped[key]
			if !wantOK {
				return &gonchoJSONDiff{Path: nextPath, Message: "unexpected key"}
			}
			if !gotOK {
				return &gonchoJSONDiff{Path: nextPath, Message: "missing key"}
			}
			if diff := firstJSONDiff(nextPath, wantValue, gotValue); diff != nil {
				return diff
			}
		}
	case []any:
		gotTyped, ok := got.([]any)
		if !ok {
			return &gonchoJSONDiff{Path: path, Message: fmt.Sprintf("type mismatch: want array, got %T", got)}
		}
		if len(wantTyped) != len(gotTyped) {
			return &gonchoJSONDiff{Path: path, Message: fmt.Sprintf("array length mismatch: want %d, got %d", len(wantTyped), len(gotTyped))}
		}
		for i := range wantTyped {
			if diff := firstJSONDiff(fmt.Sprintf("%s[%d]", path, i), wantTyped[i], gotTyped[i]); diff != nil {
				return diff
			}
		}
	default:
		return &gonchoJSONDiff{Path: path, Message: fmt.Sprintf("value mismatch: want %#v, got %#v", want, got)}
	}
	return &gonchoJSONDiff{Path: path, Message: "JSON values differ"}
}
