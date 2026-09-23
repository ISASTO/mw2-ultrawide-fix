package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestResolutionPattern3440x1440(t *testing.T) {
	got, ratio, err := resolutionPattern(3440, 1440)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x8E, 0xE3, 0x18, 0x40}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
	if ratio != float32(3440.0/1440.0) {
		t.Fatalf("unexpected ratio: %f", ratio)
	}
}

func TestApplyAndRestore(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, targetExeName)
	original := append([]byte("header"), originalAspectPattern...)
	original = append(original, []byte("middle")...)
	original = append(original, originalAspectPattern...)
	original = append(original, []byte("footer")...)
	if err := os.WriteFile(target, original, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := applyPatch(dir, 3440, 1440)
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 2 {
		t.Fatalf("patched %d occurrences, want 2", result.Count)
	}

	patched, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	repl := make([]byte, 4)
	binary.LittleEndian.PutUint32(repl, math.Float32bits(float32(3440.0/1440.0)))
	if len(findAll(patched, repl)) != 2 {
		t.Fatalf("patched executable does not contain two replacement patterns")
	}

	if _, err := applyPatch(dir, 2560, 1080); err != nil {
		t.Fatal(err)
	}

	if err := restoreOriginal(dir); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored, original) {
		t.Fatal("restored executable differs from original")
	}
}
