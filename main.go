package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/jroimartin/gocui"
)

// NumGoroutines is the number of Go Routines.
// const NumGoroutines = 10

var (
	done  = make(chan struct{})
	mu    sync.Mutex // protects words
	pause = false
	wpm   = flag.Int("wpm", 400, "WPM")
	file  = "text.txt"
	idx   = 0
	words = []string{""}
)

var wordRegExp = regexp.MustCompile(`\pL+('\pL+)*|.`)

func reader(g *gocui.Gui) {
	// Parse flags.
	flag.Parse()

	// Get filename.
	file := flag.Arg(0)
	if len(flag.Args()) < 1 {
		file = "text.txt"
	}

	hdl, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer hdl.Close()

	// Calculate delay in milliseconds.
	delay := time.Duration(int(1000.0 / (float32(*wpm) / 60.0)))

	// Process each line.
	scanner := bufio.NewScanner(hdl)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
		line := scanner.Text()
		words = wordRegExp.FindAllString(line, -1)
		// Until Ctrl+C or EOF
		for {
			// Start iterating at index idx (which can be changed at any time)
			v, _ := g.View("reader")
			fmt.Fprintln(v, "\x1b[0;31m", words[idx])
			for _, word := range words[idx:] {
				g.Update(func(g *gocui.Gui) error {
					v, err := g.View("reader")
					v.Clear()
					if err != nil {
						return err
					}
					fmt.Fprintln(v, "\x1b[0;31m", word)
					return nil
				})
				if len(word) > 1 {
					time.Sleep(delay * time.Millisecond)
				}
				for {
					if !pause {
						v, _ := g.View("reader")
						fmt.Fprintln(v, "\x1b[0;31m", word)
						break
					}
					v, _ := g.View("reader")
					fmt.Fprintln(v, "\x1b[0;31m", word)
					time.Sleep(50 * time.Millisecond)
				}
			}
			break
		}

	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if _, err := g.SetView("reader", maxX/2-30, maxY/2, maxX/2+30, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
	}
	return nil
}

func writeWord(g *gocui.Gui, word string) error {
	v, err := g.View("reader")
	v.Clear()
	if err != nil {
		return err
	}
	fmt.Fprintln(v, "\x1b[0;31m", word)
	return nil
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			if idx-1 >= 0 {
				idx = idx - 1
			}
			word := words[idx]
			writeWord(g, word)
			return nil
		}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			if idx+1 < len(words)-1 {
				idx = idx + 1
			}
			word := words[idx]
			writeWord(g, word)
			return nil
		}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeySpace, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			pause = !pause
			return nil
		}); err != nil {
		return err
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	close(done)
	return gocui.ErrQuit
}

func main() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}
	go reader(g)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}
