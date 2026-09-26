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
	"fmt"
	"log"
	"math"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/mappu/miqt/qt6"
	"github.com/salviati/gomics/archive"
	"github.com/salviati/gomics/imgdiff"
)

import (
	_ "embed"
)

//go:embed about.jpg
var about []byte

//go:embed icon.png
var icon []byte

type GUI struct {
	*MainWindowUi
	PreferencesUI    *PreferencesDialogUi
	GoToUI           *GoToDialogUi
	AboutUI          *AboutDialogUi
	Config           Config
	State            State
	zoomLevelActions []*qt6.QAction
	rotationActions  []*qt6.QAction
	StatusParts      *StatusParts
}

var zoomLevels = []int{25, 33, 50, 66, 75, 100, 125, 150, 200, 300, 400, 800}
var rotationLevels = []int{0, 90, 180, 270}

type Bookmark struct {
	Path       string
	Page       uint
	TotalPages uint
	Added      time.Time
}

var bookmarkActionsList []*qt6.QAction
var recentFileActions []*qt6.QAction

const StatusPartsMinWidth = 12

// The status bar is a row of individual sunken cells, one per logical part
// of the message, not QStatusBar's temporary message slot: Qt writes menu
// status hints into that slot while the mouse is over the menubar and
// clears it on leave, which wiped our status.
type StatusParts struct {
	labels []*qt6.QLabel
	bar    *qt6.QStatusBar
}

func NewStatusParts(bar *qt6.QStatusBar, texts ...string) *StatusParts {
	sp := &StatusParts{bar: bar}
	sp.Set(texts...)
	return sp
}

func (sp *StatusParts) set(idx int, text string) {
	sp.labels[idx].SetText(text)
}

func (sp *StatusParts) Set(texts ...string) {
	if len(sp.labels) != len(texts)+1 {
		for _, lbl := range sp.labels {
			sp.bar.RemoveWidget(lbl.QWidget)
			lbl.DeleteLater()
		}

		if len(texts) == 1 && texts[0] == "" {
			texts = make([]string, 0, 0)
		}

		sp.labels = make([]*qt6.QLabel, len(texts)+1)

		for i := range sp.labels {
			lbl := qt6.NewQLabel2()
			lbl.SetFrameShape(qt6.QFrame__Panel)
			lbl.SetFrameShadow(qt6.QFrame__Sunken)
			if i < len(texts) {
				lbl.SetMinimumWidth(StatusPartsMinWidth)
				sp.bar.AddWidget2(lbl.QWidget, 0)
			} else {
				sp.bar.AddWidget2(lbl.QWidget, 1) // absorbs extra space
			}
			sp.labels[i] = lbl
		}
	}

	for i, text := range texts {
		sp.set(i, text)
	}
}

func (gui *GUI) SetStatus(msgs ...string) {
	gui.StatusParts.Set(msgs...)
}

func (gui *GUI) ResizeEvent() {
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) ShowError(msg string) {
	log.Println(msg)
	gui.SetStatus(msg)
}

func (gui *GUI) deletePixmaps() {
	if gui.State.QImageL != nil {
		deleteQImage(gui.State.QImageL)
		gui.State.QImageL = nil
	}
	if gui.State.QImageR != nil {
		deleteQImage(gui.State.QImageR)
		gui.State.QImageR = nil
	}
	if gui.State.GoToThumbnail != nil {
		deleteQImage(gui.State.GoToThumbnail)
		gui.State.GoToThumbnail = nil
	}
	gc()
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

	// Reset the scene rect to the viewport: Blit only sizes it while an
	// image is loaded, so after Close the stale oversized rect would let
	// Qt show scrollbars on a window shrink (scene rect > viewport).
	vw, vh := gui.GetSize()
	gui.Image.SetSceneRect2(0, 0, float64(vw), float64(vh))
	gui.Image.Scene().Clear()
	gui.deletePixmaps()
	gui.State.CursorLastMoved = time.Now()
	gui.State.CursorHidden = false
	gui.State.CursorForceShown = false
	gui.SetStatus("No file loaded")
	gui.MainWindow.SetWindowTitle("Gomics")
	gc()
}

// recordRecent: only explicit opens (Open dialog, CLI) add Recent Files
// entries; seamless navigation and reopens never do.
func (gui *GUI) LoadArchive(path string, recordRecent bool) {
	if len(path) == 0 {
		return
	}

	// Canonicalize to an absolute path up front so NewArchive, navigation
	// listing, Recent Files and bookmarks all resolve identically regardless
	// of the process cwd. Replaces the old os.Chdir quirk that moved the
	// cwd into the archive dir and made a relative ArchivePath re-resolve
	// against the wrong directory (breaking next/prev archive navigation).
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}

	if gui.Loaded() {
		gui.Close()
	}

	gui.State.ImageHash = make(map[int]imgdiff.Hash)

	gui.State.ArchivePath = path
	if archive.IsImageFile(path) {
		gui.State.ArchiveName = filepath.Base(filepath.Dir(path))
	} else {
		gui.State.ArchiveName = filepath.Base(path)
	}

	var err error
	if gui.State.Archive, err = archive.NewArchive(path); err != nil {
		gui.ShowError("Failed to open " + path + ": " + err.Error())
		return
	}

	start := 0
	if archive.IsImageFile(path) {
		if i := gui.State.Archive.Locate(filepath.Base(path)); i >= 0 {
			start = i
		}
	}

	gui.setPage(start)

	if recordRecent {
		gui.updateRecentFiles(path)
	}
}

