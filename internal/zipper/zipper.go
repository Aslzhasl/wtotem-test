package zipper

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Zipper struct{}

func New() *Zipper { return &Zipper{} }

// ZipDir zips "dir" into "dst" (relative paths inside the zip), skipping common junk.
func (z *Zipper) ZipDir(dir, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the output file itself if zipping "."
		if filepath.Base(path) == filepath.Base(dst) {
			return nil
		}

		// Skip junk dirs
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".idea", ".vscode", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}

		// Skip previously generated artifacts
		name := d.Name()
		if strings.HasSuffix(name, ".zip") || (strings.HasPrefix(name, "report_") && strings.HasSuffix(name, ".json")) {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		return addFile(zw, path, filepath.ToSlash(rel))
	})
}

func addFile(zw *zip.Writer, path, name string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	h, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	h.Name = name
	h.Method = zip.Deflate

	w, err := zw.CreateHeader(h)
	if err != nil {
		return err
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}
