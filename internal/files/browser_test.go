package files

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIsPathAllowed_Boundaries(t *testing.T) {
	root := t.TempDir()

	allowed := filepath.Join(root, "opt")
	sibling := filepath.Join(root, "opt-backup")
	for _, d := range []string{allowed, sibling} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	nested := filepath.Join(allowed, "app", "logs")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	b := NewBrowser([]string{allowed})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"the allowed root itself", allowed, true},
		{"a nested directory", nested, true},
		{"a sibling sharing the prefix", sibling, false},
		{"a file in the sibling", filepath.Join(sibling, "secrets"), false},
		{"the parent", root, false},
		{"traversal back out", filepath.Join(allowed, "..", "opt-backup"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := b.IsPathAllowed(tt.path); got != tt.want {
				t.Errorf("IsPathAllowed(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsPathAllowed_SymlinkEscape(t *testing.T) {
	root := t.TempDir()

	allowed := filepath.Join(root, "allowed")
	secret := filepath.Join(root, "secret")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(secret, 0o755); err != nil {
		t.Fatal(err)
	}

	secretFile := filepath.Join(secret, "key")
	if err := os.WriteFile(secretFile, []byte("sensitive"), 0o600); err != nil {
		t.Fatal(err)
	}

	// A symlink planted inside the allowed tree pointing out of it.
	link := filepath.Join(allowed, "escape")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	b := NewBrowser([]string{allowed})

	if b.IsPathAllowed(link) {
		t.Error("symlink pointing outside the allowed tree was accepted")
	}
	if b.IsPathAllowed(filepath.Join(link, "key")) {
		t.Error("file reached through an escaping symlink was accepted")
	}
	if _, err := b.ReadFile(filepath.Join(link, "key")); err == nil {
		t.Error("ReadFile followed a symlink out of the allowed tree")
	}
}

func TestIsPathAllowed_Wildcard(t *testing.T) {
	b := NewBrowser([]string{"*"})

	if !b.IsPathAllowed("/anywhere/at/all") {
		t.Error("wildcard should allow any path")
	}
	if got := b.GetAllowedPaths(); len(got) != 1 || got[0] != "/" {
		t.Errorf("GetAllowedPaths() = %v, want [/]", got)
	}
}

func TestGetDiskUsage_TopFilesBounded(t *testing.T) {
	root := t.TempDir()

	// More files than the reported top-N, with increasing sizes.
	for i := 0; i < TopFilesCount+5; i++ {
		body := make([]byte, i+1)
		if err := os.WriteFile(filepath.Join(root, string(rune('a'+i))), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	b := NewBrowser([]string{root})

	usage, err := b.GetDiskUsage(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	if usage.FileCount != TopFilesCount+5 {
		t.Errorf("FileCount = %d, want %d", usage.FileCount, TopFilesCount+5)
	}
	if len(usage.LargestFiles) != TopFilesCount {
		t.Fatalf("LargestFiles = %d entries, want %d", len(usage.LargestFiles), TopFilesCount)
	}
	// Must be the biggest ones, sorted descending.
	for i := 1; i < len(usage.LargestFiles); i++ {
		if usage.LargestFiles[i-1].Size < usage.LargestFiles[i].Size {
			t.Errorf("LargestFiles not sorted descending at %d", i)
		}
	}
	if want := int64(TopFilesCount + 5); usage.LargestFiles[0].Size != want {
		t.Errorf("largest file = %d bytes, want %d", usage.LargestFiles[0].Size, want)
	}
}

func TestGetDiskUsage_CancelledContext(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 2000; i++ {
		if err := os.WriteFile(filepath.Join(root, string(rune('a'+i%26))+string(rune('a'+i/26))), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	b := NewBrowser([]string{root})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := b.GetDiskUsage(ctx, root); err == nil {
		t.Error("expected an error once the caller went away")
	}
}
