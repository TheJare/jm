// Copyright 2017 Javier Arevalo <jare@iguanademos.com>

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/mitchellh/go-homedir"
	"github.com/nsf/termbox-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"code.cloudfoundry.org/bytefmt"
)

// ------------------

func tbprint(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x += runewidth.RuneWidth(c)
	}
}

func tbprintw(x, y, w int, fg, bg termbox.Attribute, msg string) {
	w += x
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x += runewidth.RuneWidth(c)
		if x >= w {
			break
		}
	}
}

func fill(x, y, w, h int, cell termbox.Cell) {
	for ly := 0; ly < h; ly++ {
		for lx := 0; lx < w; lx++ {
			termbox.SetCell(x+lx, y+ly, cell.Ch, cell.Fg, cell.Bg)
		}
	}
}

// ------------------

// ByFolderThenName implements sort.Interface for []os.FileInfo based on
// leaving folders first and otherwise sorting by case-insensitive name
type ByFolderThenName []os.FileInfo

func (a ByFolderThenName) Len() int      { return len(a) }
func (a ByFolderThenName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByFolderThenName) Less(i, j int) bool {
	if a[i].IsDir() == a[j].IsDir() {
		return strings.ToLower(a[i].Name()) < strings.ToLower(a[j].Name())
	}
	return a[i].IsDir()
}

// Panel contains the state of one panel
type Panel struct {
	Cwd      string
	Entries  []os.FileInfo
	Top      int
	Cursor   int
	Selected map[int]bool
}

// NewPanel creates and initializes a new panel given a directory
// and an entry to set the cursor at
func NewPanel(cwd string, cursor string) (*Panel, error) {
	p := &Panel{}
	err := p.Reset(cwd, cursor)
	return p, err
}

// Reset reinitializes a panel to given a directory
// and an entry to set the cursor at
func (p *Panel) Reset(cwd string, cursor string) error {
	entries, err := ioutil.ReadDir(cwd)
	sort.Sort(ByFolderThenName(entries))
	p.Cwd = cwd
	p.Entries = entries
	p.Top = 0
	p.Cursor = 0
	p.Selected = make(map[int]bool)
	for i, v := range p.Entries {
		if v.Name() == cursor {
			p.Cursor = i
			break
		}
	}
	return err
}

func permc(c string, mode os.FileMode) string {
	if mode != 0 {
		return c[:1]
	}
	if len(c) > 1 {
		return c[1:]
	}
	return " "
}

func permissions(mode os.FileMode) string {
	s := ""
	s = s + permc("d.", mode&os.ModeDir)
	s = s + permc("r-", mode&(1<<8))
	s = s + permc("w-", mode&(1<<7))
	s = s + permc("x-", mode&(1<<6))
	s = s + permc("r-", mode&(1<<5))
	s = s + permc("w-", mode&(1<<4))
	s = s + permc("x-", mode&(1<<3))
	s = s + permc("r-", mode&(1<<2))
	s = s + permc("w-", mode&(1<<1))
	s = s + permc("x-", mode&1)
	return s
}

// Render draws a panel at the given position, restricted
// to the given dimensions, and with different colors if it's
// the active panel
func (p *Panel) Render(x, w, h int, active bool) {
	for i := 0; i < (len(p.Entries)-p.Top) && i < h; i++ {
		var fg, bg termbox.Attribute = termbox.ColorDefault, termbox.ColorDefault
		n := i + p.Top
		e := p.Entries[n]
		if active {
			if p.Selected[n] {
				if n == p.Cursor {
					bg = termbox.ColorCyan
					fg = termbox.ColorBlack
				} else {
					bg = termbox.ColorBlue
				}
			} else if n == p.Cursor {
				bg = termbox.ColorGreen
				fg = termbox.ColorBlack
			}
		} else {
			if p.Selected[n] {
				bg = termbox.ColorBlue
			}
		}
		if e.IsDir() {
			if fg == termbox.ColorBlack {
				fg = termbox.ColorBlue
			} else {
				fg = termbox.ColorYellow
			}
		}
		fn := e.Name()
		if e.IsDir() {
			fn = fn + string(os.PathSeparator)
		}
		if p.Selected[n] {
			fn = "*" + fn
		}
		if w > 50 {
			fn = fmt.Sprintf("%-*.*s %-*.*s %*.*s", w-32, w-32, fn, 20, 20, e.ModTime().Format("02 Jan 2006 15:04:05"), 10, 10, bytefmt.ByteSize(uint64(e.Size())))
		} else if w > 30 {
			fn = fmt.Sprintf("%-*.*s %*.*s", w-11, w-11, fn, 10, 10, bytefmt.ByteSize(uint64(e.Size())))
		}

		tbprintw(x, i, w, fg, bg, fn)
	}
	fn := p.Cwd
	if p.Cursor < len(p.Entries) {
		e := p.Entries[p.Cursor]
		fn = fmt.Sprintf("%s %s %s %s %d", p.Cwd, permissions(e.Mode()), e.ModTime().Format("Mon, 02 Jan 2006 15:04:05"), e.Name(), e.Size())
	}
	tbprintw(x, h, w, termbox.ColorWhite, termbox.ColorRed, fn)
}

