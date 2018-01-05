# Gomics

A gtk3 comic and image archive viewer written in Go, freely available under GPL v3+.

## Screenshot

![Screenshot](https://raw.githubusercontent.com/salviati/gomics/master/screenshot.png)

## Features

- Reads zip (and cbz) files directly, without writing to disk at all.
- Small memory footprint.
- Double and single-page mode.
- Comic and manga-mode (left-to-right and right-to-left page order).
- Basic scaling modes: original size, fit to height, fit to width, best fit.
- Image effects: horizontal flip, vertical flip.
- Bookmarks.
- Randomized page ordering.
- Can navigate between CG scenes (based on image similarity).

## Requirements

gtk3, gdk-pixbuf2, glib2, go-bindata.

## Installation
Run ./make.sh.

Arch Linux users can alternatively install the AUR package `gomics-git`.

## Shortcuts
* Up/down or page up/down or mouse wheel up/down: previous/next page.
* Ctrl + up/down: previous/next archive.
* Left/right: skip backward/forward (# of pages is configurable).
* Ctrl + left/right: previous/next scene (useful for CG archives).
* Scroll image: shift + direction keys (not implemented at the moment).

## License
This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program. If not, see http://www.gnu.org/licenses/.
