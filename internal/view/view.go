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
	var lastKey string
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	redraw := func(force bool) {
		g, err := store.Load(graphName)
		if err != nil {
			return
		}
		widthPx, heightPx, scale := frameSize(ws)
		key := fmt.Sprintf("%d:%d:%.2f:%d", widthPx, heightPx, scale, lastMod)
		if !force && key == lastKey {
			return
		}
		img := RenderScaled(g, widthPx, heightPx, scale)
		kittyDelete(out, imgID)
		kittyUpload(out, img, imgID)
		kittyPlace(out, imgID, 0, img.Bounds().Dx(), img.Bounds().Dy(), ws.cols, ws.rows)
		_ = out.Flush()
		lastKey = key
	}

	if mod, err := store.FileModTime(graphName); err == nil {
		lastMod = mod
	}
	redraw(true)

	for {
		select {
		case <-sig:
			return nil
		case <-ticker.C:
			ws = getWinSize(fd)
			mod, err := store.FileModTime(graphName)
			if err == nil && mod != lastMod {
				lastMod = mod
			}
			redraw(false)
		}
	}
}

type winSize struct {
	rows, cols, xpix, ypix int
}

func getWinSize(fd int) winSize {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil || ws.Row == 0 || ws.Col == 0 {
		return winSize{rows: 24, cols: 80, xpix: 80 * 16, ypix: 24 * 34}
	}
	return normalizeWinSize(winSize{
		rows: int(ws.Row), cols: int(ws.Col),
		xpix: int(ws.Xpixel), ypix: int(ws.Ypixel),
	})
}

func normalizeWinSize(w winSize) winSize {
	if w.xpix == 0 || w.ypix == 0 {
		// No pixel size reported: assume a Retina cell (same as rexpad).
		w.xpix, w.ypix = w.cols*16, w.rows*34
		return w
	}
	// Many ptys still advertise the historical 8×16 cell when HiDPI metrics
	// never arrived. Treat that like a missing pixel size so we do not upload
	// a 1x bitmap that Kitty then stretches across a Retina pane.
	cellW, cellH := w.xpix/w.cols, w.ypix/w.rows
	if cellW > 0 && cellH > 0 && cellW <= 8 && cellH <= 16 {
		w.xpix, w.ypix = w.cols*16, w.rows*34
	}
	return w
}

// frameSize returns a whole-cell pixel frame and the layout scale.
func frameSize(ws winSize) (widthPx, heightPx int, scale float64) {
	if ws.cols <= 0 || ws.rows <= 0 {
		return 320, 400, 1
	}
	cellW := ws.xpix / ws.cols
	cellH := ws.ypix / ws.rows
	if cellW <= 0 {
		cellW = 16
	}
	if cellH <= 0 {
		cellH = 34
	}
	ws.xpix, ws.ypix = cellW*ws.cols, cellH*ws.rows
	return ws.xpix, ws.ypix, pickScale(0, ws)
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
	widthPx, heightPx, scale := frameSize(ws)
	img := RenderScaled(g, widthPx, heightPx, scale)
	out := bufio.NewWriter(os.Stdout)
	kittyUpload(out, img, 1)
	kittyPlace(out, 1, 0, img.Bounds().Dx(), img.Bounds().Dy(), ws.cols, ws.rows)
	return out.Flush()
}
