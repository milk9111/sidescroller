package editorio

import "testing"

func TestNormalizePrefabTarget(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{input: "enemy_variant", want: "enemy_variant.yaml"},
		{input: "enemy_variant.yaml", want: "enemy_variant.yaml"},
		{input: "enemy_variant.txt", want: "enemy_variant.txt.yaml"},
		{input: "nested/name", wantErr: true},
		{input: "   ", wantErr: true},
	}
	for _, test := range tests {
		got, err := NormalizePrefabTarget(test.input)
		if test.wantErr {
			if err == nil {
				t.Fatalf("NormalizePrefabTarget(%q) expected error", test.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("NormalizePrefabTarget(%q) unexpected error: %v", test.input, err)
		}
		if got != test.want {
			t.Fatalf("NormalizePrefabTarget(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}
