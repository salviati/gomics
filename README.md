# Gomics

A Qt6 image and image archive (manga/comic/etc CBZ/ZIP archives) viewer written in Go, freely available under GPL v3+.

## Screenshot

![Screenshot](https://raw.githubusercontent.com/salviati/gomics/master/screenshot.png)

## Features

- Reads zip (and cbz) files directly, without writing to disk/tmpfs at all.
- Double and single-page mode.
- Comic and manga-mode (left-to-right and right-to-left page order).
- Smart scrolling.
- Basic scaling modes: manual zoom, fit to height, fit to width, best fit.
- Image effects: horizontal flip, vertical flip, rotate.
- Bookmarks.
- Randomized page ordering.
- Can navigate between CG scenes (based on image similarity).

## Requirements

qt6. qt6-imageformats and kimageformats for additional image format support. For compiling from source, go is also required.

## Installation
Run `./make.sh`.

Arch Linux users can alternatively install the AUR package `gomics-git`.

## Controls

### Keyboard

| Key                   | Description                                        |
|-----------------------|----------------------------------------------------|
| Up/Down, Page Up/Down | Next/previous image                                |
| Left/Right            | Skip backward/forward (by configurable # of pages) |
| Ctrl+Page Up/Down     | Next/previous archive                              |
| Ctrl+Left/Right       | Previous/next scene (useful for CG archives)       |
| Shift+Direction Keys  | Scroll the current image                           |
| Home                  | Jump to first image                                |
| End                   | Jump to last image                                 |
| G                     | Go to selected image                               |
| F                     | Toggle fullscreen                                  |
| D                     | Toggle double-page mode                            |
| M                     | Toggle manga mode (right-to-left)                  |
| S                     | Shrink large images                                |
| E                     | Enlarge small images                               |
| B, Num Pad *          | Best fit                                           |
| W                     | Fit to width                                       |
| H                     | Fit to height                                      |
| Num Pad +/-           | Manual zoom (in/out)                               |
| O, Num Pad /          | Original size (100% zoom)                          |
| V                     | Vertical flip                                      |
| Ctrl+V                | Horizontal flip                                    |
| R                     | Rotate (counter-clockwise)                         |
| Shift+R               | Rotate (clockwise)                                 |
| Ctrl+R                | Toggle random ordering                             |
| Ctrl+Q                | Quit                                               |
| Ctrl+O                | Open                                               |
| Ctrl+S                | Save current image as                              |
| Ctrl+B                | Bookmark the current image                         |
| Ctrl+P                | Open preferences window                            |
| F1                    | About                                              |

### Mouse

| Action            | Description              |
|-------------------|--------------------------|
| Mouse drag        | Scroll the current image |
| Mouse wheel       | Next/previous image      |
| Mouse wheel+Ctrl  | Zoom in/out              |



## License

### Gomics

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program. If not, see http://www.gnu.org/licenses/.

### Ubunchu! images

Ubunchu! artwork by Hiroshi Seo, licensed under Creative Commons: Attribution-NonCommercial-ShareAlike 2.1.
