package infrastructure

import "testing"

func TestHTMLReferenceExtractor_HrefAndSrc(t *testing.T) {
	extr := NewHTMLReferenceExtractor("/root")
	content := `<a href="other.html">Other</a> <img src="img/logo.png"> <link href="style.css">`
	got := extr.Extract(content, "/root/index.html")
	want := []string{"/root/other.html", "/root/img/logo.png", "/root/style.css"}
	assertPaths(t, got, want)
}

func TestHTMLReferenceExtractor_DropsExternalURLs(t *testing.T) {
	extr := NewHTMLReferenceExtractor("/root")
	content := `<a href="https://example.com/page">ext</a> <a href="mailto:a@b.com">mail</a>`
	got := extr.Extract(content, "/root/index.html")
	if len(got) != 0 {
		t.Fatalf("expected no references, got %v", got)
	}
}

func TestHTMLReferenceExtractor_DropsOutsideRoot(t *testing.T) {
	extr := NewHTMLReferenceExtractor("/root")
	content := `<a href="../outside.html">outside</a>`
	got := extr.Extract(content, "/root/index.html")
	if len(got) != 0 {
		t.Fatalf("expected no references, got %v", got)
	}
}

func TestHTMLReferenceExtractor_Dedupes(t *testing.T) {
	extr := NewHTMLReferenceExtractor("/root")
	content := `<a href="other.html">a</a> <a href="other.html">b</a>`
	got := extr.Extract(content, "/root/index.html")
	want := []string{"/root/other.html"}
	assertPaths(t, got, want)
}
