package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

var originalAspectPattern = []byte{0x39, 0x8E, 0xE3, 0x3F} // float32(16/9)

const (
	targetExeName = "iw4sp.exe"
	backupExeName = "iw4sp.exe.ultrawide-backup"
)

type patchResult struct {
	Count       int
	AspectRatio float32
	BackupPath  string
}

func resolutionPattern(width, height int) ([]byte, float32, error) {
	if width <= 0 || height <= 0 {
		return nil, 0, errors.New("resolution must be greater than zero")
	}
	ratio := float32(width) / float32(height)
	if ratio <= float32(16.0/9.0)+0.0001 {
		return nil, 0, fmt.Errorf("%dx%d is not wider than 16:9, so an ultrawide patch is not needed", width, height)
	}
	out := make([]byte, 4)
	binary.LittleEndian.PutUint32(out, math.Float32bits(ratio))
	return out, ratio, nil
}

func findAll(data, pattern []byte) []int {
	var offsets []int
	for start := 0; start <= len(data)-len(pattern); {
		i := bytes.Index(data[start:], pattern)
		if i < 0 {
			break
		}
		off := start + i
		offsets = append(offsets, off)
		start = off + len(pattern)
	}
	return offsets
}

func copyBytes(src []byte) []byte {
	out := make([]byte, len(src))
	copy(out, src)
	return out
}

func sameBuildExceptAspect(current, backup []byte) bool {
	if len(current) != len(backup) {
		return false
	}
	offsets := findAll(backup, originalAspectPattern)
	if len(offsets) == 0 {
		return false
	}

	ignored := make([]bool, len(current))
	for _, off := range offsets {
		for i := 0; i < len(originalAspectPattern); i++ {
			ignored[off+i] = true
		}
	}
	for i := range current {
		if !ignored[i] && current[i] != backup[i] {
			return false
		}
	}
	return true
}

func archiveBackup(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	stamp := time.Now().Format("20060102-150405")
	archived := fmt.Sprintf("%s.%s", path, stamp)
	return os.Rename(path, archived)
}

func writeFilePreserveMode(path string, data []byte, mode os.FileMode) error {
	temp := path + ".ultrawide-temp"
	if err := os.WriteFile(temp, data, mode); err != nil {
		return err
	}
	defer os.Remove(temp)

	// Windows cannot atomically rename over an existing file with os.Rename.
	// Writing through the existing path is safe here because a verified backup exists first.
	if err := os.WriteFile(path, data, mode); err != nil {
		return err
	}
	return nil
}

func applyPatch(dir string, width, height int) (patchResult, error) {
	var result patchResult
	replacement, ratio, err := resolutionPattern(width, height)
	if err != nil {
		return result, err
	}

	target := filepath.Join(dir, targetExeName)
	backup := filepath.Join(dir, backupExeName)

	current, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			return result, fmt.Errorf("%s was not found next to this program", targetExeName)
		}
		return result, fmt.Errorf("could not read %s: %w", targetExeName, err)
	}
	info, err := os.Stat(target)
	if err != nil {
		return result, err
	}

	currentOldOffsets := findAll(current, originalAspectPattern)
	var pristine []byte

	backupData, backupErr := os.ReadFile(backup)
	backupExists := backupErr == nil
	if backupErr != nil && !os.IsNotExist(backupErr) {
		return result, fmt.Errorf("could not read backup: %w", backupErr)
	}

	if len(currentOldOffsets) > 0 {
		if backupExists && !bytes.Equal(current, backupData) {
			if err := archiveBackup(backup); err != nil {
				return result, fmt.Errorf("could not archive the old backup: %w", err)
			}
			backupExists = false
		}
		if !backupExists {
			if err := os.WriteFile(backup, current, info.Mode()); err != nil {
				return result, fmt.Errorf("could not create backup: %w", err)
			}
		}
		pristine = copyBytes(current)
	} else {
		if !backupExists {
			return result, errors.New("this executable appears to be patched already, but no matching backup was found")
		}
		if !sameBuildExceptAspect(current, backupData) {
			return result, errors.New("the current executable and backup do not appear to be the same game build; verify the game files in Steam, then run the patcher again")
		}
		pristine = copyBytes(backupData)
	}

	offsets := findAll(pristine, originalAspectPattern)
	if len(offsets) == 0 {
		return result, errors.New("the expected 16:9 aspect-ratio pattern was not found in this MW2 build")
	}

	patched := copyBytes(pristine)
	for _, off := range offsets {
		copy(patched[off:off+4], replacement)
	}

	if err := writeFilePreserveMode(target, patched, info.Mode()); err != nil {
		return result, fmt.Errorf("could not write %s (make sure MW2 is closed): %w", targetExeName, err)
	}

	check, err := os.ReadFile(target)
	if err != nil {
		return result, fmt.Errorf("could not verify patched executable: %w", err)
	}
	for _, off := range offsets {
		if off+4 > len(check) || !bytes.Equal(check[off:off+4], replacement) {
			return result, fmt.Errorf("verification failed at offset 0x%X; restore the backup before launching the game", off)
		}
	}

	result.Count = len(offsets)
	result.AspectRatio = ratio
	result.BackupPath = backup
	return result, nil
}

func restoreOriginal(dir string) error {
	target := filepath.Join(dir, targetExeName)
	backup := filepath.Join(dir, backupExeName)

	data, err := os.ReadFile(backup)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("no backup has been created yet")
		}
		return fmt.Errorf("could not read backup: %w", err)
	}
	info, err := os.Stat(backup)
	if err != nil {
		return err
	}
	if err := writeFilePreserveMode(target, data, info.Mode()); err != nil {
		return fmt.Errorf("could not restore %s (make sure MW2 is closed): %w", targetExeName, err)
	}
	return nil
}
