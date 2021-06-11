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
	"archive/zip"
	"errors"
	"github.com/gotk3/gotk3/gdk"
	"path/filepath"
	"sort"
)

type Zip struct {
	files  []*zip.File // File elements sorted by their Names
	reader *zip.ReadCloser
	name   string // Name of the Zip file
}

type zipfile []*zip.File

func (p zipfile) Len() int           { return len(p) }
func (p zipfile) Less(i, j int) bool { return strcmp(p[i].Name, p[j].Name, true) }
func (p zipfile) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

/* Reads filenames from a given zip archive, and sorts them */
func NewZip(name string) (*Zip, error) {
	var err error

	ar := new(Zip)

	ar.name = filepath.Base(name)
	ar.files = make([]*zip.File, 0, MaxArchiveEntries)
	ar.reader, err = zip.OpenReader(name)
	if err != nil {
		return nil, err
	}

	for _, f := range ar.reader.File {
		if ExtensionMatch(f.Name, ImageExtensions) == false {
			continue
		}
		ar.files = append(ar.files, f)
	}

	if len(ar.files) == 0 {
		return nil, errors.New(ar.name + ": no images in the zip file")
	}

	sort.Sort(zipfile(ar.files))

	return ar, nil
}

func (ar *Zip) checkbounds(i int) error {
	if i < 0 || i >= len(ar.files) {
		return ErrBounds
	}
	return nil
}

func (ar *Zip) Load(i int, autorotate bool) (*gdk.Pixbuf, error) {
	if err := ar.checkbounds(i); err != nil {
		return nil, err
	}

	f, err := ar.files[i].Open()
	if err != nil {
		return nil, err
	}

	defer f.Close()
	return LoadPixbuf(f, autorotate)
}

func (ar *Zip) Name(i int) (string, error) {
	if err := ar.checkbounds(i); err != nil {
		return "", err
	}

	return ar.files[i].Name, nil
}

func (ar *Zip) Len() int {
	return len(ar.files)
}

func (ar *Zip) Close() error {
	return ar.reader.Close()
}
