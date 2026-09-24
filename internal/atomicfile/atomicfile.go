// Package atomicfile replaces a file in one step, through a temporary in the
// same directory and a rename.
//
// The manifest and the boundary file are artifacts, and os.WriteFile
// truncates before it writes: a crash, a full disk, or a reader arriving
// mid-write sees an artifact that has lost most of itself. Closing two
// thousand differences rewrites the manifest two thousand times, and a
// concurrent reader catching a half-written file is the cheap version of the
// failure this prevents.
package atomicfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// newFileMode is the mode of a file that did not exist before: an artifact
// is tracked in git and read by other tools, so it is world-readable.
const newFileMode fs.FileMode = 0o644

// Write replaces path with b. An existing file keeps its permission bits. The
// temporary lives in the same directory because rename is only atomic within
// a filesystem.
func Write(path string, b []byte) error {
	mode := newFileMode
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	dir, name := filepath.Split(path)
	f, err := os.CreateTemp(dir, name+".*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp) //nolint:errcheck // a leftover after a failed write costs nothing
	if _, err = f.Write(b); err != nil {
		f.Close() //nolint:errcheck,gosec // the write error is the one to report
		return err
	}
	if err = f.Chmod(mode); err != nil {
		f.Close() //nolint:errcheck,gosec // same
		return err
	}
	// Sync before the rename: the rename is atomic with respect to readers,
	// not with respect to a machine that loses power holding unflushed pages.
	if err = f.Sync(); err != nil {
		f.Close() //nolint:errcheck,gosec // same
		return err
	}
	if closeErr := f.Close(); closeErr != nil {
		return closeErr
	}
	return os.Rename(tmp, path)
}