// ClampPos limits the state of the panel to valid values
func (p *Panel) ClampPos(h int) {
	if p.Cursor <= 0 {
		p.Cursor = 0
	} else if p.Cursor >= len(p.Entries) {
		p.Cursor = len(p.Entries) - 1
	}
	if p.Cursor < p.Top {
		p.Top = p.Cursor
	} else if p.Cursor >= p.Top+h {
		p.Top = p.Cursor - h + 1
	}
	if p.Top >= len(p.Entries) {
		p.Top = len(p.Entries) - 1
	}
	if p.Top < 0 {
		p.Top = 0
	}
}

// ------------------

var cursorCache = make(map[string]string)

func getCachedCursor(key string) string {
	if runtime.GOOS != "unix" {
		key = strings.ToLower(key)
	}
	return cursorCache[key]
}

func setCachedCursor(key string, val string) {
	if runtime.GOOS != "unix" {
		key = strings.ToLower(key)
	}
	cursorCache[key] = val
}

// ------------------

var lp, rp *Panel
var ap *Panel

func redrawAll() int {
	const coldef = termbox.ColorDefault
	termbox.Clear(coldef, coldef)
	w, h := termbox.Size()

	midx := w / 2

	lp.ClampPos(h - 2)
	rp.ClampPos(h - 2)

	fill(midx, 0, 1, h-2, termbox.Cell{Ch: '|', Fg: termbox.ColorRed})
	fill(0, h-2, w, 1, termbox.Cell{Ch: '─', Bg: termbox.ColorRed})
	lp.Render(0, midx-1, h-2, lp == ap)
	rp.Render(midx+1, w-midx-1, h-2, rp == ap)
	tbprint(0, h-1, coldef, coldef, "[ESC,q quit] [TAB switch] [SPC select] [ARROWS nav]")
	termbox.Flush()

	return h - 2
}

// ------------------

var configFile string

type config struct {
	LeftPath    string
	RightPath   string
	CursorCache map[string]string
}

func writeConfig() error {
	var c config
	c.LeftPath = lp.Cwd
	c.RightPath = rp.Cwd
	c.CursorCache = cursorCache

	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	f, err := os.Create(configFile)
	if err != nil {
		return err
	}

	defer f.Close()

	f.WriteString(string(b))

	return nil
}

// ------------------

func run(ld, rd string) {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	lp, _ = NewPanel(ld, getCachedCursor(ld))
	rp, _ = NewPanel(rd, getCachedCursor(rd))
	ap = lp

	pagesize := redrawAll()
mainloop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyEsc:
				break mainloop
			case termbox.KeyTab:
				if ap == lp {
					ap = rp
				} else {
					ap = lp
				}
			case termbox.KeyArrowUp:
				ap.Cursor--
			case termbox.KeyArrowDown:
				ap.Cursor++
			case termbox.KeyPgup:
				ap.Cursor -= pagesize
			case termbox.KeyPgdn:
				ap.Cursor += pagesize
			case termbox.KeyArrowLeft:
				if ap.Cursor < len(ap.Entries) {
					setCachedCursor(ap.Cwd, ap.Entries[ap.Cursor].Name())
					setCachedCursor(filepath.Dir(ap.Cwd), filepath.Base(ap.Cwd))
				}
				ap.Reset(filepath.Dir(ap.Cwd), getCachedCursor(filepath.Dir(ap.Cwd)))
			case termbox.KeyArrowRight:
				if ap.Cursor < len(ap.Entries) && ap.Entries[ap.Cursor].IsDir() {
					setCachedCursor(ap.Cwd, ap.Entries[ap.Cursor].Name())
					n := filepath.Join(ap.Cwd, ap.Entries[ap.Cursor].Name())
					ap.Reset(n, getCachedCursor(n))
				}
			case termbox.KeySpace:
				if ap.Cursor < len(ap.Entries) {
					if ap.Selected[ap.Cursor] {
						delete(ap.Selected, ap.Cursor)
					} else {
						ap.Selected[ap.Cursor] = true
					}
				}
			default:
				switch ev.Ch {
				case 'q':
					break mainloop
				case 'e':
					op := lp
					if ap == lp {
						op = rp
					}
					for k := range ap.Selected {
						f := ap.Entries[k]
						RunCommand("echo", filepath.Join(ap.Cwd, f.Name()), op.Cwd)
					}
				}
			}
		case termbox.EventError:
			panic(ev.Err)
		}
		pagesize = redrawAll()
	}
	writeConfig()
}

var rootCmd = &cobra.Command{
	Use:   "jm [left path] [right path]",
	Short: "jm is a small terminal-based file manager",
	Long:  `A simple and small terminal-based file manager that tries to be friendly`,
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetConfigFile(configFile)
		viper.SetConfigType("json")
		err := viper.ReadInConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error config file: %s \n", err)
		}
		cursorCache = viper.GetStringMapString("CursorCache")
		fmt.Fprintf(os.Stderr, "cursorCache: %#v\n", cursorCache)
		if len(args) < 2 {
			d := viper.GetString("LeftPath")
			if d == "" {
				d, _ = os.Getwd()
			}
			args = append(args, d)
		}
		if len(args) < 2 {
			d := viper.GetString("RightPath")
			if d == "" {
				d, _ = homedir.Dir()
			}
			args = append(args, d)
		}
		run(args[0], args[1])
	},
}

func main() {
	viper.SetDefault("LeftPath", "")
	viper.SetDefault("RightPath", "")
	viper.SetDefault("CursorCache", map[string]string{})
	home, _ := homedir.Dir()
	configFile = filepath.Join(home, ".jm")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", configFile, "config file")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}
