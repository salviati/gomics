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

package main

import (
	"flag"
	"fmt"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/salviati/gomics/archive"
	"github.com/salviati/gomics/imgdiff"
	"log"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

type State struct {
	Archive                 archive.Archive
	ArchivePos              int
	ArchivePath             string
	ArchiveName             string
	PixbufL, PixbufR        *gdk.Pixbuf
	GoToThumnailPixbuf      *gdk.Pixbuf
	DeltaW, DeltaH          int
	Scale                   float64
	UserHome                string
	ConfigPath              string
	ImageHash               map[int]imgdiff.Hash
	CursorLastMoved         time.Time
	CursorHidden            bool
	CursorForceShown        bool
	BackgroundStyleProvider *gtk.CssProvider
}

func (gui *GUI) SetStatus(msg string) {
	context_id := gui.Statusbar.GetContextId("main")
	gui.Statusbar.Push(context_id, msg)
}

func (gui *GUI) ResizeEvent() {
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) ShowError(msg string) {
	log.Println(msg)
	gui.SetStatus(msg)
}

func (gui *GUI) Close() {
	if !gui.Loaded() {
		return
	}

	gui.State.Archive.Close()

	gui.State.Archive = nil
	gui.State.ArchiveName = ""
	gui.State.ArchivePath = ""
	gui.State.ArchivePos = 0

	gui.State.ImageHash = nil

	gui.ImageL.Clear()
	gui.ImageR.Clear()
	gui.State.PixbufL = nil
	gui.State.PixbufR = nil
	gui.State.CursorLastMoved = time.Now()
	gui.State.CursorHidden = false
	gui.State.CursorForceShown = false
	gui.SetStatus("")
	gui.MainWindow.SetTitle("Gomics")
	gc()
}

func (gui *GUI) LoadArchive(path string) {
	// TODO(utkan): non-local (http:// or https://) stuff someday?

	if strings.TrimSpace(path) == "" {
		return
	}

	if filepath.IsAbs(path) == false {
		wd, err := os.Getwd()
		if err != nil {
			log.Println(err)
			return
		}
		path = filepath.Join(wd, path)
	}

	if gui.Loaded() {
		gui.Close()
	}

	gui.State.ImageHash = make(map[int]imgdiff.Hash)

	gui.State.ArchivePath = path
	gui.State.ArchiveName = filepath.Base(path)

	var err error
	if gui.State.Archive, err = archive.NewArchive(path); err != nil {
		gui.ShowError("Failed to open " + path + ": " + err.Error())
		return
	}

	gui.setPage(0) // FIXME(utkan): this might fail.
	os.Chdir(gui.State.ArchivePath)

	u := &url.URL{Path: path, Scheme: "file"}

	ok := gui.RecentManager.AddItem(u.String())
	if !ok {
		log.Println("Failed to add", path, "as a recent item")
	}
}

func (gui *GUI) LoadImage(n int) (*gdk.Pixbuf, error) {
	ar := gui.State.Archive
	pixbuf, err := ar.Load(n, gui.Config.EmbeddedOrientation)

	if err != nil {
		filename, _ := ar.Name(n)
		gui.ShowError(fmt.Sprintf(`Failed to load file #%d "%s": %s`, n+1, filename, err.Error()))
		return nil, err
	}

	gui.ImageHash(n, pixbuf)
	return pixbuf, nil
}

func (gui *GUI) SetPage(n int) {
	if !gui.Loaded() {
		return
	}

	if n < 0 {
		n = 0
	}

	if n >= gui.State.Archive.Len() {
		n = gui.State.Archive.Len() - 1
	}

	if n == gui.State.ArchivePos {
		return
	}

	gui.setPage(n)
}

func (gui *GUI) setPage(n int) {
	if !gui.Loaded() {
		return
	}

	gui.State.ArchivePos = n
	gui.State.PixbufR = nil
	// TODO clear images in the UI on error

	var err error
	if gui.State.PixbufL, err = gui.LoadImage(n); err != nil {
		//return
	}

	gui.State.PixbufR = nil
	if gui.Config.DoublePage && n+1 < gui.State.Archive.Len() {
		if gui.State.PixbufR, err = gui.LoadImage(n + 1); err != nil {
			//return
		}
	}

	gc()

	gui.Blit()
	gui.StatusImage()

	gui.scrollToTop()
}

func (gui *GUI) Scroll(dx, dy float64) {
	if !gui.Loaded() {
		return
	}

	imgw, imgh := gui.GetSize()

	vadj := gui.ScrolledWindow.GetVAdjustment()
	hadj := gui.ScrolledWindow.GetHAdjustment()

	vdx := vadj.GetMinimumIncrement()
	vval := vadj.GetValue()
	vupper := vadj.GetUpper() - float64(imgh) - 4
	vlower := vadj.GetLower()

	hdx := hadj.GetMinimumIncrement()
	hval := hadj.GetValue()
	hupper := hadj.GetUpper() - float64(imgw) - 4
	hlower := hadj.GetLower()

	if dy > 0 {
		if vval >= vupper {
			if gui.Config.SmartScroll {
				gui.NextPage()
			}
		} else {
			vadj.SetValue(clamp(vval+vdx, vlower, vupper))
			gui.ScrolledWindow.SetVAdjustment(vadj)
		}
	} else if dy < 0 {
		if vval <= vlower {
			if gui.Config.SmartScroll {
				gui.PreviousPage()
			}
		} else {
			vadj.SetValue(clamp(vval-vdx, vlower, vupper))
			gui.ScrolledWindow.SetVAdjustment(vadj)
		}
	}

	if dx > 0 {
		if hval >= hupper {
			if gui.Config.SmartScroll {
				// TODO scroll down a bit
			}
		} else {
			hadj.SetValue(clamp(hval+hdx, hlower, hupper))
			gui.ScrolledWindow.SetHAdjustment(hadj)
		}
	} else if dx < 0 {
		if hval <= hlower {
			if gui.Config.SmartScroll {
				// TODO scroll up a bit
			}
		} else {
			hadj.SetValue(clamp(hval-hdx, hlower, hupper))
			gui.ScrolledWindow.SetHAdjustment(hadj)
		}
	}
}

func (gui *GUI) scrollToTop() {
	if !gui.Loaded() {
		return
	}

	vadj := gui.ScrolledWindow.GetVAdjustment()
	vadj.SetValue(0)
	gui.ScrolledWindow.SetVAdjustment(vadj)

	hadj := gui.ScrolledWindow.GetHAdjustment()
	hadj.SetValue(0)
	gui.ScrolledWindow.SetHAdjustment(hadj)
}

/*
func (gui *GUI) scroll(int dx, int dy) {
	gui.ScrolledWindow.GetVAdjustment()
	vadj.SetValue(0)
}
*/

func (gui *GUI) Quit() {
	gui.Config.WindowWidth, gui.Config.WindowHeight = gui.MainWindow.GetSize()

	if err := gui.Config.Save(filepath.Join(gui.State.ConfigPath, ConfigFile)); err != nil {
		log.Println(err)
	}
	gtk.MainQuit()
}

func (gui *GUI) Init() {
	// Load configuration
	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	gui.State.UserHome = u.HomeDir
	gui.State.ConfigPath = filepath.Join(u.HomeDir, ConfigDir)

	gui.Config.Defaults()
	gui.Config.LastDirectory = gui.State.UserHome

	if err := os.MkdirAll(gui.State.ConfigPath, 0755); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(gui.State.ConfigPath, ImageDir), 0755); err != nil {
		log.Fatal(err)
	}

	if err := gui.Config.Load(filepath.Join(gui.State.ConfigPath, ConfigFile)); err != nil {
		if os.IsNotExist(err) == false {
			log.Fatal(err)
		}
	}

	gui.RecentManager, err = gtk.RecentManagerGetDefault()
	if err != nil {
		log.Fatal(err)
	}

	gui.initUI()
}

