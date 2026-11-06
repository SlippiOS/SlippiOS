package host

import "testing"

func TestVersionFromImage(t *testing.T) {
	cases := map[string]string{
		"slippios_0.4.2.1793730367.raw":        "0.4.2.1793730367",
		"slippios_0.4.2.1793730367.raw.xz":     "0.4.2.1793730367",
		"slippios_0.4.2.1793730367-signed.raw": "0.4.2.1793730367",
	}
	for input, want := range cases {
		if got := versionFromImage(input); got != want {
			t.Fatalf("versionFromImage(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestHumanSize(t *testing.T) {
	if got := HumanSize(2621440000); got != "2.4 GiB" {
		t.Fatalf("HumanSize = %q", got)
	}
	if got := HumanSize(512); got != "512 B" {
		t.Fatalf("HumanSize = %q", got)
	}
}
