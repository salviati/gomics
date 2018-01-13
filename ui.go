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

//go:generate go-bindata about.jpg icon.png gomics.glade

package main

import (
	"fmt"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"log"
	"reflect"
	"runtime"
)

type GUI struct {
	MainWindow                     *gtk.Window            `build:"MainWindow"`
	VBox                           *gtk.Box               `build:"VBox"`
	Menubar                        *gtk.MenuBar           `build:"Menubar"`
	ScrolledWindow                 *gtk.ScrolledWindow    `build:"ScrolledWindow"`
	Viewport                       *gtk.Viewport          `build:"Viewport"`
	ImageBox                       *gtk.Box               `build:"ImageBox"`
	ImageL                         *gtk.Image             `build:"ImageL"`
	ImageR                         *gtk.Image             `build:"ImageR"`
	Statusbar                      *gtk.Statusbar         `build:"Statusbar"`
	AboutDialog                    *gtk.AboutDialog       `build:"AboutDialog"`
	MenuItemAbout                  *gtk.MenuItem          `build:"MenuItemAbout"`
	MenuItemOpen                   *gtk.MenuItem          `build:"MenuItemOpen"`
	MenuItemClose                  *gtk.MenuItem          `build:"MenuItemClose"`
	MenuItemQuit                   *gtk.MenuItem          `build:"MenuItemQuit"`
	MenuItemSaveImage              *gtk.MenuItem          `build:"MenuItemSaveImage"`
	FileChooserDialogArchive       *gtk.FileChooserDialog `build:"FileChooserDialogArchive"`
	Toolbar                        *gtk.Toolbar           `build:"Toolbar"`
	ButtonNextPage                 *gtk.ToolButton        `build:"ButtonNextPage"`
	ButtonPreviousPage             *gtk.ToolButton        `build:"ButtonPreviousPage"`
	ButtonLastPage                 *gtk.ToolButton        `build:"ButtonLastPage"`
	ButtonFirstPage                *gtk.ToolButton        `build:"ButtonFirstPage"`
	ButtonNextArchive              *gtk.ToolButton        `build:"ButtonNextArchive"`
	ButtonPreviousArchive          *gtk.ToolButton        `build:"ButtonPreviousArchive"`
	ButtonNextScene                *gtk.ToolButton        `build:"ButtonNextScene"`
	ButtonPreviousScene            *gtk.ToolButton        `build:"ButtonPreviousScene"`
	ButtonSkipForward              *gtk.ToolButton        `build:"ButtonSkipForward"`
	ButtonSkipBackward             *gtk.ToolButton        `build:"ButtonSkipBackward"`
	MenuItemNextPage               *gtk.MenuItem          `build:"MenuItemNextPage"`
	MenuItemPreviousPage           *gtk.MenuItem          `build:"MenuItemPreviousPage"`
	MenuItemLastPage               *gtk.MenuItem          `build:"MenuItemLastPage"`
	MenuItemFirstPage              *gtk.MenuItem          `build:"MenuItemFirstPage"`
	MenuItemNextArchive            *gtk.MenuItem          `build:"MenuItemNextArchive"`
	MenuItemPreviousArchive        *gtk.MenuItem          `build:"MenuItemPreviousArchive"`
	MenuItemSkipForward            *gtk.MenuItem          `build:"MenuItemSkipForward"`
	MenuItemSkipBackward           *gtk.MenuItem          `build:"MenuItemSkipBackward"`
	MenuItemEnlarge                *gtk.CheckMenuItem     `build:"MenuItemEnlarge"`
	MenuItemShrink                 *gtk.CheckMenuItem     `build:"MenuItemShrink"`
	MenuItemFullscreen             *gtk.CheckMenuItem     `build:"MenuItemFullscreen"`
	MenuItemSeamless               *gtk.CheckMenuItem     `build:"MenuItemSeamless"`
	MenuItemRandom                 *gtk.CheckMenuItem     `build:"MenuItemRandom"`
	MenuItemPreferences            *gtk.MenuItem          `build:"MenuItemPreferences"`
	MenuItemHFlip                  *gtk.CheckMenuItem     `build:"MenuItemHFlip"`
	MenuItemVFlip                  *gtk.CheckMenuItem     `build:"MenuItemVFlip"`
	MenuItemMangaMode              *gtk.CheckMenuItem     `build:"MenuItemMangaMode"`
	MenuItemDoublePage             *gtk.CheckMenuItem     `build:"MenuItemDoublePage"`
	MenuItemGoTo                   *gtk.MenuItem          `build:"MenuItemGoTo"`
	GoToThumbnailImage             *gtk.Image             `build:"GoToThumbnailImage"`
	MenuItemBestFit                *gtk.RadioMenuItem     `build:"MenuItemBestFit"`
	MenuItemOriginal               *gtk.RadioMenuItem     `build:"MenuItemOriginal"`
	MenuItemFitToWidth             *gtk.RadioMenuItem     `build:"MenuItemFitToWidth"`
	MenuItemFitToHeight            *gtk.RadioMenuItem     `build:"MenuItemFitToHeight"`
	PreferencesDialog              *gtk.Dialog            `build:"PreferencesDialog"`
	PagesToSkipSpinButton          *gtk.SpinButton        `build:"PagesToSkipSpinButton"`
	GoToDialog                     *gtk.Dialog            `build:"GoToDialog"`
	GoToSpinButton                 *gtk.SpinButton        `build:"GoToSpinButton"`
	GoToScrollbar                  *gtk.Scrollbar         `build:"GoToScrollbar"`
	InterpolationComboBoxText      *gtk.ComboBoxText      `build:"InterpolationComboBoxText"`
	OneWideCheckButton             *gtk.CheckButton       `build:"OneWideCheckButton"`
	SmartScrollCheckButton         *gtk.CheckButton       `build:"SmartScrollCheckButton"`
	EmbeddedOrientationCheckButton *gtk.CheckButton       `build:"EmbeddedOrientationCheckButton"`
	AddBookmarkMenuItem            *gtk.MenuItem          `build:"AddBookmarkMenuItem"`
	MenuBookmarks                  *gtk.Menu              `build:"MenuBookmarks"`
	RecentChooserMenu              *gtk.RecentChooserMenu `build:"RecentChooserMenu"`
	Config                         Config
	State                          State
	RecentManager                  *gtk.RecentManager
}

