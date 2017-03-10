package main

import (
    "fmt"
    "os"
    "io/ioutil"
    "path/filepath"

    "github.com/mattn/go-runewidth"
    "github.com/nsf/termbox-go"
//    "unicode/utf8"
    "github.com/mitchellh/go-homedir"
    "code.cloudfoundry.org/bytefmt"
)

type Panel struct {
    Cwd string
    Entries []os.FileInfo
    Top int
    Cursor int
    Selected map[int]bool
}

func NewPanel(cwd string) (*Panel, error) {
    p := &Panel{}
    err := p.Reset(cwd, cursorCache[cwd])
    return p, err
}

func (p *Panel) Reset(cwd string, cursor string) error {
    entries, err := ioutil.ReadDir(cwd)
    p.Cwd = cwd
    p.Entries = entries
    p.Top = 0
    p.Cursor = 0
    p.Selected = make(map[int]bool)
    for i,v := range p.Entries {
        if v.Name() == cursor {
            p.Cursor = i
            break;
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
    s = s + permc("d.", mode & os.ModeDir)
    s = s + permc("r-", mode & (1 << 8))
    s = s + permc("w-", mode & (1 << 7))
    s = s + permc("x-", mode & (1 << 6))
    s = s + permc("r-", mode & (1 << 5))
    s = s + permc("w-", mode & (1 << 4))
    s = s + permc("x-", mode & (1 << 3))
    s = s + permc("r-", mode & (1 << 2))
    s = s + permc("w-", mode & (1 << 1))
    s = s + permc("x-", mode & 1)
    return s
}

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

func (p *Panel) ClampPos(h int) {
    if p.Cursor <= 0 {
        p.Cursor = 0
    } else if p.Cursor >= len(p.Entries) {
        p.Cursor = len(p.Entries)-1
    }
    if p.Cursor < p.Top {
        p.Top = p.Cursor
    } else if p.Cursor >= p.Top + h {
        p.Top = p.Cursor - h + 1
    }
    if p.Top >= len(p.Entries) {
        p.Top = len(p.Entries)-1
    }
    if p.Top < 0 {
        p.Top = 0
    }
}

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

var lp, rp *Panel
var ap *Panel

var cursorCache = make(map[string]string)

func redraw_all() int {
	const coldef = termbox.ColorDefault
	termbox.Clear(coldef, coldef)
	w, h := termbox.Size()

	midx := w / 2

    lp.ClampPos(h-2)
    rp.ClampPos(h-2)

    //fill(0, 0,  midx-1, h-2, termbox.Cell{Ch: 'L'})
    //fill(midx+1, 0,  midx-1, h-2, termbox.Cell{Ch: 'R'})
    fill(midx, 0, 1, h-2, termbox.Cell{Ch: '|', Fg: termbox.ColorRed})
    fill(0, h-2, w, 1, termbox.Cell{Ch: '─', Bg: termbox.ColorRed})
    lp.Render(0, midx-1, h-2, lp == ap)
    rp.Render(midx+1, w-midx-1, h-2, rp == ap)
	tbprint(0, h-1, coldef, coldef, "[ESC,q quit] [TAB switch] [SPC select] [ARROWS nav]")
	termbox.Flush()

    return h-2
}

func main() {
    err := termbox.Init()
    if err != nil {
        panic(err)
    }
    defer termbox.Close()
    termbox.SetInputMode(termbox.InputEsc)

    wd, _ := os.Getwd()
    hd, _ := homedir.Dir()
    lp, _ = NewPanel(wd)
    rp, _ = NewPanel(hd)
    ap = lp

    pagesize := redraw_all()
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
                    cursorCache[ap.Cwd] = ap.Entries[ap.Cursor].Name()
                    cursorCache[filepath.Dir(ap.Cwd)] = filepath.Base(ap.Cwd)
                }
                ap.Reset(filepath.Dir(ap.Cwd), cursorCache[filepath.Dir(ap.Cwd)])
            case termbox.KeyArrowRight:
                if ap.Cursor < len(ap.Entries) && ap.Entries[ap.Cursor].IsDir() {
                    cursorCache[ap.Cwd] = ap.Entries[ap.Cursor].Name()
                    n := filepath.Join(ap.Cwd, ap.Entries[ap.Cursor].Name())
                    ap.Reset(n, cursorCache[n])
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
                }
			}
        case termbox.EventError:
            panic(ev.Err)
        }
        pagesize = redraw_all()
    }
}