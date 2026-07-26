package javaversion

import "testing"

func TestMajor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  string
		valid bool
	}{
		{name: "modern", value: "21.0.8", want: "21", valid: true},
		{name: "legacy", value: "1.8.0_452", want: "8", valid: true},
		{name: "trimmed", value: " 17-ea ", want: "17", valid: true},
		{name: "missing", value: "", valid: false},
		{name: "unsupported", value: "current", valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := Major(test.value)
			if (err == nil) != test.valid {
				t.Fatalf("Major(%q) error = %v, valid = %t", test.value, err, test.valid)
			}
			if got != test.want {
				t.Errorf("Major(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestOptionalMajorAcceptsEmptyValue(t *testing.T) {
	t.Parallel()

	major, err := OptionalMajor("  ")
	if err != nil {
		t.Fatalf("OptionalMajor() error = %v", err)
	}
	if major != "" {
		t.Errorf("OptionalMajor() = %q, want empty", major)
	}
}
