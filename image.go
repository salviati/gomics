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
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/mappu/miqt/qt6"
)

func (gui *GUI) pixbufLoaded() bool {
	if gui.Config.DoublePage && gui.forceSinglePage() == false {
		return gui.State.QImageL != nil && gui.State.QImageR != nil
	}
	return gui.State.QImageL != nil
}

func (gui *GUI) pixbufSize() (w, h int) {
	if !gui.pixbufLoaded() {
		return 0, 0
	}

	s := &gui.State

	if gui.Config.DoublePage && gui.forceSinglePage() == false {
		return s.QImageL.Width() + s.QImageR.Width(), max(s.QImageL.Height(), s.QImageR.Height())
	}
	return s.QImageL.Width(), s.QImageL.Height()
}

func (gui *GUI) rotatedDims(w, h int) (rw, rh int) {
	switch gui.Config.Rotation % 360 {
	case 90, 270:
		return h, w
	default:
		return w, h
	}
}

// renderedSize is the on-screen content size after applying the current
// rotation to each loaded image (rotation swaps width/height at 90/270).
func (gui *GUI) renderedSize() (w, h int) {
	if !gui.pixbufLoaded() {
		return 0, 0
	}

	s := &gui.State

	if gui.Config.DoublePage && gui.forceSinglePage() == false {
		lw, lh := s.QImageL.Width(), s.QImageL.Height()
		rw, rh := s.QImageR.Width(), s.QImageR.Height()
		l2w, l2h := gui.rotatedDims(lw, lh)
		r2w, r2h := gui.rotatedDims(rw, rh)
		return l2w + r2w, max(l2h, r2h)
	}

	w, h = s.QImageL.Width(), s.QImageL.Height()
	return gui.rotatedDims(w, h)
}

func (gui *GUI) StatusImage() {
	s := &gui.State

	if !gui.pixbufLoaded() {
		return
	}

	zoom := int(100 * gui.State.Scale)

	var msg, title string
	if gui.Config.DoublePage && gui.forceSinglePage() == false {
		leftPath, _ := s.Archive.Name(s.ArchivePos)
		left := filepath.Base(leftPath)
		rightPath, _ := s.Archive.Name(s.ArchivePos + 1)
		right := filepath.Base(rightPath)

		leftIndex := s.ArchivePos + 1
		rightIndex := s.ArchivePos + 2

		leftw, lefth := s.QImageL.Width(), s.QImageL.Height()
		rightw, righth := s.QImageR.Width(), s.QImageR.Height()

		if gui.Config.MangaMode {
			left, right = right, left
			leftIndex, rightIndex = rightIndex, leftIndex
			leftw, rightw = rightw, leftw
		}
		msg = fmt.Sprintf("%d,%d/%d   |   %s (%dx%d) - %s (%dx%d)   |   %d%%", leftIndex, rightIndex, s.Archive.Len(), left, leftw, lefth, right, rightw, righth, zoom)
		title = fmt.Sprintf("[%d,%d / %d] %s", leftIndex, rightIndex, s.Archive.Len(), s.ArchiveName)
	} else {
		imgPath, _ := s.Archive.Name(s.ArchivePos)
		w, h := s.QImageL.Width(), s.QImageL.Height()
		msg = fmt.Sprintf("%d/%d   |   %s (%dx%d)   |   %d%%  ", s.ArchivePos+1, s.Archive.Len(), imgPath, w, h, zoom)
		title = fmt.Sprintf("[%d / %d] %s", s.ArchivePos+1, s.Archive.Len(), s.ArchiveName)
	}
	gui.SetStatus(strings.Split(msg, "   |   ")...)

	gui.MainWindow.SetWindowTitle(title)
}

func (gui *GUI) GetSize() (width, height int) {
	// The inner viewport is the real paintable area: it shrinks when a
	// scrollbar appears, so fitting to it keeps the fit dimension exact and
	// avoids both blank bars and spurious scrollbars.
	alloc := gui.Image.Viewport().Geometry()
	return alloc.Width(), alloc.Height()
}

func (gui *GUI) ScaledSize() (scale float64) {
	if !gui.pixbufLoaded() {
		return
	}

	scrw, scrh := gui.GetSize()

	// Scale purely from the zoom mode so the mode-dictated dimension fills
	// the viewport exactly (fit-to-width -> full width, fit-to-height ->
	// full height, best-fit -> both fit). No scrollbar-space reservation:
	// that shrank the dictated dimension and left blank padding.
	return gui.scaledSize(scrw, scrh)
}

