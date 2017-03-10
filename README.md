# JM

A simple and small terminal-based file manager that tries to be friendly.

## Usage
    jm [left path] [right path] [flags]

    Flags
      -c, --config string   config file (default "$HOME/.jm")

## Docs
The interface displays two side by side panels, each with the contents of a directory. You can toggle between panels with the TAB key. Exit with the Q or ESC keys.

The Up & Down arrows, and Page Up & Page Down keys let you navigate up and down the files in the current panel.

Left arrow goes to the parent directory, Right arrow enters the directory the cursor is on.

<<File operations coming soon>>

jm uses a configuration file by default in $HOME/.jm, to remember the state of your last session.

## Technical

jm is written in [Go](https://golang.org/) and runs on Windows, Linux and OSX. Besides the go standard library, it uses a number of wonderful packages written by others:

- Rune utils: [github.com/mattn/go-runewidth](https://github.com/mattn/go-runewidth)
- Comprehensive $HOME: [github.com/mitchellh/go-homedir](https://github.com/mitchellh/go-homedir)
- Terminal I/O: [github.com/nsf/termbox-go](https://github.com/nsf/termbox-go)
- Command line: [github.com/spf13/cobra](https://github.com/spf13/cobra)
- Config: [github.com/spf13/viper](https://github.com/spf13/viper)
- File size pretty printing: [code.cloudfoundry.org/bytefmt](https://code.cloudfoundry.org/bytefmt)

## License

Copyright 2017 Javier Arevalo <jare@iguanademos.com>

jm is released under the Apache 2.0 license. See [LICENSE.txt](LICENSE.txt)
