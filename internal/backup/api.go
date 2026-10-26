package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rocky-ads/site/internal/imagestore"
	"github.com/rocky-ads/site/internal/logger"
)

const DefaultDir = "backups"

// DefaultArchivePath returns backups/backup-YYYYMMDD-HHMMSS.tar.gz (UTC).
func DefaultArchivePath(t time.Time) string {
	name := fmt.Sprintf("backup-%s.tar.gz", t.UTC().Format("20060102-150405"))
	return filepath.Join(DefaultDir, name)
}

// ListArchives returns *.tar.gz names in backups/, newest first.
func ListArchives() ([]string, error) {
	entries, err := os.ReadDir(DefaultDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".tar.gz") {
			continue
		}
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return names[i] > names[j]
	})
	return names, nil
}

func ArchivePath(name string) string {
	if filepath.IsAbs(name) {
		return resolveArchivePath(name)
	}
	if filepath.Dir(name) == "." || filepath.Dir(name) == "" {
		return filepath.Join(DefaultDir, resolveArchivePath(name))
	}
	return resolveArchivePath(name)
}

// BackupToArchive writes a tar.gz backup. Empty out uses DefaultArchivePath.
func BackupToArchive(out string, store imagestore.Store, dryRun, verbose bool) (string, error) {
	if dryRun {
		if err := runBackup("", store, true, verbose); err != nil {
			return "", err
		}
		return "", nil
	}
	archivePath := out
	if archivePath == "" {
		archivePath = DefaultArchivePath(time.Now())
	} else {
		archivePath = ArchivePath(archivePath)
	}
	staging, err := os.MkdirTemp("", "backup-*")
	if err != nil {
		return "", fmt.Errorf("create staging dir: %w", err)
	}
	defer os.RemoveAll(staging)

	if err := runBackup(staging, store, false, verbose); err != nil {
		return "", err
	}
	if err := createTarGz(archivePath, staging); err != nil {
		return "", err
	}
	logger.Info("Wrote archive", "path", archivePath)
	return archivePath, nil
}

// RestoreFromArchive restores from a .tar.gz path (relative names use backups/).
// backupKey decrypts archive user fields. Empty uses backups/users.csv
// when that file exists, otherwise config.DBEncryptionKey.
func RestoreFromArchive(from string, store imagestore.Store,
	backupKey []byte, dryRun, verbose bool) error {
	archivePath := ArchivePath(from)
	staging, err := os.MkdirTemp("", "backup-restore-*")
	if err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}
	defer os.RemoveAll(staging)

	if err := extractTarGz(archivePath, staging); err != nil {
		return err
	}
	return runRestore(staging, store, backupKey, dryRun, verbose)
}
