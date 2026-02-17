package security

import "testing"

func TestNormalizeTenantPath(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "normalizes separators", in: `tenants\acme\docs\a.txt`, want: "/tenants/acme/docs/a.txt"},
		{name: "rejects traversal", in: "/tenants/acme/../etc/passwd", wantErr: true},
		{name: "rejects non-tenant prefix", in: "/projects/a", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeTenantPath(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got path %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("want %q got %q", tc.want, got)
			}
		})
	}
}
