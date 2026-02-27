package vagrant

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// maxProfileTarSize is the maximum allowed size for the profile tar archive (100MB).
const maxProfileTarSize = 100 * 1024 * 1024

// createProfileTarFunc is a package-level injectable for testing.
// Follows the existing extractOAuthCredentialsFunc pattern in oauth.go.
var createProfileTarFunc = createProfileTar

// profileDir defines a source directory on the host and its tar path prefix.
type profileDir struct {
	srcDir  string // absolute path on host
	tarBase string // path prefix in tar archive
}

// createProfileTar builds an in-memory tar archive of the user's Claude Code
// profile directories (skills, commands, agents, rules). Symlinks are dereferenced
// and validated. Returns empty bytes with nil error if no profile dirs exist.
func createProfileTar(homeDir string) ([]byte, error) {
	dirs := []profileDir{
		{filepath.Join(homeDir, ".claude", "skills"), "skills"},
		{filepath.Join(homeDir, ".claude", "commands"), "commands"},
		{filepath.Join(homeDir, ".claude", "agents"), "agents"},
		{filepath.Join(homeDir, ".claude", "rules"), "rules"},
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	totalSize := int64(0)
	hasContent := false

	for _, dir := range dirs {
		written, err := addDirToTar(tw, dir, homeDir, &totalSize)
		if err != nil {
			return nil, err
		}
		if written {
			hasContent = true
		}
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize tar archive: %w", err)
	}

	if !hasContent {
		return []byte{}, nil
	}

	return buf.Bytes(), nil
}

// addDirToTar walks a single profile directory and adds its entries to the tar.
// Returns true if any content was written, false if the directory was missing.
func addDirToTar(tw *tar.Writer, dir profileDir, homeDir string, totalSize *int64) (bool, error) {
	if _, err := os.Stat(dir.srcDir); os.IsNotExist(err) {
		return false, nil
	}

	claudeBase := filepath.Join(homeDir, ".claude")
	wrote := false

	err := filepath.WalkDir(dir.srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip entries with walk errors
		}

		if shouldSkipEntry(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(claudeBase, path)
		if err != nil {
			return nil // skip entries where relative path fails
		}

		if isExcludedProfilePath(relPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Detect symlinks with Lstat, then validate and dereference
		linfo, err := os.Lstat(path)
		if err != nil {
			return nil
		}

		if linfo.Mode()&os.ModeSymlink != 0 {
			if err := validateSymlinkTarget(path, homeDir); err != nil {
				return nil // skip invalid symlinks silently
			}
		}

		// Follow symlinks with os.Stat for the real file info
		info, err := os.Stat(path)
		if err != nil {
			return nil // broken symlink — skip
		}

		if info.IsDir() {
			return addDirHeader(tw, relPath, info.ModTime())
		}

		if !info.Mode().IsRegular() {
			return nil // skip non-regular files (pipes, sockets, etc.)
		}

		return addFileEntry(tw, path, relPath, info, totalSize, &wrote)
	})

	return wrote, err
}

// addDirHeader writes a directory entry to the tar archive.
func addDirHeader(tw *tar.Writer, relPath string, modTime time.Time) error {
	header := &tar.Header{
		Typeflag: tar.TypeDir,
		Name:     relPath + "/",
		Mode:     0755,
		ModTime:  modTime,
	}
	return tw.WriteHeader(header)
}

// addFileEntry reads a file and writes it to the tar archive with size tracking.
func addFileEntry(tw *tar.Writer, path, relPath string, info fs.FileInfo, totalSize *int64, wrote *bool) error {
	*totalSize += info.Size()
	if *totalSize > maxProfileTarSize {
		return fmt.Errorf("profile tar exceeds %d byte limit", maxProfileTarSize)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil // skip unreadable files
	}

	header := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     relPath,
		Size:     int64(len(content)),
		Mode:     0644,
		ModTime:  info.ModTime(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header for %s: %w", relPath, err)
	}

	if _, err := tw.Write(content); err != nil {
		return fmt.Errorf("failed to write tar content for %s: %w", relPath, err)
	}

	*wrote = true
	return nil
}

// shouldSkipEntry returns true for filesystem entries that should never be synced.
func shouldSkipEntry(name string) bool {
	return name == ".DS_Store" || name == ".git"
}

// isExcludedProfilePath returns true for paths that are excluded from the tar.
// These are either VM-managed or handled separately in other sync steps.
func isExcludedProfilePath(relPath string) bool {
	normalized := filepath.ToSlash(relPath)

	// Exclude CLAUDE.md (handled separately in step 6 merge)
	if strings.HasSuffix(normalized, "/CLAUDE.md") || normalized == "CLAUDE.md" {
		return true
	}

	// Exclude operating-in-vagrant skill (VM-managed by EnsureSudoSkill)
	if strings.HasPrefix(normalized, "skills/operating-in-vagrant/") || normalized == "skills/operating-in-vagrant" {
		return true
	}

	// Exclude settings.local.json (VM-managed)
	if normalized == "settings.local.json" || strings.HasSuffix(normalized, "/settings.local.json") {
		return true
	}

	return false
}

// validateSymlinkTarget ensures a symlink resolves to a path within allowed directories.
// Allowed prefixes: ~/.claude/ and ~/.agents/ (user profile directories).
func validateSymlinkTarget(path, homeDir string) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("cannot resolve symlink %s: %w", path, err)
	}

	allowedPrefixes := []string{
		filepath.Join(homeDir, ".claude") + string(filepath.Separator),
		filepath.Join(homeDir, ".agents") + string(filepath.Separator),
	}

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(resolved, prefix) {
			return nil
		}
	}

	return fmt.Errorf("symlink %s points outside allowed directories: %s", path, resolved)
}

// syncProfileToVM creates a tar of the user's profile directories and extracts
// it inside the VM via a single SSH pipe. Post-extraction, file and directory
// permissions are normalized to 644/755 respectively.
func (m *Manager) syncProfileToVM() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	tarData, err := createProfileTarFunc(homeDir)
	if err != nil {
		return fmt.Errorf("failed to create profile tar: %w", err)
	}

	if len(tarData) == 0 {
		return nil
	}

	cmd := m.vagrantCmd("ssh", "-c", "mkdir -p ~/.claude && tar xf - -C ~/.claude/")
	cmd.Stdin = bytes.NewReader(tarData)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract profile tar in VM: %w", err)
	}

	chmodCmd := m.vagrantCmd("ssh", "-c",
		"find ~/.claude/skills ~/.claude/commands ~/.claude/agents ~/.claude/rules -type f -exec chmod 644 {} + 2>/dev/null; "+
			"find ~/.claude/skills ~/.claude/commands ~/.claude/agents ~/.claude/rules -type d -exec chmod 755 {} + 2>/dev/null")
	_ = chmodCmd.Run() // Non-fatal

	return nil
}
