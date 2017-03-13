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
	"unicode"

	"github.com/mattn/go-runewidth"
	"github.com/mitchellh/go-homedir"
	"github.com/nsf/termbox-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"code.cloudfoundry.org/bytefmt"
)

// ------------------

func tbprint(x, y int, fg, bg termbox.Attribute, msg string) int {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x += runewidth.RuneWidth(c)
	}
	return x
}

func tbprintw(x, y, w int, fg, bg termbox.Attribute, msg string) int {
	w += x
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x += runewidth.RuneWidth(c)
		if x >= w {
			break
		}
	}
	return x
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

// Refresh reinitializes a panel with its directory's contents,
// keeping the current cursor and selection if possible
func (p *Panel) Refresh() error {
	if len(p.Entries) == 0 {
		return p.Reset(p.Cwd, "")
	}
	var selection = make(map[string]bool)
	top := p.Top
	cursor := p.Entries[p.Cursor].Name()
	cursorIndex := p.Cursor
	for k := range p.Selected {
		selection[p.Entries[k].Name()] = true
	}
	err := p.Reset(p.Cwd, cursor)
	if len(p.Entries) == 0 {
		return err
	}
	p.Top = top
	if err != nil {
		return err
	}
	if p.Entries[p.Cursor].Name() != cursor {
		if cursorIndex < len(p.Entries) {
			p.Cursor = cursorIndex
		} else {
			p.Cursor = len(p.Entries) - 1
		}
	}
	for k, v := range p.Entries {
		if selection[v.Name()] {
			p.Selected[k] = true
		}
	}
	return nil
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
	//No hidden for you s = s + permc("H.", mode&os.ModeHidden)
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
	nx := tbprintw(x, h, w, termbox.ColorWhite, termbox.ColorRed, p.Cwd)
	if p.Cursor < len(p.Entries) {
		e := p.Entries[p.Cursor]
		fn := fmt.Sprintf("%s %s %d %s", permissions(e.Mode()), e.ModTime().Format("Mon, 02 Jan 2006 15:04:05"), e.Size(), e.Name())
		tbprintw(nx+1, h, w-(nx+1-x), termbox.ColorYellow, termbox.ColorRed, fn)
	}
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
var ap, op *Panel
var status string

var bookmarks map[string]string

func redrawAll() int {
	const coldef = termbox.ColorDefault
	termbox.Clear(coldef, coldef)
	w, h := termbox.Size()

	midx := w / 2

	lp.ClampPos(h - 2)
	rp.ClampPos(h - 2)

	fill(midx, 0, 1, h-2, termbox.Cell{Ch: ' ', Bg: termbox.ColorRed})
	fill(0, h-2, w, 1, termbox.Cell{Ch: ' ', Bg: termbox.ColorRed})
	lp.Render(0, midx, h-2, lp == ap)
	rp.Render(midx+1, w-midx-1, h-2, rp == ap)

	// HACK:
	// Some terminals can't hide the cursor, which may mean that writing to the bottom rightmost
	// character will cause the cursor to wrap to the next line and make the terminal scroll
	// one line. This ruins the display! So use w-1 to prevent writing to that last char.
	if status != "" {
		tbprintw(0, h-1, w-1, termbox.ColorMagenta, coldef, status)
	} else {
		tbprintw(0, h-1, w-1, coldef, coldef, "[ESC,q quit] [TAB switch] [SPC select] [ARROWS nav] [r refresh] [c Copy] [m Move] [DD Delete] [: Shell] [b/B Bookmarks]")
	}
	status = ""
	termbox.Flush()

	return h - 2
}

func redrawStatus(status string) {
	const coldef = termbox.ColorDefault
	w, h := termbox.Size()
	fill(0, h-1, w, 1, termbox.Cell{Ch: ' '})
	tbprint(0, h-1, coldef, coldef, status)
	termbox.Flush()
}

func runShell() string {
	termbox.Close()
	err := RunShell(ap.Cwd)
	termbox.Init()
	if err == nil {
		return ""
	}
	return err.Error()
}

// ------------------

var configFile string

type config struct {
	LeftPath    string
	RightPath   string
	CursorCache map[string]string
	Bookmarks   map[string]string
}

func writeConfig() error {
	var c config
	c.LeftPath = lp.Cwd
	c.RightPath = rp.Cwd
	c.CursorCache = cursorCache
	c.Bookmarks = bookmarks

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

func getCommandArguments() ([]string, string) {
	var src []string
	dst := op.Cwd
	if len(ap.Selected) > 0 {
		for k := range ap.Selected {
			f := ap.Entries[k]
			src = append(src, filepath.Join(ap.Cwd, f.Name()))
		}
	} else if len(ap.Entries) > 0 {
		f := ap.Entries[ap.Cursor]
		src = append(src, filepath.Join(ap.Cwd, f.Name()))
	}
	return src, dst
}

func run(ld, rd string) {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	lp, _ = NewPanel(ld, getCachedCursor(ld))
	rp, _ = NewPanel(rd, getCachedCursor(rd))
	ap, op = lp, rp

	pagesize := redrawAll()
	// Used by commands that work with multiple keystrokes, eg DD
	prefixCommand := ""
mainloop:
	for {
		newCommand := ""
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:

			// Prefix commands
			if prefixCommand == "b" {
				drives, _ := GetDrives()
				drive := unicode.ToUpper(ev.Ch)
				newCwd := ""
				if drives[drive] {
					newCwd = string(drive) + `:\`
				} else if strings.IndexRune("0123456789", ev.Ch) >= 0 {
					newCwd = bookmarks[string(ev.Ch)]
				} else if ev.Ch == '/' {
					newCwd = filepath.VolumeName(ap.Cwd) + string(os.PathSeparator)
				} else if ev.Ch == '~' {
					newCwd, _ = homedir.Dir()
				}
				if newCwd != "" {
					ap.Reset(newCwd, getCachedCursor(newCwd))
				}
				break
			} else if prefixCommand == "B" {
				if strings.IndexRune("0123456789", ev.Ch) >= 0 {
					bookmarks[string(ev.Ch)] = ap.Cwd
				}
				break
			} else if prefixCommand == "D" {
				if ev.Ch == 'D' {
					src, _ := getCommandArguments()
					for i, s := range src {
						redrawStatus(fmt.Sprintf("Deleting file %d/%d: %s", i+1, len(src), s))
						err := CommandDelete(s)
						if err != nil {
							status = status + " " + err.Error()
						}
					}
					ap.Refresh()
					if ap.Cwd == op.Cwd {
						op.Refresh()
					}
				}
				break
			}

			// Regular commands (or detecting prefixes)
			if ev.Key == termbox.KeyEsc || ev.Ch == 'q' || ev.Ch == 'Q' {
				if prefixCommand == "" {
					break mainloop
				}
			} else if ev.Key == termbox.KeyTab {
				ap, op = op, ap
			} else if ev.Key == termbox.KeyArrowUp || ev.Ch == 'k' {
				ap.Cursor--
			} else if ev.Key == termbox.KeyArrowDown || ev.Ch == 'j' {
				ap.Cursor++
			} else if ev.Key == termbox.KeyPgup || ev.Ch == 'u' {
				ap.Cursor -= pagesize
			} else if ev.Key == termbox.KeyPgdn || ev.Ch == 'i' {
				ap.Cursor += pagesize
			} else if ev.Key == termbox.KeyHome || ev.Ch == 'y' {
				ap.Cursor = 0
			} else if ev.Key == termbox.KeyEnd || ev.Ch == 'o' {
				if len(ap.Entries) > 0 {
					ap.Cursor = len(ap.Entries) - 1
				}
			} else if ev.Key == termbox.KeyArrowLeft || ev.Ch == 'h' {
				if ap.Cursor < len(ap.Entries) {
					setCachedCursor(ap.Cwd, ap.Entries[ap.Cursor].Name())
					setCachedCursor(filepath.Dir(ap.Cwd), filepath.Base(ap.Cwd))
				}
				ap.Reset(filepath.Dir(ap.Cwd), getCachedCursor(filepath.Dir(ap.Cwd)))
			} else if ev.Key == termbox.KeyArrowRight || ev.Ch == 'l' {
				if ap.Cursor < len(ap.Entries) && ap.Entries[ap.Cursor].IsDir() {
					setCachedCursor(ap.Cwd, ap.Entries[ap.Cursor].Name())
					n := filepath.Join(ap.Cwd, ap.Entries[ap.Cursor].Name())
					ap.Reset(n, getCachedCursor(n))
				}
			} else if ev.Key == termbox.KeySpace {
				if ap.Cursor < len(ap.Entries) {
					if ap.Selected[ap.Cursor] {
						delete(ap.Selected, ap.Cursor)
					} else {
						ap.Selected[ap.Cursor] = true
					}
				}
			} else if ev.Ch == 'a' {
				if len(ap.Selected) == len(ap.Entries) {
					ap.Selected = make(map[int]bool)
				} else {
					for i := range ap.Entries {
						ap.Selected[i] = true
					}
				}
			} else if ev.Key == termbox.KeyF5 || ev.Ch == 'r' {
				ap.Refresh()
				op.Refresh()
			} else if ev.Ch == ':' {
				status = runShell()
				ap.Refresh()
				op.Refresh()
			} else if ev.Ch == 'b' {
				if prefixCommand == "" {
					if runtime.GOOS == "windows" {
						status = "Press a drive letter or bookmark to cd to"
					} else {
						status = "Press a bookmark to cd to"
					}
					newCommand = "b"
				}
			} else if ev.Ch == 'B' {
				if prefixCommand == "" {
					status = "Press digit to bookmark to"
					newCommand = "B"
				}
			} else if ev.Ch == 'c' {
				if ap.Cwd == op.Cwd {
					// Maybe add a way to duplicate files?
					break
				}
				src, dst := getCommandArguments()
				for i, s := range src {
					redrawStatus(fmt.Sprintf("Copying file %d/%d: %s", i+1, len(src), s))
					err := CommandCopy(s, dst)
					if err != nil {
						status = status + " " + err.Error()
					}
				}
				op.Refresh()
				if ap.Cwd == op.Cwd {
					ap.Refresh()
				}
			} else if ev.Ch == 'm' {
				if ap.Cwd == op.Cwd {
					break
				}
				src, dst := getCommandArguments()
				for i, s := range src {
					redrawStatus(fmt.Sprintf("Moving file %d/%d: %s", i+1, len(src), s))
					err := CommandMove(s, dst)
					if err != nil {
						status = status + " " + err.Error()
					}
				}
				ap.Refresh()
				op.Refresh()
			} else if ev.Ch == 'D' {
				src, _ := getCommandArguments()
				if len(src) > 0 {
					status = fmt.Sprintf("Press D again to confirm deleting %d files (%s)", len(src), strings.Join(src, " "))
					newCommand = "D"
				}
			}
		case termbox.EventError:
			panic(ev.Err)
		}
		pagesize = redrawAll()

		// Keep prefix if one was stored by a command
		if newCommand != "" {
			prefixCommand = newCommand
		} else {
			prefixCommand = ""
		}

	}
	writeConfig()
}

// ------------------

var rootCmd = &cobra.Command{
	Use:   "jm [left path] [right path]",
	Short: "jm is a small terminal-based file manager",
	Long:  `A simple and small terminal-based file manager that tries to be friendly`,
	Run: func(cmd *cobra.Command, args []string) {
		// Read config file
		viper.SetConfigFile(configFile)
		viper.SetConfigType("json")
		// Ignore errors if config file does not exist.
		viper.ReadInConfig()

		cursorCache = viper.GetStringMapString("CursorCache")
		bookmarks = viper.GetStringMapString("Bookmarks")

		// Precedence to paths from the command line
		// Cwd and $HOME as last resort defaults
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
		args[0], _ = filepath.Abs(args[0])
		args[1], _ = filepath.Abs(args[1])

		run(args[0], args[1])
	},
}

func main() {
	viper.SetDefault("LeftPath", "")
	viper.SetDefault("RightPath", "")
	viper.SetDefault("CursorCache", map[string]string{})
	viper.SetDefault("Bookmarks", map[string]string{})
	home, _ := homedir.Dir()
	configFile = filepath.Join(home, ".jm")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", configFile, "config file")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}
