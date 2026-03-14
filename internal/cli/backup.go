package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Backup copies a .draft file (and WAL/SHM companions) to outputPath.
// If outputPath is empty it generates one from the source name and a timestamp.
func Backup(draftPath string, outputPath string) error {
	if outputPath == "" {
		base := filepath.Base(draftPath)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		ts := time.Now().Format("20060102-150405")
		outputPath = fmt.Sprintf("%s-backup-%s.draft", name, ts)
	}

	size, err := copyFile(draftPath, outputPath)
	if err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	// Copy WAL and SHM companion files if they exist.
	for _, suffix := range []string{"-wal", "-shm"} {
		src := draftPath + suffix
		if _, statErr := os.Stat(src); os.IsNotExist(statErr) {
			continue
		}
		if _, cpErr := copyFile(src, outputPath+suffix); cpErr != nil {
			return fmt.Errorf("backup companion %s: %w", suffix, cpErr)
		}
	}

	fmt.Printf("Backed up %s → %s (%d bytes)\n", draftPath, outputPath, size)
	return nil
}

// Restore copies a backup file to the target path.
func Restore(backupPath string, targetPath string) error {
	if _, err := copyFile(backupPath, targetPath); err != nil {
		return fmt.Errorf("restore: %w", err)
	}

	// Restore companion files if present.
	for _, suffix := range []string{"-wal", "-shm"} {
		src := backupPath + suffix
		if _, statErr := os.Stat(src); os.IsNotExist(statErr) {
			continue
		}
		if _, cpErr := copyFile(src, targetPath+suffix); cpErr != nil {
			return fmt.Errorf("restore companion %s: %w", suffix, cpErr)
		}
	}

	fmt.Printf("Restored %s → %s\n", backupPath, targetPath)
	return nil
}

// copyFile copies src to dst byte-by-byte and returns the number of bytes written.
func copyFile(src, dst string) (int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("open source %q: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return 0, fmt.Errorf("create destination %q: %w", dst, err)
	}
	defer out.Close()

	n, err := io.Copy(out, in)
	if err != nil {
		return 0, fmt.Errorf("copy %q → %q: %w", src, dst, err)
	}
	if err := out.Sync(); err != nil {
		return 0, fmt.Errorf("sync %q: %w", dst, err)
	}
	return n, nil
}
