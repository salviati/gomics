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

package imgdiff

import (
	"github.com/mappu/miqt/qt6"
	"math/bits"
	"runtime"
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

// http://www.hackerfactor.com/blog/?/archives/529-Kind-of-Like-That.html
func DHash(p *qt6.QImage) Hash {
	q := p.Scaled(dhashImageWidth, dhashImageHeight)
	defer func() {
		runtime.SetFinalizer(q, nil)
		q.Delete()
	}()

	data := make([]byte, dhashImageWidth*dhashImageHeight, dhashImageWidth*dhashImageHeight)
	for iy := 0; iy < dhashImageHeight; iy++ {
		for ix := 0; ix < dhashImageWidth; ix++ {
			v := q.Pixel(ix, iy)
			r := (v >> 16) & 0xff
			g := (v >> 8) & 0xff
			b := v & 0xff
			data[iy*dhashImageWidth+ix] = byte((19595*r + 38470*g + 7471*b + 1<<15) >> 16)
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
