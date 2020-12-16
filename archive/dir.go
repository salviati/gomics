// Copyright (c) 2013-2020 Utkan Güngördü <utkan@freeconsole.org>
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
	"errors"
	"github.com/gotk3/gotk3/gdk"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
)

type Dir struct {
	filenames filenames
	name      string
	path      string
}

/* Reads filenames from a given zip archive, and sorts them */
func NewDir(path string) (*Dir, error) {
	var err error

	d := new(Dir)

	d.name = filepath.Base(path)
	d.path = path
	d.filenames = make([]string, 0, MaxArchiveEntries)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, fi := range files {
		if ExtensionMatch(fi.Name(), ImageExtensions) == false {
			continue
		}
		d.filenames = append(d.filenames, fi.Name())
	}

	if len(d.filenames) == 0 {
		return nil, errors.New(d.name + ": no images in the directory")
	}

	sort.Sort(d.filenames)

	return d, nil
}

func (d *Dir) checkbounds(i int) error {
	if i < 0 || i >= len(d.filenames) {
		return ErrBounds
	}
	return nil
}

func (d *Dir) Load(i int, autorotate bool) (*gdk.Pixbuf, error) {
	if err := d.checkbounds(i); err != nil {
		return nil, err
	}

	path := filepath.Join(d.path, d.filenames[i])
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()
	return LoadPixbuf(f, autorotate)
}

func (d *Dir) Name(i int) (string, error) {
	if err := d.checkbounds(i); err != nil {
		return "", err
	}

	return d.filenames[i], nil
}

func (d *Dir) Len() int {
	return len(d.filenames)
}

func (d *Dir) Close() error {
	return nil
}
