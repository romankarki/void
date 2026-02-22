package installer

import "testing"

func TestReleaseAssetName(t *testing.T) {
	cases := []struct {
		goos   string
		goarch string
		want   string
		ok     bool
	}{
		{goos: "windows", goarch: "amd64", want: "void-windows-amd64.exe", ok: true},
		{goos: "windows", goarch: "arm64", want: "void-windows-arm64.exe", ok: true},
		{goos: "linux", goarch: "amd64", want: "void-linux-amd64", ok: true},
		{goos: "darwin", goarch: "arm64", want: "void-darwin-arm64", ok: true},
		{goos: "linux", goarch: "386", ok: false},
	}

	for _, tc := range cases {
		got, err := releaseAssetName(tc.goos, tc.goarch)
		if tc.ok && err != nil {
			t.Fatalf("releaseAssetName(%s, %s) unexpected error: %v", tc.goos, tc.goarch, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("releaseAssetName(%s, %s) expected error, got %q", tc.goos, tc.goarch, got)
		}
		if tc.ok && got != tc.want {
			t.Fatalf("releaseAssetName(%s, %s) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
		}
	}
}
