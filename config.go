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

package main

import (
	"encoding/json"
	"os"
)

const (
	ConfigDir  = ".config/gomics" // relative to user's home
	ConfigFile = "config"         // relative to config dir
	ImageDir   = "images"         // relative to config dir
)

type Config struct {
	ZoomMode            string
	Enlarge             bool
	Shrink              bool
	LastDirectory       string
	Fullscreen          bool
	WindowWidth         int
	WindowHeight        int
	NSkip               int
	Random              bool
	Seamless            bool
	HFlip               bool
	VFlip               bool
	DoublePage          bool
	MangaMode           bool
	OneWide             bool
	EmbeddedOrientation bool
	Interpolation       int
	ImageDiffThres      float32
	SceneScanSkip       int
	SmartScroll         bool
	Bookmarks           []Bookmark
	HideIdleCursor      bool
}

func (c *Config) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err = d.Decode(c); err != nil {
		return err
	}

	return nil
}

func (c *Config) Save(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

func (c *Config) Defaults() {
	c.ZoomMode = "BestFit"
	c.Shrink = true
	c.Enlarge = false
	c.WindowWidth = 640
	c.WindowHeight = 480
	c.NSkip = 10
	c.Seamless = true
	c.Interpolation = 2
	c.EmbeddedOrientation = true
	c.ImageDiffThres = 0.4
	c.SceneScanSkip = 5
	c.SmartScroll = true
	c.HideIdleCursor = true
}
