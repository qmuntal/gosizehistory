package platform

import "testing"

func TestCanonicalArch(t *testing.T) {
	for _, test := range []struct {
		arch string
		want string
	}{
		{arch: "arm", want: "arm"},
		{arch: "armv6", want: "arm"},
		{arch: "armv6l", want: "arm"},
		{arch: "arm64", want: "arm64"},
		{arch: "amd64", want: "amd64"},
	} {
		if got := CanonicalArch(test.arch); got != test.want {
			t.Errorf("CanonicalArch(%q) = %q, want %q", test.arch, got, test.want)
		}
	}
}