func (gui *GUI) SetFullscreen(fullscreen bool) {
	gui.Config.Fullscreen = fullscreen
	if fullscreen {
		gui.Statusbar.Hide()
		gui.Toolbar.Hide()
		gui.Menubar.Hide() // BUG: menubar visible on fullscreen
		gui.MainWindow.Fullscreen()
	} else {
		gui.Statusbar.Show()
		gui.Toolbar.Show()
		gui.Menubar.Show()
		gui.MainWindow.Unfullscreen()
	}
	gui.MenuItemFullscreen.SetActive(fullscreen)
}

func (gui *GUI) SetShrink(shrink bool) {
	gui.Config.Shrink = shrink
	gui.MenuItemShrink.SetActive(shrink)
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) SetEnlarge(enlarge bool) {
	gui.Config.Enlarge = enlarge
	gui.MenuItemEnlarge.SetActive(enlarge)
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) SetRandom(random bool) {
	gui.Config.Random = random
	gui.MenuItemRandom.SetActive(random)
}

func (gui *GUI) SetSeamless(seamless bool) {
	gui.Config.Seamless = seamless
	gui.MenuItemSeamless.SetActive(seamless)
}

func (gui *GUI) SetHFlip(hflip bool) {
	gui.Config.HFlip = hflip
	gui.Blit()
}