func (gui *GUI) LoadImage(n int) (*qt6.QImage, error) {
	ar := gui.State.Archive
	img, err := ar.Load(n, gui.Config.EmbeddedOrientation)

	if err != nil {
		filename, _ := ar.Name(n)
		gui.ShowError(fmt.Sprintf(`Failed to load file #%d "%s": %s`, n+1, filename, err.Error()))
		return nil, err
	}

	gui.State.ImageHash[n], _ = gui.ImageHash(n, img)
	return img, nil
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

// clearDisplay resets the loaded-image state and clears the scene so no
// stale image remains after a failed load.
func (gui *GUI) clearDisplay() {
	gui.deletePixmaps()
	gui.Image.Scene().Clear()
}

func (gui *GUI) setPage(n int) {
	if !gui.Loaded() {
		return
	}

	gui.State.ArchivePos = n

	gui.deletePixmaps()

	var err error
	gui.State.QImageL, err = gui.LoadImage(n)
	if err != nil {
		gui.clearDisplay()
		return
	}

	if gui.Config.DoublePage && !(gui.Config.SingleCover && n == 0) && n+1 < gui.State.Archive.Len() {
		if img, err := gui.LoadImage(n + 1); err == nil {
			gui.State.QImageR = img
		}
		// else: keep QR nil so Blit renders the left image only.
	}

	gui.Blit()
	gui.StatusImage()

	gui.scrollToTop()
}

func (gui *GUI) Scroll(dx, dy float64) {
	if !gui.Loaded() {
		return
	}

	vadj := gui.Image.VerticalScrollBar()
	hadj := gui.Image.HorizontalScrollBar()

	vdx := vadj.SingleStep()
	vval := float64(vadj.Value())
	// Qt clamps the value to [Minimum, Maximum] and Maximum is exactly where
	// the viewport bottom meets the scene bottom (scene height - viewport
	// height), so the edge is reached at value >= Maximum. Fitting content
	// gives Maximum == 0, i.e. already at both edges -> page turn.
	vupper := float64(vadj.Maximum())
	vlower := float64(vadj.Minimum())

	hdx := hadj.SingleStep()
	hval := float64(hadj.Value())
	hupper := float64(hadj.Maximum())
	hlower := float64(hadj.Minimum())

	if dy > 0 {
		if vval >= vupper {
			if gui.Config.SmartScroll {
				gui.NextPage()
			}
		} else {
			vadj.SetValue(int(clamp(vval+float64(vdx), vlower, vupper)))
		}
	} else if dy < 0 {
		if vval <= vlower {
			if gui.Config.SmartScroll {
				gui.PreviousPage()
			}
		} else {
			vadj.SetValue(int(clamp(vval-float64(vdx), vlower, vupper)))
		}
	}

	if dx > 0 {
		if hval >= hupper {
			// TODO
		} else {
			hadj.SetValue(int(clamp(hval+float64(hdx), hlower, hupper)))
		}
	} else if dx < 0 {
		if hval <= hlower {
			// TODO
		} else {
			hadj.SetValue(int(clamp(hval-float64(hdx), hlower, hupper)))
		}
	}
}

func (gui *GUI) scrollToTop() {
	if !gui.Loaded() {
		return
	}

	vadj := gui.Image.VerticalScrollBar()
	vadj.SetValue(0)

	hadj := gui.Image.HorizontalScrollBar()
	hadj.SetValue(0)
}

func (gui *GUI) Quit() {
	gui.saveConfig()
	gui.MainWindow.Close()
}

func (gui *GUI) saveConfig() {
	gui.Config.WindowWidth, gui.Config.WindowHeight = gui.MainWindow.Size().Width(), gui.MainWindow.Size().Height()

	if err := gui.Config.Save(filepath.Join(gui.State.ConfigPath, ConfigFile)); err != nil {
		log.Println(err)
	}
}

func (gui *GUI) Init() {
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

	// Migrate legacy zoom mode: "Original" (removed) becomes Manual 100%.
	if gui.Config.ZoomMode == "Original" {
		gui.Config.ZoomMode = "Manual"
	}
	if gui.Config.ZoomMode == "Manual" && gui.Config.ZoomLevel == 0 {
		gui.Config.ZoomLevel = 100
	}

	gui.initUI()
}

func (gui *GUI) SetFullscreen(fullscreen bool) {
	gui.Config.Fullscreen = fullscreen
	if fullscreen {
		gui.statusbar.Hide()
		gui.toolBar.Hide()
		gui.menubar.Hide()
		gui.MainWindow.ShowFullScreen()
	} else {
		gui.statusbar.Show()
		gui.toolBar.Show()
		gui.menubar.Show()
		gui.MainWindow.ShowNormal()
	}

	// Fullscreen: image fills edge-to-edge (0 margins); windowed keeps 11px.
	if fullscreen {
		gui.verticalLayout.SetContentsMargins(0, 0, 0, 0)
	} else {
		//gui.verticalLayout.SetContentsMargins(11, 11, 11, 11)
		gui.verticalLayout.SetContentsMargins(0, 0, 0, 0)
	}

	gui.actionFullscreen.SetChecked(fullscreen)
}

func (gui *GUI) SetShrink(shrink bool) {
	gui.Config.Shrink = shrink
	gui.actionShrink_large_images.SetChecked(shrink)
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) SetEnlarge(enlarge bool) {
	gui.Config.Enlarge = enlarge
	gui.actionEnlarge_small_images.SetChecked(enlarge)
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) SetRandom(random bool) {
	gui.Config.Random = random
	gui.actionRandom_ordering.SetChecked(random)
}

func (gui *GUI) SetSeamless(seamless bool) {
	gui.Config.Seamless = seamless
	gui.actionSeamless_mode.SetChecked(seamless)
}

func (gui *GUI) SetHFlip(hflip bool) {
	gui.Config.HFlip = hflip
	gui.actionH_flip.SetChecked(hflip)
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) SetVFlip(vflip bool) {
	gui.Config.VFlip = vflip
	gui.actionV_flip.SetChecked(vflip)
	gui.Blit()
	gui.StatusImage()
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
	ok := gui.State.QImageL.Save(pngPath)
	if !ok {
		gui.ShowError("Failed to save image")
		return
	}

	gui.SetStatus("Saved to " + pngBase)
}

func (gui *GUI) SetZoomMode(mode string) {
	switch mode {
	case "FitToWidth":
		gui.actionFit_to_width.SetChecked(true)
	case "FitToHeight":
		gui.actionFit_to_height.SetChecked(true)
	case "BestFit":
		gui.actionBest_fit.SetChecked(true)
	default:
		return
	}

	gui.Config.ZoomMode = mode
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) setZoomLevel(level int) {
	gui.Config.ZoomMode = "Manual"
	gui.Config.ZoomLevel = level
	for i, a := range gui.zoomLevelActions {
		a.SetChecked(zoomLevels[i] == level)
	}
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) zoomStep(dir int) {
	// Step from the on-screen zoom (the manual level in Manual mode, the fit
	// scale otherwise), so + lands on the nearest preset above and - on the
	// nearest preset below where we currently are.
	cur := int(math.Round(100 * gui.State.Scale))
	n := -1
	for j, l := range zoomLevels {
		if dir > 0 && l > cur {
			n = j
			break
		}
		if dir < 0 && l < cur {
			n = j
		}
	}
	if n == -1 {
		// No preset in that direction: clamp to the end of the list.
		if dir > 0 {
			n = len(zoomLevels) - 1
		} else {
			n = 0
		}
	}
	gui.setZoomLevel(zoomLevels[n])
}

func (gui *GUI) setRotation(angle int) {
	r := angle % 360
	if r < 0 {
		r += 360
	}
	gui.Config.Rotation = r
	for i, a := range gui.rotationActions {
		a.SetChecked(rotationLevels[i] == r)
	}
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) Rotate(delta int) {
	gui.setRotation(gui.Config.Rotation + delta)
}

func (gui *GUI) SetDoublePage(doublePage bool) {
	gui.Config.DoublePage = doublePage
	gui.actionDouble_page.SetChecked(doublePage)
	gui.setPage(gui.State.ArchivePos)
}

func (gui *GUI) SetMangaMode(mangaMode bool) {
	gui.Config.MangaMode = mangaMode
	gui.actionManga_mode.SetChecked(mangaMode)
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) SetOneWide(oneWide bool) {
	gui.Config.OneWide = oneWide
	gui.PreferencesUI.OneWideCheckButton.SetChecked(oneWide)
	gui.Blit()
	gui.StatusImage()
}

func (gui *GUI) SetSingleCover(singleCover bool) {
	gui.Config.SingleCover = singleCover
	gui.PreferencesUI.SingleCoverCheckButton.SetChecked(singleCover)
	if gui.State.ArchivePos == 0 {
		gui.setPage(0)
	}
}

func (gui *GUI) SetSmartScroll(smartScroll bool) {
	gui.Config.SmartScroll = smartScroll
	gui.PreferencesUI.SmartScrollCheckButton.SetChecked(smartScroll)
}

func (gui *GUI) SetHideIdleCursor(hideIdleCursor bool) {
	gui.Config.HideIdleCursor = hideIdleCursor
	gui.PreferencesUI.HideIdleCursorCheckButton.SetChecked(hideIdleCursor)
}

func (gui *GUI) SetEmbeddedOrientation(embeddedOrientation bool) {
	gui.Config.EmbeddedOrientation = embeddedOrientation
	gui.PreferencesUI.EmbeddedOrientationCheckButton.SetChecked(embeddedOrientation)
	gui.Blit()
	gui.StatusImage()
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

// UpdateCursorVisibility checks if cursor should be hidden based on inactivity
func (gui *GUI) UpdateCursorVisibility() bool {
	cursorShouldBeHidden := false

	if gui.Config.HideIdleCursor && !gui.State.CursorForceShown {
		cursorShouldBeHidden = time.Since(gui.State.CursorLastMoved).Seconds() > 1
	}

	if cursorShouldBeHidden && !gui.State.CursorHidden {
		gui.HideCursor()
	}

	if !cursorShouldBeHidden && gui.State.CursorHidden {
		gui.ShowCursor()
	}

	return true
}

func (gui *GUI) HideCursor() {
	gui.Image.SetCursor(qt6.NewQCursor2(qt6.BlankCursor))
	gui.State.CursorHidden = true
}

func (gui *GUI) ShowCursor() {
	gui.Image.SetCursor(qt6.NewQCursor2(qt6.ArrowCursor))
	gui.State.CursorHidden = false
}

// setCursorVisible toggles cursor visibility based on force-show state
func (gui *GUI) setCursorVisible(visible bool) {
	if visible {
		gui.State.CursorForceShown = true
		gui.ShowCursor()
	} else {
		gui.State.CursorForceShown = false
		// Let UpdateCursorVisibility decide
	}
}

// showCursorForDialog keeps cursor visible during dialog execution
func (gui *GUI) showCursorForDialog(fn func()) {
	gui.State.CursorForceShown = true
	gui.ShowCursor()
	fn()
	gui.State.CursorForceShown = false
	// UpdateCursorVisibility will restore normal behavior
}

func (gui *GUI) initUI() {
	// Load generated UI structures
	gui.MainWindowUi = NewMainWindowUi()
	gui.PreferencesUI = NewPreferencesDialogUi()
	gui.GoToUI = NewGoToDialogUi()
	gui.AboutUI = NewAboutDialogUi()

	// Create and assign the graphics scene so Image.Scene() is non-nil.
	scene := qt6.NewQGraphicsScene()
	gui.Image.SetScene(scene)

	// Pin the scene to the viewport origin instead of centering it against
	// the view widget: when a scrollbar shrinks the paintable viewport below
	// the widget, Qt's centering (against the widget rect) left a blank bar
	// on the fit dimension. We do our own centering in Blit via offx/offy.
	gui.Image.SetAlignment(qt6.AlignLeft | qt6.AlignTop)

	// Drag-to-scroll: LMB press+move scrolls larger-than-view images.
	gui.Image.SetDragMode(qt6.QGraphicsView__ScrollHandDrag)

	// Go To dialog thumbnail view: assign its scene so .Scene() is non-nil
	// (a fresh QGraphicsView has none; goToDialogLoadThumbnail clears it).
	gui.GoToUI.GoToThumbnailImage.SetScene(qt6.NewQGraphicsScene())

	// Same pinning as the main Image view so best-fit centering is applied
	// exactly once (no widget-vs-viewport offset).
	gui.GoToUI.GoToThumbnailImage.SetAlignment(qt6.AlignLeft | qt6.AlignTop)

	// Re-fit to the available space when the view is resized (also fires on
	// first show, once the dialog is laid out).
	gui.GoToUI.GoToThumbnailImage.OnResizeEvent(func(super func(event *qt6.QResizeEvent), event *qt6.QResizeEvent) {
		super(event)
		gui.goToDialogLoadThumbnail()
	})

	// Keep spin and scrollbar in sync; the spin handler loads the thumbnail.
	gui.GoToUI.GoToSpinButton.OnValueChanged(func(val int) {
		gui.GoToUI.GoToScrollbar.SetValue(val)
		gui.goToDialogLoadThumbnail()
	})
	gui.GoToUI.GoToScrollbar.OnValueChanged(func(val int) {
		gui.GoToUI.GoToSpinButton.SetValue(val)
	})

	// Window icon on the main window and every dialog.
	iconImg := mustLoadQImage(icon)
	winIcon := qt6.NewQIcon2(qt6.QPixmap_FromImage(iconImg))
	gui.MainWindow.SetWindowIcon(winIcon)
	gui.PreferencesUI.PreferencesDialog.SetWindowIcon(winIcon)
	gui.AboutUI.AboutDialog.SetWindowIcon(winIcon)
	gui.GoToUI.Dialog.SetWindowIcon(winIcon)

	// Status parts: one sunken cell per logical part of a status message,
	// plus the stretch cell Set appends; a message with a different part
	// count rebuilds the row (the four-part page status vs this single part).
	gui.StatusParts = NewStatusParts(gui.statusbar, "No file loaded")

	// About dialog logo: set the pixmap on the existing "image" QLabel.
	aboutImg := mustLoadQImage(about)
	gui.AboutUI.image.SetPixmap(qt6.QPixmap_FromImage(aboutImg))

	// Version text: append to the existing "text" QLabel.
	if len(gitVersion) >= 7 {
		version := fmt.Sprintf("Version: git-%s (built: %s), compiler: %s", gitVersion[:7], buildDate, runtime.Version())
		gui.AboutUI.text.SetText(gui.AboutUI.text.Text() + "\n" + version)
	}

	// Set up background color brush
	gui.setupBackgroundColor()

	// Setup viewport input handlers
	gui.setupViewportInput()

	// Setup cursor auto-hide
	gui.setupCursorAutoHide()

	// Save config on any quit path (window close or menu Quit)
	gui.MainWindow.OnCloseEvent(func(super func(event *qt6.QCloseEvent), event *qt6.QCloseEvent) {
		super(event)
		gui.saveConfig()
	})

	// Connect menu actions
	gui.connectMenuActions()

	// Set up checkable actions (uic does NOT emit SetCheckable)
	gui.setupCheckableActions()

	// Promote action shortcuts so they still fire in fullscreen mode
	gui.setupActionShortcuts()

	// Wire preferences dialog controls to config
	gui.setupPreferencesDialog()

	// Setup zoom action group (exclusive)
	gui.setupZoomActionGroup()

	// Sync UI state from config
	gui.syncUI()

	// Rebuild bookmarks and recent files (runtime QActions)
	gui.RebuildBookmarksMenu()
	gui.menu_Recent_Files.QWidget.RemoveAction(gui.action_no_recent_items)
	gui.buildRecentFilesMenu()

	// Set window size
	gui.MainWindow.Resize(gui.Config.WindowWidth, gui.Config.WindowHeight)

	// Setup fullscreen state
	gui.SetFullscreen(gui.Config.Fullscreen)

	// Set zoom mode
	gui.SetZoomMode(gui.Config.ZoomMode)
	gui.SetDoublePage(gui.Config.DoublePage)
	gui.SetMangaMode(gui.Config.MangaMode)

	gui.MainWindow.Show()
}

func (gui *GUI) setupViewportInput() {
	// Install handlers on the directly-constructed QGraphicsView. miqt only
	// allows overriding virtual methods on types we constructed ourselves;
	// Image.Viewport() returns an internally-created QWidget that is not
	// overridable, so all event slots are wired on gui.Image instead.
	vp := gui.Image

	// Mouse press: middle=next-archive; LMB is handled by Qt (ScrollHandDrag).
	vp.OnMousePressEvent(func(super func(event *qt6.QMouseEvent), event *qt6.QMouseEvent) {
		gui.State.CursorLastMoved = time.Now()
		switch event.Button() {
		case qt6.LeftButton:
			super(event)
		case qt6.MiddleButton:
			gui.NextArchive()
		}
	})

	// Wheel: up=prev page, down=next page; Ctrl+wheel zooms in/out.
	vp.OnWheelEvent(func(_ func(event *qt6.QWheelEvent), event *qt6.QWheelEvent) {
		gui.State.CursorLastMoved = time.Now()
		ctrl := qt6.QGuiApplication_QueryKeyboardModifiers()&qt6.ControlModifier != 0
		if ctrl {
			if event.AngleDelta().Y() > 0 {
				gui.zoomStep(1)
			} else {
				gui.zoomStep(-1)
			}
		} else if event.AngleDelta().Y() > 0 {
			gui.PreviousPage()
		} else {
			gui.NextPage()
		}
	})

	// Key press
	vp.OnKeyPressEvent(func(_ func(event *qt6.QKeyEvent), event *qt6.QKeyEvent) {
		shift := event.Modifiers()&qt6.ShiftModifier != 0
		ctrl := event.Modifiers()&qt6.ControlModifier != 0

		switch key := qt6.Key(event.Key()); key {
		case qt6.Key_Down:
			if ctrl {
				gui.NextArchive()
			} else if shift {
				gui.Scroll(0, 1)
			} else {
				gui.NextPage()
			}
		case qt6.Key_Up:
			if ctrl {
				gui.PreviousArchive()
			} else if shift {
				gui.Scroll(0, -1)
			} else {
				gui.PreviousPage()
			}
		case qt6.Key_Right:
			if ctrl {
				gui.NextScene()
			} else if shift {
				gui.Scroll(1, 0)
			} else {
				gui.SkipForward()
			}
		case qt6.Key_Left:
			if ctrl {
				gui.PreviousScene()
			} else if shift {
				gui.Scroll(-1, 0)
			} else {
				gui.SkipBackward()
			}
		case qt6.Key_R:
			if !ctrl {
				if shift {
					gui.Rotate(-90)
				} else {
					gui.Rotate(90)
				}
			}
		case qt6.Key_Plus:
			gui.zoomStep(1)
		case qt6.Key_Minus:
			gui.zoomStep(-1)
		case qt6.Key_Asterisk:
			gui.setZoomLevel(100)
		case qt6.Key_Slash:
			gui.SetZoomMode("BestFit")
		case qt6.Key_Space:
			gui.NextPage()
		}
	})

	// Mouse move: forward to C++ base (ScrollHandDrag needs it) + cursor tracking
	vp.OnMouseMoveEvent(func(super func(event *qt6.QMouseEvent), event *qt6.QMouseEvent) {
		super(event)
		gui.State.CursorLastMoved = time.Now()
	})

	// Resize: relayout so the image stays centered and correctly sized
	vp.OnResizeEvent(func(super func(event *qt6.QResizeEvent), event *qt6.QResizeEvent) {
		super(event)
		gui.ResizeEvent()
	})

	// Enter/leave for cursor
	vp.OnEnterEvent(func(_ func(event *qt6.QEnterEvent), event *qt6.QEnterEvent) {
		if gui.State.CursorHidden {
			gui.State.CursorForceShown = true
		}
	})
	vp.OnLeaveEvent(func(_ func(event *qt6.QEvent), event *qt6.QEvent) {
		gui.State.CursorForceShown = false
	})
}

func (gui *GUI) setupCursorAutoHide() {
	// Timer for cursor visibility
	timer := qt6.NewQTimer()
	timer.SetInterval(250)
	timer.OnTimeout(func() {
		gui.UpdateCursorVisibility()
	})
	timer.Start(250)
}

func (gui *GUI) setupPreferencesDialog() {
	p := gui.PreferencesUI

	p.OneWideCheckButton.OnToggled(func(checked bool) {
		gui.SetOneWide(checked)
	})
	p.SingleCoverCheckButton.OnToggled(func(checked bool) {
		gui.SetSingleCover(checked)
	})
	p.SmartScrollCheckButton.OnToggled(func(checked bool) {
		gui.SetSmartScroll(checked)
	})
	p.HideIdleCursorCheckButton.OnToggled(func(checked bool) {
		gui.SetHideIdleCursor(checked)
	})
	p.EmbeddedOrientationCheckButton.OnToggled(func(checked bool) {
		gui.SetEmbeddedOrientation(checked)
	})
	p.UseBackgroundColorCheckButton.OnToggled(func(checked bool) {
		gui.Config.UseBackgroundColor = checked
		gui.setupBackgroundColor()
	})
	p.PagesToSkipSpinButton.SetRange(1, 100)
	p.PagesToSkipSpinButton.OnValueChanged(func(v int) {
		gui.Config.NSkip = v
	})
	p.InterpolationComboBoxText.OnCurrentIndexChanged(func(index int) {
		gui.Config.Interpolation = index
	})
	p.BackgroundColorButton.OnClicked(func() {
		var init *qt6.QColor
		if gui.Config.UseBackgroundColor {
			c := qt6.NewQColor()
			c.SetNamedColor(gui.Config.BackgroundColor)
			init = c
		} else {
			init = qt6.NewQColor2(qt6.DarkGray)
		}
		dlg := qt6.NewQColorDialog4(init, p.PreferencesDialog.QWidget)
		if res := dlg.Exec(); res == int(qt6.QDialog__Accepted) {
			gui.Config.UseBackgroundColor = true
			gui.Config.BackgroundColor = dlg.CurrentColor().Name()
			gui.setupBackgroundColor()
		}
	})
}

func (gui *GUI) connectMenuActions() {
	// File
	gui.actionOpen.OnTriggered(func() {
		gui.openArchive()
	})
	gui.actionClose.OnTriggered(func() {
		gui.Close()
	})
	gui.actionSave_Image.OnTriggered(func() {
		gui.SavePNG()
	})
	gui.action_Quit.OnTriggered(func() {
		gui.Quit()
	})

	// Edit
	gui.action_Preferences.OnTriggered(func() {
		gui.showPreferences()
	})

	// Navigation
	gui.actionFirst_page.OnTriggered(gui.FirstPage)
	gui.actionLast_page.OnTriggered(gui.LastPage)
	gui.actionPrevious_page.OnTriggered(gui.PreviousPage)
	gui.actionNext_page.OnTriggered(gui.NextPage)
	gui.actionSkip_forward.OnTriggered(gui.SkipForward)
	gui.actionSkip_backward.OnTriggered(gui.SkipBackward)
	gui.actionPrevious_archive.OnTriggered(func() {
		gui.PreviousArchive()
	})
	gui.actionNext_archive.OnTriggered(func() {
		gui.NextArchive()
	})
	gui.actionPrevious_Scene.OnTriggered(gui.PreviousScene)
	gui.actionNext_scene.OnTriggered(gui.NextScene)
	gui.actionGo_to_page.OnTriggered(func() {
		gui.RunGoToDialog()
	})

	// View
	gui.actionBest_fit.OnTriggered(func() {
		gui.SetZoomMode("BestFit")
	})
	gui.actionZoom_25.OnTriggered(func() { gui.setZoomLevel(25) })
	gui.actionZoom_33.OnTriggered(func() { gui.setZoomLevel(33) })
	gui.actionZoom_50.OnTriggered(func() { gui.setZoomLevel(50) })
	gui.actionZoom_66.OnTriggered(func() { gui.setZoomLevel(66) })
	gui.actionZoom_75.OnTriggered(func() { gui.setZoomLevel(75) })
	gui.actionZoom_100.OnTriggered(func() { gui.setZoomLevel(100) })
	gui.actionZoom_125.OnTriggered(func() { gui.setZoomLevel(125) })
	gui.actionZoom_150.OnTriggered(func() { gui.setZoomLevel(150) })
	gui.actionZoom_200.OnTriggered(func() { gui.setZoomLevel(200) })
	gui.actionZoom_300.OnTriggered(func() { gui.setZoomLevel(300) })
	gui.actionZoom_400.OnTriggered(func() { gui.setZoomLevel(400) })
	gui.actionZoom_800.OnTriggered(func() { gui.setZoomLevel(800) })
	gui.actionFit_to_width.OnTriggered(func() {
		gui.SetZoomMode("FitToWidth")
	})
	gui.actionFit_to_height.OnTriggered(func() {
		gui.SetZoomMode("FitToHeight")
	})
	gui.actionRotate_0.OnTriggered(func() { gui.setRotation(0) })
	gui.actionRotate_90.OnTriggered(func() { gui.setRotation(90) })
	gui.actionRotate_180.OnTriggered(func() { gui.setRotation(180) })
	gui.actionRotate_270.OnTriggered(func() { gui.setRotation(270) })
	gui.actionFullscreen.OnTriggered(func() {
		gui.SetFullscreen(!gui.Config.Fullscreen)
	})
	gui.actionSeamless_mode.OnTriggered(func() {
		gui.SetSeamless(!gui.Config.Seamless)
	})
	gui.actionRandom_ordering.OnTriggered(func() {
		gui.SetRandom(!gui.Config.Random)
	})
	gui.actionManga_mode.OnTriggered(func() {
		gui.SetMangaMode(!gui.Config.MangaMode)
	})
	gui.actionDouble_page.OnTriggered(func() {
		gui.SetDoublePage(!gui.Config.DoublePage)
	})
	gui.actionH_flip.OnTriggered(func() {
		gui.SetHFlip(!gui.Config.HFlip)
	})
	gui.actionV_flip.OnTriggered(func() {
		gui.SetVFlip(!gui.Config.VFlip)
	})
	gui.actionShrink_large_images.OnTriggered(func() {
		gui.SetShrink(!gui.Config.Shrink)
	})
	gui.actionEnlarge_small_images.OnTriggered(func() {
		gui.SetEnlarge(!gui.Config.Enlarge)
	})

	// Help
	gui.actionAbout.OnTriggered(func() {
		gui.showAbout()
	})

	// Bookmarks
	gui.actionAdd_bookmark.OnTriggered(func() {
		gui.AddBookmark()
	})
}

func (gui *GUI) setupCheckableActions() {
	actions := []struct {
		*qt6.QAction
		bool
	}{
		{gui.actionBest_fit, true},
		{gui.actionZoom_25, true},
		{gui.actionZoom_33, true},
		{gui.actionZoom_50, true},
		{gui.actionZoom_66, true},
		{gui.actionZoom_75, true},
		{gui.actionZoom_100, true},
		{gui.actionZoom_125, true},
		{gui.actionZoom_150, true},
		{gui.actionZoom_200, true},
		{gui.actionZoom_300, true},
		{gui.actionZoom_400, true},
		{gui.actionZoom_800, true},
		{gui.actionRotate_0, true},
		{gui.actionRotate_90, true},
		{gui.actionRotate_180, true},
		{gui.actionRotate_270, true},
		{gui.actionFit_to_width, true},
		{gui.actionFit_to_height, true},
		{gui.actionFullscreen, true},
		{gui.actionSeamless_mode, true},
		{gui.actionRandom_ordering, true},
		{gui.actionManga_mode, true},
		{gui.actionDouble_page, true},
		{gui.actionH_flip, true},
		{gui.actionV_flip, true},
		{gui.actionShrink_large_images, true},
		{gui.actionEnlarge_small_images, true},
	}
	for _, a := range actions {
		a.QAction.SetCheckable(true)
	}
}

func (gui *GUI) setupActionShortcuts() {
	// Menu/toolbar actions default to Qt::WidgetShortcut, which is scoped to the
	// menubar; when it's hidden in fullscreen mode those shortcuts go inactive.
	// Promote every action to a window-wide shortcut so they still fire there.
	actions := []*qt6.QAction{
		gui.actionOpen, gui.actionClose, gui.actionSave_Image, gui.action_Quit,
		gui.action_Preferences,
		gui.actionShrink_large_images, gui.actionEnlarge_small_images,
		gui.actionBest_fit, gui.actionFit_to_width,
		gui.actionFit_to_height, gui.actionFullscreen, gui.actionRandom_ordering,
		gui.actionV_flip, gui.actionH_flip, gui.actionManga_mode, gui.actionDouble_page,
		gui.actionPrevious_page, gui.actionNext_page, gui.actionFirst_page,
		gui.actionLast_page, gui.actionGo_to_page, gui.actionAdd_bookmark, gui.actionAbout, gui.actionZoom_100,
	}
	for _, a := range actions {
		a.SetShortcutContext(qt6.WindowShortcut)
		gui.MainWindow.AddAction(a)
	}
}

func (gui *GUI) setupZoomActionGroup() {
	// One exclusive group spans every zoom choice (the three fit modes plus
	// the manual levels), so exactly one is checked at a time. Rotation has
	// its own exclusive group of the four angles.
	group := qt6.NewQActionGroup(gui.MainWindow.QWidget.QObject)
	group.AddAction(gui.actionBest_fit)
	group.AddAction(gui.actionFit_to_width)
	group.AddAction(gui.actionFit_to_height)

	gui.zoomLevelActions = []*qt6.QAction{
		gui.actionZoom_25, gui.actionZoom_33, gui.actionZoom_50,
		gui.actionZoom_66, gui.actionZoom_75, gui.actionZoom_100,
		gui.actionZoom_125, gui.actionZoom_150, gui.actionZoom_200,
		gui.actionZoom_300, gui.actionZoom_400, gui.actionZoom_800,
	}
	for _, a := range gui.zoomLevelActions {
		group.AddAction(a)
	}

	rGroup := qt6.NewQActionGroup(gui.MainWindow.QWidget.QObject)
	gui.rotationActions = []*qt6.QAction{
		gui.actionRotate_0, gui.actionRotate_90,
		gui.actionRotate_180, gui.actionRotate_270,
	}
	for _, a := range gui.rotationActions {
		rGroup.AddAction(a)
	}
}

func (gui *GUI) updateRecentFiles(path string) {
	recent := gui.Config.Recent
	for i, r := range recent {
		if r == path {
			recent = append(recent[:i], recent[i+1:]...)
			break
		}
	}
	recent = append([]string{path}, recent...)
	if len(recent) > MaxRecent {
		recent = recent[:MaxRecent]
	}
	gui.Config.Recent = recent
	gui.buildRecentFilesMenu()
}

func (gui *GUI) buildRecentFilesMenu() {
	for i := range recentFileActions {
		recentFileActions[i].Delete()
	}
	recentFileActions = nil

	if len(gui.Config.Recent) == 0 {
		gui.menu_Recent_Files.QWidget.AddAction(gui.action_no_recent_items)
		return
	}

	gui.menu_Recent_Files.QWidget.RemoveAction(gui.action_no_recent_items)

	for _, path := range gui.Config.Recent {
		a := qt6.NewQAction()
		a.SetText(path)
		p := path
		a.OnTriggered(func() {
			gui.LoadArchive(p, false)
		})
		recentFileActions = append(recentFileActions, a)
		gui.menu_Recent_Files.QWidget.AddAction(a)
	}
}

func (gui *GUI) syncUI() {
	// Sync config with UI
	gui.actionEnlarge_small_images.SetChecked(gui.Config.Enlarge)
	gui.actionShrink_large_images.SetChecked(gui.Config.Shrink)
	gui.actionH_flip.SetChecked(gui.Config.HFlip)
	gui.actionV_flip.SetChecked(gui.Config.VFlip)
	gui.actionRandom_ordering.SetChecked(gui.Config.Random)
	gui.actionSeamless_mode.SetChecked(gui.Config.Seamless)
	gui.actionDouble_page.SetChecked(gui.Config.DoublePage)
	gui.actionManga_mode.SetChecked(gui.Config.MangaMode)
	gui.PreferencesUI.OneWideCheckButton.SetChecked(gui.Config.OneWide)
	gui.PreferencesUI.SingleCoverCheckButton.SetChecked(gui.Config.SingleCover)
	gui.PreferencesUI.SmartScrollCheckButton.SetChecked(gui.Config.SmartScroll)
	gui.PreferencesUI.EmbeddedOrientationCheckButton.SetChecked(gui.Config.EmbeddedOrientation)
	gui.PreferencesUI.HideIdleCursorCheckButton.SetChecked(gui.Config.HideIdleCursor)
	gui.PreferencesUI.UseBackgroundColorCheckButton.SetChecked(gui.Config.UseBackgroundColor)
	gui.PreferencesUI.PagesToSkipSpinButton.SetValue(gui.Config.NSkip)
	gui.PreferencesUI.InterpolationComboBoxText.SetCurrentIndex(gui.Config.Interpolation)

	// Set zoom mode action checked state
	switch gui.Config.ZoomMode {
	case "FitToWidth":
		gui.actionFit_to_width.SetChecked(true)
	case "FitToHeight":
		gui.actionFit_to_height.SetChecked(true)
	case "BestFit":
		gui.actionBest_fit.SetChecked(true)
	default: // Manual
		for i, a := range gui.zoomLevelActions {
			a.SetChecked(zoomLevels[i] == gui.Config.ZoomLevel)
		}
	}

	// Set rotation angle checked state
	for i, a := range gui.rotationActions {
		a.SetChecked(rotationLevels[i] == gui.Config.Rotation)
	}

	// Set go-to spin button range
	gui.GoToUI.GoToSpinButton.SetRange(1, 1)
	gui.GoToUI.GoToScrollbar.SetRange(1, 1)
}

func (gui *GUI) setupBackgroundColor() {
	c := qt6.NewQColor()
	defer c.Delete()
	c.SetNamedColor(gui.Config.BackgroundColor)
	brush := qt6.NewQBrush3(c)
	defer brush.Delete()
	gui.PreferencesUI.BackgroundColorButton.SetStyleSheet("background-color: " + c.Name() + ";")

	if gui.Config.UseBackgroundColor {
		gui.Image.Scene().SetBackgroundBrush(brush)
	} else {
		// Default dark background
		brush := qt6.NewQBrush()
		defer brush.Delete()
		gui.Image.Scene().SetBackgroundBrush(brush)
	}
	// Mirror the background on the color button so it shows the current pick.
}

func (gui *GUI) openArchive() {
	imgGlobs := make([]string, len(archive.ImageExtensions))
	for i, e := range archive.ImageExtensions {
		imgGlobs[i] = "*" + e
	}
	sort.Strings(imgGlobs)

	arcGlobs := make([]string, len(archive.ArchiveExtensions))
	for i, e := range archive.ArchiveExtensions {
		arcGlobs[i] = "*" + e
	}
	sort.Strings(arcGlobs)

	allGlobs := append([]string{}, imgGlobs...)
	allGlobs = append(allGlobs, arcGlobs...)

	filter := fmt.Sprintf("All supported (%s);;Images (%s);;Archives (%s);;All Files (*)",
		strings.Join(allGlobs, " "), strings.Join(imgGlobs, " "), strings.Join(arcGlobs, " "))

	res := qt6.QFileDialog_GetOpenFileName4(gui.MainWindow.QWidget, "Open Archive", gui.Config.LastDirectory, filter)
	if res == "" {
		return
	}
	gui.Config.LastDirectory = filepath.Dir(res)
	gui.LoadArchive(res, true)
}

func (gui *GUI) showPreferences() {
	gui.showCursorForDialog(func() {
		gui.PreferencesUI.PreferencesDialog.Exec()
	})
	gc()
}

func (gui *GUI) showAbout() {
	gui.showCursorForDialog(func() {
		gui.AboutUI.AboutDialog.Exec()
	})
	gc()
}

func (gui *GUI) RunGoToDialog() {
	if !gui.Loaded() {
		return
	}

	gui.GoToUI.GoToSpinButton.SetRange(1, gui.State.Archive.Len())
	gui.GoToUI.GoToSpinButton.SetValue(int(gui.State.ArchivePos + 1))
	gui.GoToUI.GoToScrollbar.SetRange(1, gui.State.Archive.Len())

	gui.goToDialogLoadThumbnail()

	res := gui.GoToUI.Dialog.Exec()
	gui.GoToUI.GoToThumbnailImage.Scene().Clear()
	if res == int(qt6.QDialog__Accepted) {
		gui.SetPage(int(gui.GoToUI.GoToSpinButton.Value()) - 1)
	}
	gc()
}

func (gui *GUI) goToDialogLoadThumbnail() {
	if !gui.Loaded() {
		return
	}

	n := gui.GoToUI.GoToSpinButton.Value() - 1

	img, err := gui.State.Archive.Load(int(n), gui.Config.EmbeddedOrientation)
	if err != nil {
		gui.ShowError(err.Error())
		return
	}
	defer deleteQImage(img)

	thumbScene := gui.GoToUI.GoToThumbnailImage.Scene()
	thumbScene.Clear()

	// Fit to the full available space in the view (best fit). Before the
	// dialog is laid out the geometry isn't ready, so fall back to a fixed
	// size; the resize handler re-fits once it's shown.
	vw, vh := gui.GoToUI.GoToThumbnailImage.Viewport().Geometry().Width(), gui.GoToUI.GoToThumbnailImage.Viewport().Geometry().Height()
	fw, fh := vw, vh
	if fw <= 0 || fh <= 0 {
		fw, fh = ThumbnailSize, ThumbnailSize
	}

	w, h := img.Width(), img.Height()
	scaledW, scaledH := fit(w, h, fw, fh)
	if scaledW == 0 || scaledH == 0 {
		return
	}
	scaled := img.Scaled(scaledW, scaledH)
	defer deleteQImage(scaled)

	pix := qt6.QPixmap_FromImage(scaled)
	defer deleteQPixmap(pix)
	item := thumbScene.AddPixmap(pix)

	// Center the item when it fits a dimension (offx/offy), and keep the
	// scene rect >= the viewport so QGraphicsView never re-centers it.
	offx, offy := max(0, (vw-scaledW)/2), max(0, (vh-scaledH)/2)
	item.SetPos2(float64(offx), float64(offy))
	sceneW, sceneH := max(scaledW, vw), max(scaledH, vh)
	gui.GoToUI.GoToThumbnailImage.SetSceneRect2(0, 0, float64(sceneW), float64(sceneH))
}

// Bookmark action management

func (gui *GUI) AddBookmark() {
	defer gui.RebuildBookmarksMenu()

	for i := range gui.Config.Bookmarks {
		b := &gui.Config.Bookmarks[i]
		if b.Path == gui.State.ArchivePath {
			b.Page = uint(gui.State.ArchivePos + 1)
			b.TotalPages = uint(gui.State.Archive.Len())
			b.Added = time.Now()
			return
		}
	}

	gui.Config.Bookmarks = append(gui.Config.Bookmarks, Bookmark{
		Path:       gui.State.ArchivePath,
		TotalPages: uint(gui.State.Archive.Len()),
		Page:       uint(gui.State.ArchivePos + 1),
		Added:      time.Now(),
	})
}

func (gui *GUI) RebuildBookmarksMenu() {
	for i := range bookmarkActionsList {
		bookmarkActionsList[i].Delete()
	}
	bookmarkActionsList = nil

	for i := range gui.Config.Bookmarks {
		bookmark := &gui.Config.Bookmarks[i]
		base := filepath.Base(bookmark.Path)
		label := fmt.Sprintf("%s (%d/%d)", base, bookmark.Page, bookmark.TotalPages)

		a := qt6.NewQAction()
		a.SetText(label)
		a.OnTriggered(func() {
			if gui.State.ArchivePath != bookmark.Path {
				gui.LoadArchive(bookmark.Path, false)
			}
			gui.SetPage(int(bookmark.Page) - 1)
		})
		bookmarkActionsList = append(bookmarkActionsList, a)
		gui.menuBookmarks.QWidget.AddAction(a)
	}
}
