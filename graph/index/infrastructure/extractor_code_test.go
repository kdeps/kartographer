package infrastructure

import "testing"

func newCodeExtractor(root string, knownFiles []string) *SourceCodeReferenceExtractor {
	e := &SourceCodeReferenceExtractor{Root: root}
	e.SetKnownFiles(knownFiles)
	return e
}

// --- Go ---

func TestSourceCodeReferenceExtractor_Go_NonRelative(t *testing.T) {
	files := []string{"/root/pkgname/file.go", "/root/main.go"}
	extr := newCodeExtractor("/root", files)
	content := `import "github.com/x/y/pkgname"`
	got := extr.Extract(content, "/root/main.go")
	assertPaths(t, got, []string{"/root/pkgname"})
}

func TestSourceCodeReferenceExtractor_Go_ExternalDropped(t *testing.T) {
	extr := newCodeExtractor("/root", []string{"/root/main.go"})
	content := `import "fmt"`
	got := extr.Extract(content, "/root/main.go")
	if len(got) != 0 {
		t.Fatalf("expected no references for stdlib import, got %v", got)
	}
}

func TestSourceCodeReferenceExtractor_Go_AmbiguousDropped(t *testing.T) {
	files := []string{"/root/a/pkgname/f.go", "/root/b/pkgname/f.go", "/root/main.go"}
	extr := newCodeExtractor("/root", files)
	content := `import "github.com/x/pkgname"`
	got := extr.Extract(content, "/root/main.go")
	if len(got) != 0 {
		t.Fatalf("expected no reference for ambiguous package name, got %v", got)
	}
}

// --- Python ---

func TestSourceCodeReferenceExtractor_Python_Relative(t *testing.T) {
	files := []string{"/root/pkg/sibling.py", "/root/pkg/main.py"}
	extr := newCodeExtractor("/root", files)
	content := "from .sibling import thing"
	got := extr.Extract(content, "/root/pkg/main.py")
	assertPaths(t, got, []string{"/root/pkg/sibling.py"})
}

func TestSourceCodeReferenceExtractor_Python_RelativeParent(t *testing.T) {
	files := []string{"/root/other/sibling.py", "/root/pkg/main.py"}
	extr := newCodeExtractor("/root", files)
	content := "from ..other.sibling import thing"
	got := extr.Extract(content, "/root/pkg/main.py")
	assertPaths(t, got, []string{"/root/other/sibling.py"})
}

func TestSourceCodeReferenceExtractor_Python_NonRelativeDottedFile(t *testing.T) {
	files := []string{"/root/foo/bar.py", "/root/main.py"}
	extr := newCodeExtractor("/root", files)
	content := "import foo.bar"
	got := extr.Extract(content, "/root/main.py")
	assertPaths(t, got, []string{"/root/foo/bar.py"})
}

func TestSourceCodeReferenceExtractor_Python_NonRelativePackageInit(t *testing.T) {
	files := []string{"/root/foo/bar/__init__.py", "/root/main.py"}
	extr := newCodeExtractor("/root", files)
	content := "from foo.bar import thing"
	got := extr.Extract(content, "/root/main.py")
	assertPaths(t, got, []string{"/root/foo/bar/__init__.py"})
}

func TestSourceCodeReferenceExtractor_Python_ExternalDropped(t *testing.T) {
	extr := newCodeExtractor("/root", []string{"/root/main.py"})
	content := "import os\nimport requests"
	got := extr.Extract(content, "/root/main.py")
	if len(got) != 0 {
		t.Fatalf("expected no references for external imports, got %v", got)
	}
}

// --- Rust ---

func TestSourceCodeReferenceExtractor_Rust_ModRelative(t *testing.T) {
	files := []string{"/root/src/util.rs", "/root/src/main.rs"}
	extr := newCodeExtractor("/root", files)
	content := "mod util;"
	got := extr.Extract(content, "/root/src/main.rs")
	assertPaths(t, got, []string{"/root/src/util.rs"})
}

func TestSourceCodeReferenceExtractor_Rust_ModDirRelative(t *testing.T) {
	files := []string{"/root/src/util/mod.rs", "/root/src/main.rs"}
	extr := newCodeExtractor("/root", files)
	content := "mod util;"
	got := extr.Extract(content, "/root/src/main.rs")
	assertPaths(t, got, []string{"/root/src/util/mod.rs"})
}

