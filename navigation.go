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

package main

import (
	"errors"
	"github.com/mappu/miqt/qt6"
	"github.com/salviati/gomics/archive"
	"github.com/salviati/gomics/imgdiff"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
)

var (
	ErrCurrentNotFound = errors.New("Couldn't find the current archive under current dir. Deleted, perhaps?")
)

func (gui *GUI) Loaded() bool {
	return gui.State.Archive != nil && reflect.ValueOf(gui.State.Archive).IsNil() == false && gui.State.ArchivePath != ""
}

func (gui *GUI) RandomPage() {
	if !gui.Loaded() {
		return
	}

	gui.SetPage(rand.Int() % gui.State.Archive.Len())
}

func (gui *GUI) PreviousPage() {
	if !gui.Loaded() {
		if gui.Config.Seamless {
			gui.PreviousArchive()
		}
		return
	}

	if gui.Config.Random {
		gui.RandomPage()
		return
	}

	n := 1
	if gui.Config.DoublePage && gui.State.ArchivePos > 1 && gui.forceSinglePage() == false {
		n = 2
	}

	if gui.Config.Seamless && gui.State.ArchivePos+1 <= n {
		gui.PreviousArchive()
		return
	}

	gui.SetPage(gui.State.ArchivePos - n)
}

func (gui *GUI) NextPage() {
	if !gui.Loaded() {
		if gui.Config.Seamless {
			gui.NextArchive()
		}
		return
	}

	if gui.Config.Random {
		gui.RandomPage()
		return
	}

	n := 1
	if gui.Config.DoublePage && gui.forceSinglePage() == false && gui.State.Archive.Len() > gui.State.ArchivePos+2 {
		n = 2
	}

	if gui.Config.Seamless && gui.State.Archive.Len()-gui.State.ArchivePos <= n {
		gui.NextArchive()
		return
	}

	gui.SetPage(gui.State.ArchivePos + n)
}

func (gui *GUI) FirstPage() {
	if !gui.Loaded() {
		return
	}

	gui.SetPage(0)
}

func (gui *GUI) LastPage() {
	if !gui.Loaded() {
		return
	}

	if gui.Config.DoublePage && gui.State.Archive.Len() >= 2 {
		gui.SetPage(gui.State.Archive.Len() - 2)
	}
	gui.SetPage(gui.State.Archive.Len() - 1)
}

// ImageHash returns the cached dHash for page n, computing it on demand.
// pixbuf is the *qt6.QImage for the page (or nil to load it fresh).
func (gui *GUI) ImageHash(n int, pixbuf *qt6.QImage) (imgdiff.Hash, bool) {
	if hash, ok := gui.State.ImageHash[n]; ok {
		return hash, true
	}

	if pixbuf == nil {
		var err error
		if pixbuf, err = gui.State.Archive.Load(n, gui.Config.EmbeddedOrientation); err != nil {
			return 0, false
		}
	}

	return imgdiff.Hash(imgdiff.DHash(pixbuf)), true
}

func (gui *GUI) NextScene() {
	if !gui.Loaded() {
		return
	}

	if gui.State.QImageL == nil {
		return
	}
	hash := imgdiff.DHash(gui.State.QImageL)

	dn := gui.Config.SceneScanSkip
	if gui.State.Archive.Len()-1-gui.State.ArchivePos <= dn {
		dn = 1
	}

	for n := gui.State.ArchivePos + 1; n < gui.State.Archive.Len(); n += dn {
		h, ok := gui.ImageHash(n, nil)
		if !ok {
			return
		}
		distance := float32(imgdiff.Distance(imgdiff.Hash(hash), h)) / 64

		if distance > gui.Config.ImageDiffThres {
			if dn == 1 || n == gui.State.ArchivePos+1 {
				gui.setPage(n)
				return
			}

			for l := n - 1; l >= gui.State.ArchivePos+1; l-- {
				h, ok := gui.ImageHash(l, nil)
				if !ok {
					return
				}
				d := float32(imgdiff.Distance(imgdiff.Hash(hash), h)) / 64
				if d <= gui.Config.ImageDiffThres {
					gui.setPage(l + 1)
					return
				}
			}
			return
		}
	}
}

func (gui *GUI) PreviousScene() {
	if !gui.Loaded() {
		return
	}

	if gui.State.QImageL == nil {
		return
	}
	hash := imgdiff.DHash(gui.State.QImageL)

	dn := gui.Config.SceneScanSkip
	if gui.State.ArchivePos <= dn {
		dn = 1
	}

	for n := gui.State.ArchivePos - 1; n >= 0; n -= dn {
		h, ok := gui.ImageHash(n, nil)
		if !ok {
			return
		}
		distance := float32(imgdiff.Distance(imgdiff.Hash(hash), h)) / 64

		if distance > gui.Config.ImageDiffThres {
			if dn == 1 || n == gui.State.ArchivePos-1 {
				gui.setPage(n)
				return
			}

			for l := n + 1; l <= gui.State.ArchivePos-1; l++ {
				h, ok := gui.ImageHash(l, nil)
				if !ok {
					return
				}
				d := float32(imgdiff.Distance(imgdiff.Hash(hash), h)) / 64
				if d <= gui.Config.ImageDiffThres {
					gui.setPage(l - 1)
					return
				}
			}
			return
		}
	}
}

func (gui *GUI) NextArchive() bool {
	newname, err := gui.archiveNameRel(1)
	if err != nil {
		return false
	}

	gui.LoadArchive(newname, false)
	return true
}

func (gui *GUI) PreviousArchive() bool {
	newname, err := gui.archiveNameRel(-1)
	if err != nil {
		return false
	}

	gui.LoadArchive(newname, false)
	gui.LastPage()
	return true
}

func (gui *GUI) curArchive() (which int, err error) {
	dir, name := filepath.Split(gui.State.ArchivePath)
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return
		}
	}
	anames, err := archive.ListArchives(dir)
	if err != nil {
		return
	}

	which = -1
	for i := 0; i < len(anames); i++ {
		if anames[i] == name {
			which = i
		}
	}
	if which == -1 {
		return 0, ErrCurrentNotFound
	}
	return
}

func (gui *GUI) archiveNameRel(i int) (newname string, err error) {
	dir, _ := filepath.Split(gui.State.ArchivePath)
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return
		}
	}
	anames, err := archive.ListArchives(dir)
	if err != nil {
		return
	}

	curarch, err := gui.curArchive()
	if err != nil {
		return "", err
	}

	which := curarch + i
	if which < 0 || which >= len(anames) {
		err = errors.New("No more archives in the directory")
		return
	}

	newname = filepath.Join(dir, anames[which])
	return
}

func (gui *GUI) SkipForward() {
	gui.SetPage(gui.State.ArchivePos + gui.Config.NSkip)
}

func (gui *GUI) SkipBackward() {
	gui.SetPage(gui.State.ArchivePos - gui.Config.NSkip)
}
