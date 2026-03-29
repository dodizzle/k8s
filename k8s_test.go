package main

import (
	"testing"
)

func TestBytesToString(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"empty", []byte{}, ""},
		{"ascii", []byte("hello"), "hello"},
		{"with newlines", []byte("line1\nline2\n"), "line1\nline2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BytesToString(tt.input)
			if got != tt.want {
				t.Errorf("BytesToString(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseLines(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		stripPrefix string
		want        []string
	}{
		{
			name:   "typical kubectl output",
			output: "gke-prod\ngke-staging\ngke-dev\n",
			want:   []string{"gke-prod", "gke-staging", "gke-dev"},
		},
		{
			name:   "single line",
			output: "only-one\n",
			want:   []string{"only-one"},
		},
		{
			name:   "empty output with trailing newline",
			output: "\n",
			want:   []string{""},
		},
		{
			name:   "completely empty",
			output: "",
			want:   []string{},
		},
		{
			name:        "strip namespace prefix",
			output:      "namespace/default\nnamespace/kube-system\n",
			stripPrefix: "namespace/",
			want:        []string{"default", "kube-system"},
		},
		{
			name:        "strip prefix not present",
			output:      "default\nkube-system\n",
			stripPrefix: "namespace/",
			want:        []string{"default", "kube-system"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLines(tt.output, tt.stripPrefix)
			if len(got) != len(tt.want) {
				t.Fatalf("parseLines() returned %d lines, want %d\ngot:  %v\nwant: %v", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseLines()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildNumberedMap(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{"empty", []string{}},
		{"single", []string{"alpha"}},
		{"multiple", []string{"alpha", "bravo", "charlie"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := buildNumberedMap(tt.lines)
			if len(m) != len(tt.lines) {
				t.Fatalf("buildNumberedMap() returned %d entries, want %d", len(m), len(tt.lines))
			}
			for i, line := range tt.lines {
				key := i + 1
				val, ok := m[key]
				if !ok {
					t.Errorf("missing key %d", key)
					continue
				}
				if val[0] != line {
					t.Errorf("m[%d] = %q, want %q", key, val[0], line)
				}
			}
			// verify 1-indexed (no key 0)
			if _, ok := m[0]; ok {
				t.Error("map should not have key 0 (1-indexed)")
			}
		})
	}
}

func TestValidateSelection(t *testing.T) {
	items := buildNumberedMap([]string{"alpha", "bravo", "charlie"})

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid first", "1", "alpha", false},
		{"valid middle", "2", "bravo", false},
		{"valid last", "3", "charlie", false},
		{"with whitespace", "  2  \n", "bravo", false},
		{"zero", "0", "", true},
		{"too high", "4", "", true},
		{"negative", "-1", "", true},
		{"not a number", "abc", "", true},
		{"empty input", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateSelection(tt.input, items)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSelection(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateSelection(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateSelectionEmptyMap(t *testing.T) {
	items := buildNumberedMap([]string{})
	_, err := validateSelection("1", items)
	if err == nil {
		t.Error("validateSelection on empty map should return error")
	}
}

func TestParseLinesEndToEnd(t *testing.T) {
	// Simulate the full flow: kubectl output → parseLines → buildNumberedMap → validateSelection
	kubectlOutput := "gke_project_us-west1_prod\ngke_project_us-east1_staging\n"

	lines := parseLines(kubectlOutput, "")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	items := buildNumberedMap(lines)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	got, err := validateSelection("2", items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "gke_project_us-east1_staging" {
		t.Errorf("got %q, want %q", got, "gke_project_us-east1_staging")
	}
}

func TestParseLinesNamespaceEndToEnd(t *testing.T) {
	// Simulate namespace flow with prefix stripping
	kubectlOutput := "namespace/default\nnamespace/kube-system\nnamespace/monitoring\n"

	lines := parseLines(kubectlOutput, "namespace/")
	items := buildNumberedMap(lines)

	got, err := validateSelection("3", items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "monitoring" {
		t.Errorf("got %q, want %q", got, "monitoring")
	}
}