func (gui *GUI) scaledSize(scrw, scrh int) (scale float64) {
	w, h := gui.renderedSize()
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
			// Scale to the tight dimension's exact ratio so that dimension fills
			// the viewport exactly; fit() rounds and left a 1-2px edge gap.
			r := float64(w) / float64(h)
			if float64(scrw) >= float64(scrh)*r {
				return float64(scrh) / float64(h) // height is the tight dimension
			}
			return float64(scrw) / float64(w) // width is the tight dimension
		}
	case "Manual":
		// Fixed percentage, independent of Enlarge/Shrink and viewport.
		return float64(gui.Config.ZoomLevel) / 100.0
	}
	return 1
}

func (gui *GUI) forceSinglePage() bool {
	if gui.State.QImageR == nil {
		return true
	}

	return gui.Config.OneWide == true && (gui.State.QImageL.Width() > gui.State.QImageL.Height() || gui.State.QImageR.Width() > gui.State.QImageR.Height())
}

func (gui *GUI) Blit() {
	if !gui.pixbufLoaded() {
		// Keep the scene rect pinned to the viewport when nothing is loaded,
		// so a window shrink can't leave the stale oversized rect behind
		// and Qt show scrollbars for an empty scene.
		vw, vh := gui.GetSize()
		gui.Image.SetSceneRect2(0, 0, float64(vw), float64(vh))
		return
	}

	gui.State.Scale = gui.ScaledSize()

	// Clear scene
	gui.Image.Scene().Clear()

	vw, vh := gui.GetSize()
	rw, rh := gui.renderedSize()
	sw, sh := int(math.Round(float64(rw)*gui.State.Scale)), int(math.Round(float64(rh)*gui.State.Scale))
	offx, offy := max(0, (vw-sw)/2), max(0, (vh-sh)/2)

	if gui.Config.DoublePage && gui.forceSinglePage() == false {
		left := gui.State.QImageL
		right := gui.State.QImageR

		if gui.Config.MangaMode {
			left, right = right, left
		}

		lrw, _ := gui.rotatedDims(left.Width(), left.Height())
		gui.blitImage(left, gui.State.Scale, offx, offy)
		gui.blitImage(right, gui.State.Scale, offx+int(float64(lrw)*gui.State.Scale), offy)
	} else {
		gui.blitImage(gui.State.QImageL, gui.State.Scale, offx, offy)
	}

	// Scene rect spans the larger of content and viewport in each dimension, so
	// QGraphicsView never centers it (it only centers a scene rect smaller than
	// the viewport) and there's no widget-vs-viewport offset. The item's
	// offx/offy do the single centering: centered when the content fits, at
	// the origin when it overflows (then the scene rect == content for scroll).
	sceneW, sceneH := max(sw, vw), max(sh, vh)
	gui.Image.SetSceneRect2(0, 0, float64(sceneW), float64(sceneH))

	// Full-res pixmaps are created per Blit and miqt frees them via
	// runtime.SetFinalizer, so force a collection after every Blit that
	// adds items to bound peak memory.
	gc()
}

func blitTransform(w, h int, scale float64, rotation int, hflip, vflip bool) *qt6.QTransform {
	t := qt6.NewQTransform2()
	rot := rotation % 360

	if scale != 1.0 {
		t.Scale(scale, scale)
	}

	if hflip {
		t.Translate(float64(w), 0)
		t.Scale(-1, 1)
	}
	if vflip {
		t.Translate(0, float64(h))
		t.Scale(1, -1)
	}

	if rot != 0 {
		cx := float64(w) / 2.0
		cy := float64(h) / 2.0

		if rot == 90 || rot == 270 {
			t.Translate(float64(h)/2.0, float64(w)/2.0)
			t.Rotate(float64(rot))
			t.Translate(-cx, -cy)
		} else {
			t.Translate(cx, cy)
			t.Rotate(float64(rot))
			t.Translate(-cx, -cy)
		}
	}

	return t
}

func (gui *GUI) blitImage(img *qt6.QImage, scale float64, x, y int) {
	if img == nil {
		return
	}

	pix := qt6.QPixmap_FromImage(img)
	item := gui.Image.Scene().AddPixmap(pix)
	item.SetPos2(float64(x), float64(y))

	mode := qt6.FastTransformation
	if gui.Config.Interpolation == 1 {
		mode = qt6.SmoothTransformation
	}
	item.SetTransformationMode(mode)

	t := blitTransform(img.Width(), img.Height(), scale, gui.Config.Rotation, gui.Config.HFlip, gui.Config.VFlip)
	item.SetTransform(t)
}
