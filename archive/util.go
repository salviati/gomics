// Copyright (c) 2013-2025 Utkan Güngördü <utkan@freeconsole.org>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package archive

import (
	"bytes"
	"errors"
	"github.com/mappu/miqt/qt6"
	"github.com/salviati/gomics/natsort"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TODO(utkan): check rar support

// var ArchiveExtensions = []string{".zip", ".cbz", ".7z", ".rar", ".tar", ".tgz", ".tbz2", ".cb7", ".cbr", ".cbt"}
var ArchiveExtensions = []string{".zip", ".cbz"}
var ImageExtensions []string

const ImageReaderAllocLimitMB = 1024

func init() {
	seen := make(map[string]bool)
	addExt := func(ext string) {
		if !seen[ext] {
			seen[ext] = true
			ImageExtensions = append(ImageExtensions, ext)
		}
	}
	for _, format := range qt6.QImageReader_SupportedImageFormats() {
		addExt("." + strings.ToLower(string(format)))
	}

	// No allocation limit: some large images exceed the default cap and fail
	// to decode. 0 removes the limit so they load (one-time global setting).
	qt6.QImageReader_SetAllocationLimit(ImageReaderAllocLimitMB)
}

func ExtensionMatch(p string, extensions []string) bool {
	pext := strings.ToLower(filepath.Ext(p))
	for _, ext := range extensions {
		if pext == ext {
			return true
		}
	}
	return false
}

// IsImageFile reports whether path is a regular file whose extension is in
// ImageExtensions (a directory or missing path never matches).
func IsImageFile(path string) bool {
	fi, err := os.Stat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return false
	}
	return ExtensionMatch(path, ImageExtensions)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func wrap(val, low, mod int) int {
	val %= mod
	if val < low {
		val = mod + val
	}
	return val
}

var (
	// ErrCurrentNotFound is returned by ArchiveStep when name is not one of
	// dir's archives: a bare image file, or the archive was deleted out from
	// under us.
	ErrCurrentNotFound = errors.New("Couldn't find the current archive under current dir. Deleted, perhaps?")
	// ErrNoMoreArchives is returned by ArchiveStep at a boundary: no archive
	// lies offset steps past name in dir (no more to go to).
	ErrNoMoreArchives = errors.New("No more archives in the directory")
)

// ListArchives returns the names of the archives under dir, i.e. regular
// files whose extension is in ArchiveExtensions. Subdirectories are excluded
// and entries that cannot be stat'ed (broken symlinks, vanished files) are
// skipped rather than aborting the whole listing.
func ListArchives(dir string) ([]string, error) {
	file, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if !fi.IsDir() {
		return nil, errors.New(dir + " is not a directory!")
	}

	names, err := file.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	anames := make([]string, 0, len(names))
	for _, name := range names {
		fi, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			// Skip entries that vanish or are unresolvable rather than aborting.
			continue
		}

		if fi.IsDir() || !ExtensionMatch(name, ArchiveExtensions) {
			// TODO(utkan): don't add empty archives
			continue
		}
		anames = append(anames, name)
	}

	sort.Sort(filenames(anames))

	return anames, nil
}

// ArchiveStep returns the path of the archive offset steps away from name in
// dir: offset +1 is the next archive, -1 the previous. It lists dir once and
// locates name, so a single listing serves both the lookup and the step. The
// returned path is rooted at dir (which must already be absolute), so it
// resolves independently of the process cwd.
func ArchiveStep(dir, name string, offset int) (string, error) {
	anames, err := ListArchives(dir)
	if err != nil {
		return "", err
	}

	cur := -1
	for i, n := range anames {
		if n == name {
			cur = i
		}
	}
	if cur == -1 {
		return "", ErrCurrentNotFound
	}

	which := cur + offset
	if which < 0 || which >= len(anames) {
		return "", ErrNoMoreArchives
	}

	return filepath.Join(dir, anames[which]), nil
}

func LoadQImage(data []byte, ext string, autorotate bool) (*qt6.QImage, error) {
	buf := qt6.NewQBuffer()
	defer buf.Delete()
	buf.SetData(data)
	reader := qt6.NewQImageReader2(buf.QIODevice)
	defer reader.Delete()
	if ext != "" {
		reader.SetFormat([]byte(strings.TrimPrefix(ext, ".")))
	}
	reader.SetAutoTransform(autorotate)
	img := reader.Read() // NOTE: fatal-errors (unrecoverable) on undecodable data — accepted risk for EXIF autorotate
	if img == nil || img.Width() <= 0 {
		return nil, errors.New("failed to decode image")
	}
	return img, nil
}

type File struct {
	*os.File
}

func NewFile(f *os.File) *File {
	return &File{f}
}

func (r *File) Size() (int64, error) {
	fi, err := r.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

func (r *File) SetSize(n int64) error {
	return r.Truncate(n)
}

func (r *File) Ext() string {
	ext := filepath.Ext(r.Name())
	if len(ext) <= 1 || ext[0] != '.' {
		return ""
	}

	return ext[1:]
}

type Buffer struct {
	bytes.Buffer
}

func NewBuffer(data []byte) *Buffer {
	return &Buffer{*bytes.NewBuffer(data)}
}

func (b *Buffer) Seek(offset int64, whence int) (int64, error) {
	return offset, nil
}

func (b *Buffer) SetSize(int64) error {
	return nil
}

func (b *Buffer) Size() (int64, error) {
	return int64(b.Len()), nil
}

func strcmp(a, b string, nat bool) bool {
	if nat {
		return natsort.Less(a, b)
	}
	return a < b
}

type filenames []string

func (p filenames) Len() int           { return len(p) }
func (p filenames) Less(i, j int) bool { return strcmp(p[i], p[j], true) }
func (p filenames) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
