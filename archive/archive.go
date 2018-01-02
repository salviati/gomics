// Copyright (c) 2013-2018 Utkan Güngördü <utkan@freeconsole.org>
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
	"path/filepath"
	"strings"
)

var (
	ErrBounds = errors.New("Image index out of bounds.")
)

type Archive interface {
	Load(i int, autorotate bool) (*gdk.Pixbuf, error)
	Name(i int) (string, error)
	Len() int
	Close() error
}

const (
	MaxArchiveEntries = 4096 * 64
)

func NewArchive(path string) (Archive, error) {

	switch strings.ToLower(filepath.Ext(path)) {
	case ".zip", ".cbz":
		return NewZip(path)
	case ".7z", ".rar", ".tar", ".tgz", ".tbz2", ".cb7", ".cbr", ".cbt", ".lha":
		// TODO
	case ".gz":
		if strings.HasSuffix(strings.ToLower(path), ".tar.gz") {
			// TODO
		}
	}

	return nil, errors.New("Unknown archive type")
}