func TestSourceCodeReferenceExtractor_Rust_UseNonRelative(t *testing.T) {
	files := []string{"/root/src/foo/bar.rs", "/root/src/main.rs"}
	extr := newCodeExtractor("/root", files)
	content := "use crate::foo::bar;"
	got := extr.Extract(content, "/root/src/main.rs")
	assertPaths(t, got, []string{"/root/src/foo/bar.rs"})
}

func TestSourceCodeReferenceExtractor_Rust_ExternalDropped(t *testing.T) {
	extr := newCodeExtractor("/root", []string{"/root/src/main.rs"})
	content := "use std::collections::HashMap;"
	got := extr.Extract(content, "/root/src/main.rs")
	if len(got) != 0 {
		t.Fatalf("expected no references for stdlib use, got %v", got)
	}
}

// --- TypeScript/JavaScript ---

func TestSourceCodeReferenceExtractor_JS_Relative(t *testing.T) {
	files := []string{"/root/util.ts", "/root/main.ts"}
	extr := newCodeExtractor("/root", files)
	content := `import { util } from './util';`
	got := extr.Extract(content, "/root/main.ts")
	assertPaths(t, got, []string{"/root/util.ts"})
}

func TestSourceCodeReferenceExtractor_JS_RequireRelative(t *testing.T) {
	files := []string{"/root/util.js", "/root/main.js"}
	extr := newCodeExtractor("/root", files)
	content := `const util = require('./util');`
	got := extr.Extract(content, "/root/main.js")
	assertPaths(t, got, []string{"/root/util.js"})
}

func TestSourceCodeReferenceExtractor_JS_NonRelativeIndex(t *testing.T) {
	files := []string{"/root/utils/index.ts", "/root/main.ts"}
	extr := newCodeExtractor("/root", files)
	content := `import { helper } from 'utils';`
	got := extr.Extract(content, "/root/main.ts")
	assertPaths(t, got, []string{"/root/utils/index.ts"})
}

func TestSourceCodeReferenceExtractor_JS_ExternalDropped(t *testing.T) {
	extr := newCodeExtractor("/root", []string{"/root/main.ts"})
	content := `import React from 'react';`
	got := extr.Extract(content, "/root/main.ts")
	if len(got) != 0 {
		t.Fatalf("expected no references for external package, got %v", got)
	}
}

// --- C/C++ ---

func TestSourceCodeReferenceExtractor_C_Relative(t *testing.T) {
	files := []string{"/root/util.h", "/root/main.c"}
	extr := newCodeExtractor("/root", files)
	content := `#include "util.h"`
	got := extr.Extract(content, "/root/main.c")
	assertPaths(t, got, []string{"/root/util.h"})
}

func TestSourceCodeReferenceExtractor_C_SystemIncludeDropped(t *testing.T) {
	extr := newCodeExtractor("/root", []string{"/root/main.c"})
	content := `#include <stdio.h>`
	got := extr.Extract(content, "/root/main.c")
	if len(got) != 0 {
		t.Fatalf("expected no references for system include, got %v", got)
	}
}

// --- Ruby ---

func TestSourceCodeReferenceExtractor_Ruby_RequireRelative(t *testing.T) {
	files := []string{"/root/util.rb", "/root/main.rb"}
	extr := newCodeExtractor("/root", files)
	content := `require_relative 'util'`
	got := extr.Extract(content, "/root/main.rb")
	assertPaths(t, got, []string{"/root/util.rb"})
}

func TestSourceCodeReferenceExtractor_Ruby_RequireNonRelative(t *testing.T) {
	files := []string{"/root/lib/util.rb", "/root/main.rb"}
	extr := newCodeExtractor("/root", files)
	content := `require 'lib/util'`
	got := extr.Extract(content, "/root/main.rb")
	assertPaths(t, got, []string{"/root/lib/util.rb"})
}

func TestSourceCodeReferenceExtractor_Ruby_GemDropped(t *testing.T) {
	extr := newCodeExtractor("/root", []string{"/root/main.rb"})
	content := `require 'json'`
	got := extr.Extract(content, "/root/main.rb")
	if len(got) != 0 {
		t.Fatalf("expected no references for gem require, got %v", got)
	}
}

// --- Java ---

func TestSourceCodeReferenceExtractor_Java_NonRelative(t *testing.T) {
	files := []string{"/root/com/foo/Bar.java", "/root/Main.java"}
	extr := newCodeExtractor("/root", files)
	content := "import com.foo.Bar;"
	got := extr.Extract(content, "/root/Main.java")
	assertPaths(t, got, []string{"/root/com/foo/Bar.java"})
}

