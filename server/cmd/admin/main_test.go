package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDeleteArgs(t *testing.T) {
	id, withMemes, yes, err := parseDeleteArgs([]string{"abc", "-yes", "-with-memes"}, true)
	if err != nil || id != "abc" || !withMemes || !yes {
		t.Fatalf("parse = %q %v %v %v", id, withMemes, yes, err)
	}
	if _, _, _, err := parseDeleteArgs([]string{"-yes"}, false); err == nil {
		t.Fatal("expected missing id error")
	}
	if _, _, _, err := parseDeleteArgs([]string{"abc", "-with-memes"}, false); err == nil {
		t.Fatal("expected unknown flag error for memes")
	}
}

func TestUploadPath(t *testing.T) {
	root := t.TempDir()
	got := uploadPath(root, "/uploads/memes/a.png")
	want := filepath.Join(root, "uploads", "memes", "a.png")
	if got != want {
		t.Fatalf("path = %q want %q", got, want)
	}
	if got := uploadPath(root, "/uploads/../../etc/passwd"); got != "" {
		t.Fatalf("traversal path = %q", got)
	}
	if got := uploadPath(root, "https://example.com/x.png"); got != "" {
		t.Fatalf("external path = %q", got)
	}
	if !strings.HasSuffix(got, "a.png") {
		t.Fatalf("suffix = %q", got)
	}
}
