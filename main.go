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
	"flag"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/mappu/miqt/qt6"
	"github.com/salviati/gomics/archive"
	"github.com/salviati/gomics/imgdiff"
)

type State struct {
	Archive          archive.Archive
	ArchivePos       int
	ArchivePath      string
	ArchiveName      string
	QImageL, QImageR *qt6.QImage
	GoToThumbnail    *qt6.QImage
	DeltaW, DeltaH   int
	Scale            float64
	UserHome         string
	ConfigPath       string
	ImageHash        map[int]imgdiff.Hash
	CursorLastMoved  time.Time
	CursorHidden     bool
	CursorForceShown bool
}

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	qt6.NewQApplication(os.Args)
	gui := new(GUI)
	gui.Init()

	if flag.NArg() > 0 {
		gui.LoadArchive(flag.Arg(0), true)
	}

	qt6.QApplication_Exec()

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}
}
