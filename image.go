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

package main

import (
	"fmt"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"path/filepath"
)

var interpolations = []gdk.InterpType{gdk.INTERP_NEAREST, gdk.INTERP_TILES, gdk.INTERP_BILINEAR, gdk.INTERP_HYPER}

func (gui *GUI) pixbufLoaded() bool {
	if gui.Config.DoublePage && gui.forceSinglePage() == false {
		return gui.State.PixbufL != nil && gui.State.PixbufR != nil
	}
	return gui.State.PixbufL != nil
}

func (gui *GUI) pixbufSize() (w, h int) {
	if !gui.pixbufLoaded() {
		return 0, 0
	}

	s := &gui.State

	if gui.Config.DoublePage && gui.forceSinglePage() == false {
		return s.PixbufL.GetWidth() + s.PixbufR.GetWidth(), max(s.PixbufL.GetHeight(), s.PixbufR.GetHeight())
	}
	return s.PixbufL.GetWidth(), s.PixbufL.GetHeight()
}

func (gui *GUI) StatusImage() {
	s := &gui.State

	if !gui.pixbufLoaded() {
		return
	}

	// TODO

	zoom := int(100 * gui.State.Scale)

	var msg, title string
	if gui.Config.DoublePage && gui.forceSinglePage() == false {
		leftPath, _ := s.Archive.Name(s.ArchivePos)
		left := filepath.Base(leftPath)
		rightPath, _ := s.Archive.Name(s.ArchivePos + 1)
		right := filepath.Base(rightPath)

		leftIndex := s.ArchivePos + 1
		rightIndex := s.ArchivePos + 2

		leftw, lefth := s.PixbufL.GetWidth(), s.PixbufL.GetHeight()
		rightw, righth := s.PixbufR.GetWidth(), s.PixbufR.GetHeight()

		if gui.Config.MangaMode {
			left, right = right, left
			leftIndex, rightIndex = rightIndex, leftIndex
			leftw, rightw = rightw, leftw
		}
		msg = fmt.Sprintf("%d,%d / %d   |   %dx%d - %dx%d (%d%%)   |   %s   |   %s - %s", leftIndex, rightIndex, s.Archive.Len(), leftw, lefth, rightw, righth, zoom, s.ArchiveName, left, right)
		title = fmt.Sprintf("[%d,%d / %d] %s", leftIndex, rightIndex, s.Archive.Len(), s.ArchiveName)
	} else {
		imgPath, _ := s.Archive.Name(s.ArchivePos)
		w, h := s.PixbufL.GetWidth(), s.PixbufL.GetHeight()
		msg = fmt.Sprintf("(%d/%d)   |   %dx%d (%d%%)   |   %s   |   %s", s.ArchivePos+1, s.Archive.Len(), w, h, zoom, s.ArchiveName, imgPath)
		title = fmt.Sprintf("[%d / %d] %s", s.ArchivePos+1, s.Archive.Len(), s.ArchiveName)
	}
	gui.SetStatus(msg)

	gui.MainWindow.SetTitle(title)
}

func (gui *GUI) GetSize() (width, height int) {
	/*child, _ := gui.ScrolledWindow.GetChild()
	alloc := child.GetAllocation()
	return alloc.GetWidth(), alloc.GetHeight()*/

	alloc := gui.ScrolledWindow.GetAllocation()
	return alloc.GetWidth() - 4, alloc.GetHeight() - 4
}

func (gui *GUI) ScaledSize() (scale float64) {
	if !gui.pixbufLoaded() {
		return
	}

	scrw, scrh := gui.GetSize()

	// BUG: if, for instance image is taller than window width, we should subtract scrollbar width here!
	// but also consider that scrollbar might disappear when do substract!

	scale = gui.scaledSize(scrw, scrh)
	w, h := gui.pixbufSize()
	sw, sh := int(scale*float64(w)), int(scale*float64(h))
	hscroll := sw > scrw
	vscroll := sh > scrh

	if !hscroll && !vscroll {
		return
	}

	// still buggy at "transition" point

	if hscroll {
		scrh -= 16 // scrollbar size // BUG WTF?!
	}
	if vscroll {
		scrw -= 16
	}

	return gui.scaledSize(scrw, scrh)
}

func (gui *GUI) scaledSize(scrw, scrh int) (scale float64) {
	w, h := gui.pixbufSize()
	switch gui.Config.ZoomMode {
	case "FitToWidth":
		needscale := (gui.Config.Enlarge && w < scrw) || (gui.Config.Shrink && w > scrw)
		if needscale {
			return float64(scrw) / float64(w)
		}
	case "FitToHeight":
		return float64(scrh) / float64(h)
	case "BestFit":
		needscale := (gui.Config.Enlarge && (w < scrw && h < scrh)) || (gui.Config.Shrink && (w > scrw || h > scrh))
		if needscale {
			fw, _ := fit(w, h, scrw, scrh)
			return float64(fw) / float64(w)
		}
	}
	// original, others
	return 1
}

func (gui *GUI) forceSinglePage() bool {
	if gui.State.PixbufR == nil {
		return true
	}

	return gui.Config.OneWide == true && (gui.State.PixbufL.GetWidth() > gui.State.PixbufL.GetHeight() || gui.State.PixbufR.GetWidth() > gui.State.PixbufR.GetHeight())
}

func (gui *GUI) Blit() {
	if !gui.pixbufLoaded() {
		return
	}

	gui.State.Scale = gui.ScaledSize()

	// Check whether the scale of the left image is different from the old one?

	if gui.Config.DoublePage && gui.forceSinglePage() == false {
		left := gui.State.PixbufL
		right := gui.State.PixbufR

		if gui.Config.MangaMode {
			left, right = right, left
		}

		if err := gui.blit(gui.ImageL, left, gui.State.Scale); err != nil {
			gui.ShowError(err.Error())
			return
		}

		if err := gui.blit(gui.ImageR, right, gui.State.Scale); err != nil {
			gui.ShowError(err.Error())
			return
		}
	} else {
		gui.ImageR.Clear()
		if err := gui.blit(gui.ImageL, gui.State.PixbufL, gui.State.Scale); err != nil {
			gui.ShowError(err.Error())
			return
		}
	}

	if gui.State.Scale != 1 || gui.Config.HFlip || gui.Config.VFlip {
		gc()
	}
}

func (gui *GUI) blit(image *gtk.Image, pixbuf *gdk.Pixbuf, scale float64) (err error) {
	image.Clear()

	if gui.Config.HFlip {
		pixbuf, err = pixbuf.Flip(true)
		if err != nil {
			return err
		}
	}

	if gui.Config.VFlip {
		pixbuf, err = pixbuf.Flip(false)
		if err != nil {
			return err
		}
	}

	if scale != 1 {
		w, h := pixbuf.GetWidth(), pixbuf.GetHeight()
		pixbuf, err = pixbuf.ScaleSimple(int(float64(w)*scale), int(float64(h)*scale), interpolations[gui.Config.Interpolation])
		if err != nil {
			return err
		}
	}

	image.SetFromPixbuf(pixbuf)

	return nil
}
