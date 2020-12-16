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

package main

import (
	"fmt"
	"github.com/gotk3/gotk3/gtk"
	"path/filepath"
	"time"
)

var bookmarkMenuItems []*gtk.MenuItem

type Bookmark struct {
	Path       string
	Page       uint
	TotalPages uint
	Added      time.Time
}

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
	for i := range bookmarkMenuItems {
		gui.MenuBookmarks.Remove(bookmarkMenuItems[i])
		bookmarkMenuItems[i].Destroy()
	}
	bookmarkMenuItems = nil
	gc()

	for i := range gui.Config.Bookmarks {
		bookmark := &gui.Config.Bookmarks[i]
		base := filepath.Base(bookmark.Path)
		label := fmt.Sprintf("%s (%d/%d)", base, bookmark.Page, bookmark.TotalPages)
		bookmarkMenuItem, err := gtk.MenuItemNewWithLabel(label)
		if err != nil {
			gui.ShowError(err.Error())
			return
		}
		bookmarkMenuItem.Connect("activate", func() {
			if gui.State.ArchivePath != bookmark.Path {
				gui.LoadArchive(bookmark.Path)
			}
			gui.SetPage(int(bookmark.Page) - 1)
		})
		bookmarkMenuItems = append(bookmarkMenuItems, bookmarkMenuItem)
		gui.MenuBookmarks.Append(bookmarkMenuItem)
	}
	gui.MenuBookmarks.ShowAll()
}
