// Copyright (c) 2013-2021 Utkan Güngördü <utkan@freeconsole.org>
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
	"github.com/gotk3/gotk3/gdk"
	"github.com/salviati/gomics/natsort"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Loader interface {
	Load(i int) (*gdk.Pixbuf, error)
	Name(i int) (string, error)
	Len() int
}

// TODO(utkan): check rar support

//var ArchiveExtensions = []string{".zip", ".cbz", ".7z", ".rar", ".tar", ".tgz", ".tbz2", ".cb7", ".cbr", ".cbt"}
var ArchiveExtensions = []string{".zip", ".cbz"}
var ImageExtensions []string

func init() {
	ImageExtensions = make([]string, 0)
	formats := gdk.PixbufGetFormats()
	for _, format := range formats {
		ImageExtensions = append(ImageExtensions, format.GetExtensions()...)
	}

	for i := range ImageExtensions {
		ImageExtensions[i] = "." + ImageExtensions[i] // gdk pixbuf format extensions don't have the leading "."
	}
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

type stringArray []string

func (p stringArray) Len() int           { return len(p) }
func (p stringArray) Less(i, j int) bool { return strings.ToLower(p[i]) < strings.ToLower(p[j]) }
func (p stringArray) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func ListArchives(dir string) (anames []string, err error) {
	file, err := os.Open(dir)
	if err != nil {
		return
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return
	}

	if !fi.IsDir() {
		err = errors.New(dir + " is not a directory!")
		return
	}

	names, err := file.Readdirnames(-1)
	if err != nil {
		return
	}

	anames = make([]string, 0, len(names))
	for _, name := range names {
		var fi os.FileInfo
		fi, err = os.Stat(filepath.Join(dir, name))
		if err != nil {
			return
		}

		if !ExtensionMatch(name, ArchiveExtensions) && !fi.IsDir() {
			// TODO(utkan): don't add empty archives
			continue
		}
		anames = append(anames, name)
	}

	sort.Sort(stringArray(anames)) // TODO(utkan): can use natsort for archives as well

	return
}

func LoadPixbuf(r io.Reader, autorotate bool) (*gdk.Pixbuf, error) {
	w, _ := gdk.PixbufLoaderNew()
	defer w.Close()
	_, err := io.Copy(w, r)
	if err != nil {
		return nil, err
	}

	pixbuf, err := w.GetPixbuf()
	if err != nil {
		return nil, err
	}

	if autorotate == false {
		return pixbuf, nil
	}

	return pixbuf.ApplyEmbeddedOrientation()
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
