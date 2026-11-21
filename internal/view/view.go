package view

import (
	"bufio"
	"fmt"
	"image"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/athulramesh/rex-agentflow/internal/graph"
	"github.com/athulramesh/rex-agentflow/internal/store"
	"golang.org/x/sys/unix"
)

const pollInterval = 150 * time.Millisecond

// Run polls the graph file and redraws via Kitty graphics when it changes.
func Run(graphName string) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	out := bufio.NewWriterSize(os.Stdout, 1<<16)
	// Alternate screen + hide cursor, matching rexpad so placements have a clean canvas.
	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l\x1b[H\x1b[2J")
	_ = out.Flush()
	defer func() {
		kittyDelete(out, 1)
		fmt.Fprint(out, "\x1b_Ga=d,d=A,q=2\x1b\\\x1b[?25h\x1b[?1049l")
		_ = out.Flush()
	}()

	fd := int(os.Stdout.Fd())
	ws := getWinSize(fd)
	imgID := 1

	var lastMod int64
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	redraw := func() {
		g, err := store.Load(graphName)
		if err != nil {
			return
		}
		width := max(ws.xpix, ws.cols*8)
		height := max(ws.ypix, ws.rows*16)
		img := Render(g, width, height)
		kittyDelete(out, imgID)
		kittyUpload(out, img, imgID)
		kittyPlace(out, imgID, 0, img.Bounds().Dx(), img.Bounds().Dy(), ws.cols, ws.rows)
		_ = out.Flush()
	}

	if mod, err := store.FileModTime(graphName); err == nil {
		lastMod = mod
	}
	redraw()

	for {
		select {
		case <-sig:
			return nil
		case <-ticker.C:
			mod, err := store.FileModTime(graphName)
			if err != nil {
				continue
			}
			if mod != lastMod {
				lastMod = mod
				ws = getWinSize(fd)
				redraw()
			}
		}
	}
}

type winSize struct {
	rows, cols, xpix, ypix int
}

func getWinSize(fd int) winSize {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil || ws.Row == 0 {
		return winSize{rows: 24, cols: 80, xpix: 640, ypix: 480}
	}
	w := winSize{rows: int(ws.Row), cols: int(ws.Col), xpix: int(ws.Xpixel), ypix: int(ws.Ypixel)}
	if w.xpix == 0 {
		w.xpix = w.cols * 8
		w.ypix = w.rows * 16
	}
	return w
}

// RenderGraph is used by tests to exercise the render path without a terminal.
func RenderGraph(g graph.WorkGraph, w, h int) *image.RGBA {
	return Render(g, w, h)
}

// Preview writes a one-shot Kitty frame (no poll loop).
func Preview(graphName string) error {
	g, err := store.Load(graphName)
	if err != nil {
		return err
	}
	fd := int(os.Stdout.Fd())
	ws := getWinSize(fd)
	img := Render(g, max(ws.xpix, 320), max(ws.ypix, 200))
	out := bufio.NewWriter(os.Stdout)
	kittyUpload(out, img, 1)
	kittyPlace(out, 1, 0, img.Bounds().Dx(), img.Bounds().Dy(), ws.cols, ws.rows)
	return out.Flush()
}
