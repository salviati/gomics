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

package imgdiff

import (
	"github.com/gotk3/gotk3/gdk"
	//"github.com/nfnt/resize"
	"image"
	"image/color"
	"math/bits"
	//"github.com/disintegration/imaging"
)

const (
	dhashImageWidth  = 9
	dhashImageHeight = 8
)

type Hash uint64 // assuming (dhashImageWidth-1)*dhashImageHeight <= 64

func Distance(h1, h2 Hash) int {
	return bits.OnesCount64(uint64(h1 ^ h2))
}

func init() {
	if dhashImageHeight*(dhashImageWidth-1) > 64 {
		panic("dhashImageHeight is too large")
	}
}

// type grayscaleImage {
// 	int w
// 	int h
// 	data []byte
// }
//
// func newGrayscaleImage(int w, h) grayscaleImage {
// 	return grayscaleImage{
// 		w: w,
// 		h: h,
// 		data: make([]byte, w*h, w*h)
// 	}
// }

func pixbufToGrayscaleImage(p *gdk.Pixbuf) *image.Gray {
	nchan := p.GetNChannels()
	data := p.GetPixels()
	w, h := p.GetWidth(), p.GetHeight()
	rowstride := p.GetRowstride()
	im := image.NewGray(image.Rect(0, 0, w, h))

	if nchan == 1 {
		for ih := 0; ih < h; ih++ {
			for iw := 0; iw < w; iw++ {
				y := data[ih*rowstride+iw*nchan]
				im.SetGray(iw, ih, color.Gray{Y: y})
			}
		}
	} else if nchan == 3 || nchan == 4 {
		for ih := 0; ih < h; ih++ {
			for iw := 0; iw < w; iw++ {
				r := data[ih*rowstride+iw*nchan]
				g := data[ih*rowstride+iw*nchan+1]
				b := data[ih*rowstride+iw*nchan+2]
				y := uint8((19595*uint32(r) + 38470*uint32(g) + 7471*uint32(b) + 1<<15) >> 16)
				im.SetGray(iw, ih, color.Gray{Y: y})
			}
		}
	} else {
		panic("unknown image depth")
	}

	return im
}

// http://www.hackerfactor.com/blog/?/archives/529-Kind-of-Like-That.html
func DHash(p *gdk.Pixbuf) Hash {
	//im := resize.Resize(dhashImageWidth, dhashImageHeight, pixbufToGrayscaleImage(p), resize.Bilinear)
	//gray := im.(*image.Gray)

	//im := imaging.Resize(pixbufToGrayscaleImage(p), dhashImageWidth, dhashImageHeight, imaging.Linear)

	q, err := p.ScaleSimple(dhashImageWidth, dhashImageHeight, gdk.INTERP_TILES)
	if err != nil {
		panic(err.Error())
	}
	gray := pixbufToGrayscaleImage(q)

	data := make([]byte, dhashImageWidth*dhashImageHeight, dhashImageWidth*dhashImageHeight)
	for iy := 0; iy < dhashImageHeight; iy++ {
		for ix := 0; ix < dhashImageWidth; ix++ {
			data[iy*dhashImageWidth+ix] = gray.GrayAt(ix, iy).Y
		}
	}

	var hash Hash

	for iy := 0; iy < dhashImageHeight; iy++ {
		for ix := 0; ix < dhashImageWidth-1; ix++ {
			o := iy * dhashImageWidth
			if data[o+ix+1] > data[o+ix] {
				hash |= 1 << uint(iy*dhashImageHeight+ix)
			}
		}
	}

	return hash

}