// LoadWidgets() fills the GUI struct with widgets built from the
// glade UI file at the specified location
func (gui *GUI) LoadWidgets() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	builder, err := gtk.BuilderNew()
	if err != nil {
		return err
	}

	gomics_glade, err := gomics_glade()
	if err != nil {
		panic(err.Error())
	}
	if err = builder.AddFromString(string(gomics_glade)); err != nil {
		return err
	}

	guiStruct := reflect.ValueOf(gui).Elem()

	for i := 0; i < guiStruct.NumField(); i++ {
		field := guiStruct.Field(i)
		widget := guiStruct.Type().Field(i).Tag.Get("build")
		if widget == "" {
			continue
		}

		obj, err := builder.GetObject(widget)
		if err != nil {
			return err
		}

		w := reflect.ValueOf(obj).Convert(field.Type())
		field.Set(w)
	}

	return nil
}

func (gui *GUI) initUI() {
	// Load UI
	if err := gui.LoadWidgets(); err != nil {
		log.Fatal(err)
	}

	about, err := about_jpg()
	if err != nil {
		panic(err.Error())
	}
	gui.AboutDialog.SetLogo(mustLoadPixbuf(about))
	icon, err := icon_png()
	gui.MainWindow.SetIcon(mustLoadPixbuf(icon))
	if err != nil {
		panic(err.Error())
	}

	if len(gitVersion) >= 7 {
		version := fmt.Sprintf("Version: %s (built: %s)\nCompiler version: %s", gitVersion[:7], buildDate, runtime.Version())
		gui.AboutDialog.SetVersion(version)
	}

	gui.FileChooserDialogArchive.AddButton("_Open", gtk.RESPONSE_ACCEPT)
	gui.FileChooserDialogArchive.AddButton("_Cancel", gtk.RESPONSE_CANCEL)

	gui.PreferencesDialog.AddButton("_OK", gtk.RESPONSE_ACCEPT)

	gui.GoToDialog.AddButton("_Cancel", gtk.RESPONSE_CANCEL)
	gui.GoToDialog.AddButton("_Go", gtk.RESPONSE_ACCEPT)
	//gui.GoToDialog.SetDefaultResponse(gtk.RESPONSE_ACCEPT)

	gui.syncUI()

	// Connect signals
	gui.MenuItemAbout.Connect("activate", func() {
		gui.AboutDialog.Run()
		gui.AboutDialog.Hide()
	})

	gui.MenuItemOpen.Connect("activate", func() {
		res := gtk.ResponseType(gui.FileChooserDialogArchive.Run())
		gui.FileChooserDialogArchive.Hide()
		if res == gtk.RESPONSE_ACCEPT {
			filename := gui.FileChooserDialogArchive.GetFilename()
			gui.LoadArchive(filename)
		}
	})

	gui.MenuItemSaveImage.Connect("activate", gui.SavePNG)

	gui.MenuItemQuit.Connect("activate", gui.Quit)
	gui.MenuItemClose.Connect("activate", gui.Close)
	gui.MainWindow.Connect("delete-event", gui.Quit) // destroy

	var oldW, oldH int
	gui.MainWindow.Connect("size-allocate", func() {
		// Avoid unnecessary redraws
		w, h := gui.GetSize() // FIXME slow? use GdkRectangle *allocation passed in the signal
		if w == oldW && h == oldH {
			return
		}
		oldW, oldH = w, h
		gui.ResizeEvent()
	})

	gui.ButtonNextPage.Connect("clicked", gui.NextPage)
	gui.ButtonPreviousPage.Connect("clicked", gui.PreviousPage)
	gui.ButtonFirstPage.Connect("clicked", gui.FirstPage)
	gui.ButtonLastPage.Connect("clicked", gui.LastPage)
	gui.ButtonNextArchive.Connect("clicked", gui.NextArchive)
	gui.ButtonPreviousArchive.Connect("clicked", gui.PreviousArchive)
	gui.ButtonNextScene.Connect("clicked", gui.NextScene)
	gui.ButtonPreviousScene.Connect("clicked", gui.PreviousScene)
	gui.ButtonSkipForward.Connect("clicked", gui.SkipForward)
	gui.ButtonSkipBackward.Connect("clicked", gui.SkipBackward)

	gui.MenuItemNextPage.Connect("activate", gui.NextPage)
	gui.MenuItemPreviousPage.Connect("activate", gui.PreviousPage)
	gui.MenuItemFirstPage.Connect("activate", gui.FirstPage)
	gui.MenuItemLastPage.Connect("activate", gui.LastPage)
	gui.MenuItemNextArchive.Connect("activate", gui.NextArchive)
	gui.MenuItemPreviousArchive.Connect("activate", gui.PreviousArchive)
	gui.MenuItemSkipForward.Connect("activate", gui.SkipForward)
	gui.MenuItemSkipBackward.Connect("activate", gui.SkipBackward)

	gui.MenuItemEnlarge.Connect("toggled", func() {
		gui.SetEnlarge(gui.MenuItemEnlarge.GetActive())
	})

	gui.MenuItemShrink.Connect("toggled", func() {
		gui.SetShrink(gui.MenuItemShrink.GetActive())
	})

	gui.MenuItemFullscreen.Connect("toggled", func() {
		gui.SetFullscreen(gui.MenuItemFullscreen.GetActive())
	})

	gui.MenuItemSeamless.Connect("toggled", func() {
		gui.SetSeamless(gui.MenuItemSeamless.GetActive())
	})

	gui.MenuItemRandom.Connect("toggled", func() {
		gui.SetRandom(gui.MenuItemRandom.GetActive())
	})

	gui.MenuItemHFlip.Connect("toggled", func() {
		gui.SetHFlip(gui.MenuItemHFlip.GetActive())
	})

	gui.MenuItemVFlip.Connect("toggled", func() {
		gui.SetVFlip(gui.MenuItemVFlip.GetActive())
	})

	gui.MenuItemMangaMode.Connect("toggled", func() {
		gui.SetMangaMode(gui.MenuItemMangaMode.GetActive())
	})

	gui.MenuItemDoublePage.Connect("toggled", func() {
		gui.SetDoublePage(gui.MenuItemDoublePage.GetActive())
	})

	gui.MenuItemOriginal.Connect("toggled", func() {
		if gui.MenuItemOriginal.GetActive() {
			gui.SetZoomMode("Original")
		}
	})

	gui.MenuItemBestFit.Connect("toggled", func() {
		if gui.MenuItemBestFit.GetActive() {
			gui.SetZoomMode("BestFit")
		}
	})

	gui.MenuItemFitToWidth.Connect("toggled", func() {
		if gui.MenuItemFitToWidth.GetActive() {
			gui.SetZoomMode("FitToWidth")
		}
	})

	gui.MenuItemFitToHeight.Connect("toggled", func() {
		if gui.MenuItemFitToHeight.GetActive() {
			gui.SetZoomMode("FitToHeight")
		}
	})

	gui.MenuItemPreferences.Connect("activate", func() {
		res := gtk.ResponseType(gui.PreferencesDialog.Run())
		gui.PreferencesDialog.Hide()
		if res == gtk.RESPONSE_ACCEPT {
			// TODO save config
		}
	})

	gui.MenuItemGoTo.Connect("activate", func() {
		gui.RunGoToDialog()
	})

	gui.GoToSpinButton.Connect("value-changed", func() {
		gui.GoToScrollbar.SetValue(gui.GoToSpinButton.GetValue())
		// TODO load & display the thumbnail image
	})

	gui.GoToScrollbar.Connect("value-changed", func() {
		gui.GoToSpinButton.SetValue(gui.GoToScrollbar.GetValue())
		gui.goToDialogLoadSetThumbnail()
		// load & display the thumbnail image
	})

	gui.RecentChooserMenu.Connect("item-activated", func() {
		uri := gui.RecentChooserMenu.GetCurrentUri()
		gui.LoadArchive(uri)
	})

	gui.PagesToSkipSpinButton.SetRange(1, 100)
	gui.PagesToSkipSpinButton.SetIncrements(1, 10)
	gui.PagesToSkipSpinButton.SetValue(float64(gui.Config.NSkip))

	gui.PagesToSkipSpinButton.Connect("value-changed", func() {
		gui.Config.NSkip = int(gui.PagesToSkipSpinButton.GetValue())
		gui.goToDialogLoadSetThumbnail()
	})

	gui.InterpolationComboBoxText.Connect("changed", func() {
		gui.SetInterpolation(gui.InterpolationComboBoxText.GetActive())
	})

	gui.OneWideCheckButton.Connect("toggled", func() {
		gui.SetOneWide(gui.OneWideCheckButton.GetActive())
	})

	gui.SmartScrollCheckButton.Connect("toggled", func() {
		gui.SetSmartScroll(gui.SmartScrollCheckButton.GetActive())
	})

	gui.EmbeddedOrientationCheckButton.Connect("toggled", func() {
		gui.SetEmbeddedOrientation(gui.EmbeddedOrientationCheckButton.GetActive())
	})

	gui.AddBookmarkMenuItem.Connect("activate", func() {
		gui.AddBookmark()
	})

	gui.ScrolledWindow.SetEvents(gui.ScrolledWindow.GetEvents() | int(gdk.BUTTON_PRESS_MASK))

	gui.ScrolledWindow.Connect("scroll-event", func(w *gtk.ScrolledWindow, e *gdk.Event) {
		se := &gdk.EventScroll{e}

		gui.Scroll(se.DeltaX(), se.DeltaY())
	})

	// FIXME
	gui.ScrolledWindow.Connect("button-press-event", func(_ *gtk.ScrolledWindow, e *gdk.Event) bool {
		//log.Println(w)
		be := &gdk.EventButton{e}
		switch be.Button() {
		case 1:
			gui.NextPage()
		case 3:
			gui.PreviousPage()
		case 2:
			gui.NextArchive()
		}
		return true
	})

	gui.MainWindow.Connect("key-press-event", func(_ *gtk.Window, e *gdk.Event) {
		ke := &gdk.EventKey{e}

		shift := ke.State()&uint(gdk.GDK_SHIFT_MASK) != 0
		ctrl := ke.State()&uint(gdk.GDK_CONTROL_MASK) != 0

		switch ke.KeyVal() {
		case gdk.KEY_Down:
			if ctrl {
				gui.NextArchive()
			} else if shift {
				gui.Scroll(0, 1)
			} else {
				gui.NextPage()
			}
		case gdk.KEY_Up:
			if ctrl {
				gui.PreviousArchive()
			} else if shift {
				gui.Scroll(0, -1)
			} else {
				gui.PreviousPage()
			}
		case gdk.KEY_Right:
			if ctrl {
				gui.NextScene()
			} else if shift {
				gui.Scroll(1, 0)
			} else {
				gui.SkipForward()
			}
		case gdk.KEY_Left:
			if ctrl {
				gui.PreviousScene()
			} else if shift {
				gui.Scroll(-1, 0)
			} else {
				gui.SkipBackward()
			}
		}
	})

	gui.RebuildBookmarksMenu()

	gui.MainWindow.SetDefaultSize(gui.Config.WindowWidth, gui.Config.WindowHeight)
	gui.MainWindow.ShowAll()

	// Tiny hack
	mw, mh := gui.MainWindow.GetSize()
	va := gui.Viewport.GetAllocation()
	gui.State.DeltaW, gui.State.DeltaH = mw-va.GetWidth(), mh-va.GetHeight()

	gui.SetFullscreen(gui.Config.Fullscreen)

	gui.SetZoomMode(gui.Config.ZoomMode)
	gui.SetDoublePage(gui.Config.DoublePage)
	gui.SetMangaMode(gui.Config.MangaMode)

	gui.fixFocus()
}

