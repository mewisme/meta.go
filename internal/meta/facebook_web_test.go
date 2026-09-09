package meta

import "testing"

func TestParseSearchResultsSupportsCurrentAndLegacySchemas(t *testing.T) {
	edges := []any{
		map[string]any{"relay_rendering_strategy": map[string]any{"view_model": map[string]any{"profile": map[string]any{"id": "1", "name": "Current", "url": "https://facebook.com/current"}}}},
		map[string]any{"relay_rendering_strategy": map[string]any{"result_rendering_strategies": []any{map[string]any{"view_model": map[string]any{"profile": map[string]any{"id": "2", "name": "Legacy", "url": "https://facebook.com/legacy"}}}}}},
		map[string]any{"relay_rendering_strategy": map[string]any{"view_model": map[string]any{"profile": map[string]any{"id": "1", "name": "Duplicate"}}}},
	}
	results := parseSearchResults(edges, 5)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != "1" || results[0].Name != "Current" {
		t.Fatalf("unexpected current result: %#v", results[0])
	}
	if results[1].ID != "2" || results[1].Name != "Legacy" {
		t.Fatalf("unexpected legacy result: %#v", results[1])
	}
}

func TestContainsFailedSuccess(t *testing.T) {
	if !containsFailedSuccess(map[string]any{"outer": []any{map[string]any{"success": false}}}) {
		t.Fatal("expected nested failure")
	}
	if containsFailedSuccess(map[string]any{"outer": map[string]any{"success": true}}) {
		t.Fatal("unexpected failure")
	}
}