func (gui *GUI) SetVFlip(vflip bool) {
	gui.Config.VFlip = vflip
	gui.Blit()
}

func (gui *GUI) SavePNG() {
	if !gui.Loaded() {
		return
	}

	base := filepath.Base(gui.State.ArchivePath)
	if ext := filepath.Ext(base); len(ext) > 1 {
		base = strings.TrimSuffix(base, ext)
	}

	pngBase := fmt.Sprintf("%s-%000d.png", base, gui.State.ArchivePos+1)
	pngPath := filepath.Join(gui.State.ConfigPath, ImageDir, pngBase)
	if err := gui.State.PixbufL.SavePNG(pngPath, PNGCompressionLevel); err != nil {
		gui.ShowError(err.Error())
		return
	}

	// TODO: save PixbufR as well when two pages are displayed
	// TODO: save original file without conversion

	gui.SetStatus("Saved to " + pngBase)
}

func (gui *GUI) SetZoomMode(mode string) {
	switch mode {
	case "FitToWidth":
		gui.MenuItemFitToWidth.SetActive(true)
	case "FitToHeight":
		gui.MenuItemFitToHeight.SetActive(true)
	case "BestFit":
		gui.MenuItemBestFit.SetActive(true)
	default:
		gui.MenuItemOriginal.SetActive(true)
		mode = "Original"
	}

	gui.Config.ZoomMode = mode
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) SetDoublePage(doublePage bool) {
	gui.ImageR.SetVisible(doublePage)
	gui.Config.DoublePage = doublePage
	// TODO set alignment of ImageL to 0.5 or 1
	gui.setPage(gui.State.ArchivePos)
	//gui.ImageR.SetVisible(doublePage)
}

func (gui *GUI) SetMangaMode(mangaMode bool) {
	gui.Config.MangaMode = mangaMode
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) SetInterpolation(interpolation int) {
	gui.Config.Interpolation = interpolation
	gui.Blit()
}

func (gui *GUI) SetOneWide(oneWide bool) {
	gui.Config.OneWide = oneWide
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) SetSmartScroll(smartScroll bool) {
	gui.Config.SmartScroll = smartScroll
}

func (gui *GUI) SetHideIdleCursor(hideIdleCursor bool) {
	gui.Config.HideIdleCursor = hideIdleCursor
}

func (gui *GUI) SetEmbeddedOrientation(embeddedOrientation bool) {
	gui.Config.EmbeddedOrientation = embeddedOrientation
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) fixFocus() {
	//gui.ScrollWindowed.GrabFocus() // FIXME
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

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

	gtk.Init(nil)
	gui := new(GUI)
	gui.Init()

	if flag.NArg() > 0 {
		gui.LoadArchive(flag.Arg(0))
	}

	gtk.Main()

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}
}