func (gui *GUI) goToDialogLoadSetThumbnail() {
	n := int(gui.GoToSpinButton.GetValue() - 1)
	pixbuf, err := gui.State.Archive.Load(n, gui.Config.EmbeddedOrientation)
	if err != nil {
		gui.ShowError(err.Error())
		return
	}

	w, h := fit(pixbuf.GetWidth(), pixbuf.GetHeight(), 128, 128)

	scaled, err := pixbuf.ScaleSimple(w, h, interpolations[gui.Config.Interpolation])
	if err != nil {
		gui.ShowError(err.Error())
		return
	}

	gui.State.GoToThumnailPixbuf = scaled
	gui.GoToThumbnailImage.SetFromPixbuf(scaled)

	gc()
}

func (gui *GUI) syncUI() {
	// Sync config & UI
	gui.MenuItemEnlarge.SetActive(gui.Config.Enlarge)
	gui.MenuItemShrink.SetActive(gui.Config.Shrink)
	gui.MenuItemHFlip.SetActive(gui.Config.HFlip)
	gui.MenuItemVFlip.SetActive(gui.Config.VFlip)
	gui.MenuItemRandom.SetActive(gui.Config.Random)
	gui.MenuItemSeamless.SetActive(gui.Config.Seamless)
	gui.MenuItemDoublePage.SetActive(gui.Config.DoublePage)
	gui.MenuItemMangaMode.SetActive(gui.Config.MangaMode)

	switch gui.Config.ZoomMode {
	case "FitToWidth":
		gui.MenuItemFitToWidth.SetActive(true)
	case "FitToHeight":
		gui.MenuItemFitToHeight.SetActive(true)
	case "BestFit":
		gui.MenuItemBestFit.SetActive(true)
	default:
		gui.MenuItemOriginal.SetActive(true)
	}

	gui.InterpolationComboBoxText.SetActive(gui.Config.Interpolation)
	gui.OneWideCheckButton.SetActive(gui.Config.OneWide)
	gui.EmbeddedOrientationCheckButton.SetActive(gui.Config.EmbeddedOrientation)
}

func (gui *GUI) RunGoToDialog() {
	if !gui.Loaded() {
		return
	}

	gui.GoToSpinButton.SetRange(1, float64(gui.State.Archive.Len()))
	gui.GoToSpinButton.SetValue(float64(gui.State.ArchivePos) + 1)
	gui.GoToSpinButton.SetIncrements(1, float64(gui.Config.NSkip))

	gui.GoToScrollbar.SetRange(1, float64(gui.State.Archive.Len()))
	gui.GoToScrollbar.SetValue(float64(gui.State.ArchivePos) + 1)
	gui.GoToScrollbar.SetIncrements(1, float64(gui.State.Archive.Len()))

	gui.goToDialogLoadSetThumbnail()

	res := gtk.ResponseType(gui.GoToDialog.Run())
	gui.GoToDialog.Hide()
	if res == gtk.RESPONSE_ACCEPT {
		gui.SetPage(int(gui.GoToSpinButton.GetValue()) - 1)

		gui.GoToThumbnailImage.Clear()
		gui.State.GoToThumnailPixbuf = nil
		gc()
	}
}
