//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
)

type TermSize struct {
	WSRow, WSCol       uint
	WSXPixel, WSYPixel uint
}

func GetTermSize() (TermSize, error) {
	// potentially we can do this more easily with github.com/nsf/termbox-go
	var err error
	var f *os.File
	if f, err = os.OpenFile("/dev/tty", unix.O_NOCTTY|unix.O_CLOEXEC|unix.O_NDELAY|unix.O_RDWR, 0666); err == nil {
		// based on snippet: https://sw.kovidgoyal.net/kitty/graphics-protocol/#getting-the-window-size
		// see also: https://github.com/influxdata/docker-client/blob/916439ee97e9a5c5067949d0705774cb33d5c780/pkg/term/winsize.go
		defer f.Close()
		var sz *unix.Winsize
		if sz, err = unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ); err == nil {
			if sz.Xpixel == 0 && sz.Ypixel == 0 && os.Getenv("TERM") == "xterm-kitty" {
				// TODO: log that we are using a fallback?
				// TODO: test that the fallback is even working
				// "You can also use the CSI t escape code to get the screen size."
				state, err := terminal.MakeRaw(int(f.Fd()))
				if err == nil {
					defer terminal.Restore(int(f.Fd()), state) // ignoring error
					// hack to populate the size on kitty
					fmt.Printf("\033[14t")
					// total hack: try to see if we get a reply (ideally with timeout, but initial version can be without it; maybe context.WithTimeout() can be used somehow?)
					b := make([]byte, 1)
					_, err := os.Stdin.Read(b)
					if err == nil && b[0] == 033 { // TODO: log if err != nil?
						// try reading the rest then:
						// <ESC>[4;<height>;<width>t
						reader := bufio.NewReader(os.Stdin)
						s, err := reader.ReadString('t') // reads the remainder after escape, including t ; TODO limit to N characters, add timeout
						if err == nil {
							re := regexp.MustCompile(`\[4;(\d+);(\d+)t`)
							matches := re.FindStringSubmatch(s)
							if len(matches) == 3 { // 0: full match, 1: height, 2: width
								heightStr := matches[1]
								widthStr := matches[2]

								height, errH := strconv.Atoi(heightStr)
								width, errW := strconv.Atoi(widthStr)

								if errH == nil && errW == nil {
									sz.Xpixel = uint16(width)
									sz.Ypixel = uint16(height)
								}
								// TODO: log errors?
							}
						}
					}
				}

			}
			return TermSize{WSRow: uint(sz.Row), WSCol: uint(sz.Col), WSXPixel: uint(sz.Xpixel), WSYPixel: uint(sz.Ypixel)}, nil
		}
	}
	var w, h int
	if w, h, err = terminal.GetSize(0); err == nil { // or int(os.Stdin.Fd())
		return TermSize{WSRow: uint(w), WSCol: uint(h)}, nil
	}
	return TermSize{}, err
}