func TestSourceCodeReferenceExtractor_Java_ExternalDropped(t *testing.T) {
	extr := newCodeExtractor("/root", []string{"/root/Main.java"})
	content := "import java.util.List;"
	got := extr.Extract(content, "/root/Main.java")
	if len(got) != 0 {
		t.Fatalf("expected no references for jdk import, got %v", got)
	}
}

// --- Cross-cutting ---

func TestSourceCodeReferenceExtractor_UnrecognizedExtension(t *testing.T) {
	extr := newCodeExtractor("/root", []string{"/root/main.txt"})
	got := extr.Extract("import \"fmt\"", "/root/main.txt")
	if len(got) != 0 {
		t.Fatalf("expected no references for unrecognized extension, got %v", got)
	}
}

func TestSourceCodeReferenceExtractor_NoKnownFiles_RelativeStillResolves(t *testing.T) {
	// Extract called without SetKnownFiles: relative resolution doesn't need
	// the known-file set (pure path computation, like markdown/HTML links),
	// so it should still work and must not panic.
	extr := &SourceCodeReferenceExtractor{Root: "/root"}
	got := extr.Extract(`#include "util.h"`, "/root/main.c")
	assertPaths(t, got, []string{"/root/util.h"})
}

func TestSourceCodeReferenceExtractor_NoKnownFiles_NonRelativeDropped(t *testing.T) {
	// Non-relative (dotted/module) resolution requires the known-file set;
	// without it, nothing can match and no reference is produced.
	extr := &SourceCodeReferenceExtractor{Root: "/root"}
	got := extr.Extract(`import "github.com/x/y/pkgname"`, "/root/main.go")
	if len(got) != 0 {
		t.Fatalf("expected no references without a known file set, got %v", got)
	}
}

func TestNewSourceCodeReferenceExtractor(t *testing.T) {
	extr := NewSourceCodeReferenceExtractor("/root")
	if fsa, ok := extr.(interface{ SetKnownFiles([]string) }); ok {
		fsa.SetKnownFiles([]string{"/root/main.go"})
	} else {
		t.Fatal("expected SourceCodeReferenceExtractor to implement FileSetAware")
	}
	got := extr.Extract(`import "fmt"`, "/root/main.go")
	if len(got) != 0 {
		t.Fatalf("expected no references for stdlib import, got %v", got)
	}
}

func TestDedupeNonEmpty_DuplicatesAndEmpty(t *testing.T) {
	got := dedupeNonEmpty([]string{"/a", "", "/a", "/b"})
	assertPaths(t, got, []string{"/a", "/b"})
}

func TestResolveDotted_EmptySegments(t *testing.T) {
	extr := newCodeExtractor("/root", []string{"/root/main.go"})
	if _, ok := extr.resolveDotted(nil, []string{".go"}, nil); ok {
		t.Fatal("expected no match for empty segments")
	}
}

func TestSourceCodeReferenceExtractor_Python_BarePackageImportSkipped(t *testing.T) {
	files := []string{"/root/pkg/__init__.py", "/root/pkg/main.py"}
	extr := newCodeExtractor("/root", files)
	content := "from . import sibling"
	got := extr.Extract(content, "/root/pkg/main.py")
	if len(got) != 0 {
		t.Fatalf("expected no reference for 'from . import x', got %v", got)
	}
}

func TestSourceCodeReferenceExtractor_JS_RelativeWithExplicitExtension(t *testing.T) {
	files := []string{"/root/util.ts", "/root/main.ts"}
	extr := newCodeExtractor("/root", files)
	content := `import { util } from './util.ts';`
	got := extr.Extract(content, "/root/main.ts")
	assertPaths(t, got, []string{"/root/util.ts"})
}

func TestSourceCodeReferenceExtractor_JS_RelativeUnresolved(t *testing.T) {
	extr := newCodeExtractor("/root", []string{"/root/main.ts"})
	content := `import { util } from './missing';`
	got := extr.Extract(content, "/root/main.ts")
	if len(got) != 0 {
		t.Fatalf("expected no reference for a relative import with no matching file, got %v", got)
	}
}

func TestIsSourceCodeExtension(t *testing.T) {
	for _, ext := range []string{".go", ".py", ".rs", ".ts", ".tsx", ".js", ".jsx", ".c", ".h", ".cpp", ".hpp", ".cc", ".rb", ".java"} {
		if !IsSourceCodeExtension(ext) {
			t.Fatalf("expected %s to be a recognized source extension", ext)
		}
	}
	if IsSourceCodeExtension(".md") {
		t.Fatal("expected .md to not be a recognized source extension")
	}
}
