package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeDirectoryHash(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "sumaron-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test files
	file1 := filepath.Join(tmpDir, "file1.md")
	if err := os.WriteFile(file1, []byte("# Hello World"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}

	file2 := filepath.Join(tmpDir, "file2.json")
	if err := os.WriteFile(file2, []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	// File with ignored extension
	file3 := filepath.Join(tmpDir, "file3.png")
	if err := os.WriteFile(file3, []byte("fake binary data"), 0644); err != nil {
		t.Fatalf("failed to write file3: %v", err)
	}

	allowedExts := map[string]bool{
		".md":   true,
		".json": true,
	}

	fileList, hash1, err := computeDirectoryHash(tmpDir, allowedExts, 20, 2)
	if err != nil {
		t.Fatalf("computeDirectoryHash failed: %v", err)
	}

	if len(fileList) != 2 {
		t.Errorf("expected 2 files, got %d", len(fileList))
	}

	// Change file1 content and verify hash changes
	if err := os.WriteFile(file1, []byte("# Hello World modified"), 0644); err != nil {
		t.Fatalf("failed to update file1: %v", err)
	}

	_, hash2, err := computeDirectoryHash(tmpDir, allowedExts, 20, 2)
	if err != nil {
		t.Fatalf("computeDirectoryHash failed: %v", err)
	}

	if hash1 == hash2 {
		t.Errorf("expected hashes to differ after file modification, but got same hash: %s", hash1)
	}
}

func TestCheckCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sumaron-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	localPath := filepath.Join(tmpDir, ".sumaron.json")
	centralPath := filepath.Join(tmpDir, "central-cache.json")
	hash := "abcd1234efgh5678"
	summary := "This is a test summary."

	// Test cache miss initially
	if _, found := checkCache(localPath, centralPath, tmpDir, hash); found {
		t.Error("expected cache miss, got cache hit")
	}

	// Test local cache hit
	cacheEntry := SumaronCache{
		Timestamp: "2026-01-01T00:00:00Z",
		Hash:      hash,
		Summary:   summary,
	}
	if err := saveLocalCache(localPath, cacheEntry); err != nil {
		t.Fatalf("saveLocalCache failed: %v", err)
	}

	s, found := checkCache(localPath, centralPath, tmpDir, hash)
	if !found {
		t.Error("expected local cache hit, got cache miss")
	}
	if s != summary {
		t.Errorf("expected summary %q, got %q", summary, s)
	}

	// Test cache hash mismatch (local cache exists but with different hash)
	s2, found2 := checkCache(localPath, centralPath, tmpDir, "different-hash")
	if found2 {
		t.Errorf("expected cache miss on hash mismatch, but got hit with summary: %q", s2)
	}

	// Test central cache hit (with no local cache)
	os.Remove(localPath) // remove local cache to verify central cache lookup
	if err := saveCentralCache(centralPath, tmpDir, cacheEntry); err != nil {
		t.Fatalf("saveCentralCache failed: %v", err)
	}

	s3, found3 := checkCache(localPath, centralPath, tmpDir, hash)
	if !found3 {
		t.Error("expected central cache hit, got cache miss")
	}
	if s3 != summary {
		t.Errorf("expected summary %q, got %q", summary, s3)
	}
}

func TestComputeDirectoryHashLimits(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sumaron-limits-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files at different depths
	// Depth 0:
	if err := os.WriteFile(filepath.Join(tmpDir, "file0.md"), []byte("depth 0"), 0644); err != nil {
		t.Fatal(err)
	}
	// Depth 1:
	sub1 := filepath.Join(tmpDir, "sub1")
	if err := os.Mkdir(sub1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub1, "file1.md"), []byte("depth 1"), 0644); err != nil {
		t.Fatal(err)
	}
	// Depth 2:
	sub2 := filepath.Join(sub1, "sub2")
	if err := os.Mkdir(sub2, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub2, "file2.md"), []byte("depth 2"), 0644); err != nil {
		t.Fatal(err)
	}
	// Depth 3:
	sub3 := filepath.Join(sub2, "sub3")
	if err := os.Mkdir(sub3, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub3, "file3.md"), []byte("depth 3"), 0644); err != nil {
		t.Fatal(err)
	}

	allowedExts := map[string]bool{".md": true}

	// 1. Test depth limit 2 (should include file0, file1, file2; exclude file3)
	fileList, _, err := computeDirectoryHash(tmpDir, allowedExts, 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(fileList) != 3 {
		t.Errorf("expected 3 files at depth <= 2, got %d: %v", len(fileList), fileList)
	}

	// 2. Test depth limit 1 (should include file0, file1; exclude file2, file3)
	fileList1, _, err := computeDirectoryHash(tmpDir, allowedExts, 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(fileList1) != 2 {
		t.Errorf("expected 2 files at depth <= 1, got %d: %v", len(fileList1), fileList1)
	}

	// 3. Test max files limit of 1
	fileListMax, _, err := computeDirectoryHash(tmpDir, allowedExts, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(fileListMax) != 1 {
		t.Errorf("expected max 1 file, got %d: %v", len(fileListMax), fileListMax)
	}
}

func TestWriteSummaryMarkdownWithVisuals(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sumaron-markdown-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Without images
	writeSummaryMarkdown(tmpDir, "This is a summary without images.", "test-model", []string{filepath.Join(tmpDir, "README.md")})

	summaryFile := filepath.Join(tmpDir, "sumaron-summary.md")
	content, err := os.ReadFile(summaryFile)
	if err != nil {
		t.Fatalf("failed to read summary markdown: %v", err)
	}

	if !strings.Contains(string(content), "sumaron_version: 0.2.0") {
		t.Errorf("expected version 0.2.0 in frontmatter, got:\n%s", string(content))
	}
	if strings.Contains(string(content), "assets/logo.png") {
		t.Errorf("expected no logo reference, got:\n%s", string(content))
	}

	// 2. With images created in assets/
	assetsDir := filepath.Join(tmpDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("failed to create assets dir: %v", err)
	}
	_ = os.WriteFile(filepath.Join(assetsDir, "logo.png"), []byte("fakelogo"), 0644)
	_ = os.WriteFile(filepath.Join(assetsDir, "arch_diagram.png"), []byte("fakearch"), 0644)
	_ = os.WriteFile(filepath.Join(assetsDir, "er_diagram.png"), []byte("fakeer"), 0644)

	writeSummaryMarkdown(tmpDir, "This is a summary with all visuals.", "test-model", []string{filepath.Join(tmpDir, "README.md")})

	contentWithVisuals, err := os.ReadFile(summaryFile)
	if err != nil {
		t.Fatalf("failed to read summary markdown: %v", err)
	}

	contentStr := string(contentWithVisuals)
	if !strings.Contains(contentStr, "assets/logo.png") {
		t.Errorf("expected logo embedded in markdown, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "assets/arch_diagram.png") {
		t.Errorf("expected arch diagram embedded in markdown, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "assets/er_diagram.png") {
		t.Errorf("expected er diagram embedded in markdown, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "images:\n  - assets/logo.png") {
		t.Errorf("expected images listed in YAML frontmatter, got:\n%s", contentStr)
	}
}
