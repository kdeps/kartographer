package search

import "testing"

var testFilter = NewFilter(FilterOptions{
	SkipHidden: true, SkipDirs: []string{"node_modules", "vendor"}, Extensions: []string{".go", ".md"},
	SecretNames: []string{"id_rsa", ".env"}, SecretPrefixes: []string{".env."}, SecretExts: []string{".pem"},
	SecretDirs: []string{".ssh"},
})

func TestFilter(t *testing.T) {
	for name, want := range map[string]bool{"": true, ".git": true, "node_modules": true, "vendor": true, "Vendor": false, "src": false} {
		if got := testFilter.SkipEntry(name); got != want {
			t.Errorf("SkipEntry(%q) = %v", name, got)
		}
	}
	for name, want := range map[string]bool{
		"main.go": true, "README": true, "notes.md": true, "notes.MD": false, "photo.png": false, "cert.pem": false,
	} {
		if got := testFilter.Indexable(name); got != want {
			t.Errorf("Indexable(%q) = %v", name, got)
		}
	}
	for path, want := range map[string]bool{
		"/h/.ssh/config": true, "/p/id_rsa": true, "/p/.env": true, "/p/.env.local": true, "/p/KEY.PEM": true,
		"/p/env.go": false, "/p/main.go": false,
	} {
		if got := testFilter.Secret(path); got != want {
			t.Errorf("Secret(%q) = %v", path, got)
		}
	}
}
