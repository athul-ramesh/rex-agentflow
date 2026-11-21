package view

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/athulramesh/rex-agentflow/internal/graph"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type palette struct {
	bg, title, text, muted color.RGBA
	working, blocked, finished, error, queued color.RGBA
}

var darkPalette = palette{
	bg:       color.RGBA{30, 31, 34, 255},
	title:    color.RGBA{236, 237, 239, 255},
	text:     color.RGBA{226, 228, 231, 255},
	muted:    color.RGBA{128, 131, 137, 255},
	working:  color.RGBA{84, 136, 238, 255},
	blocked:  color.RGBA{230, 170, 60, 255},
	finished: color.RGBA{76, 175, 110, 255},
	error:    color.RGBA{220, 80, 80, 255},
	queued:   color.RGBA{120, 123, 130, 255},
}

func phaseColor(p palette, ph graph.Phase) color.RGBA {
	switch ph {
	case graph.PhaseWorking:
		return p.working
	case graph.PhaseBlocked:
		return p.blocked
	case graph.PhaseFinished:
		return p.finished
	case graph.PhaseError:
		return p.error
	default:
		return p.queued
	}
}

const (
	marginX = 12
	marginY = 10
	lineH   = 18
	indent  = 14
)

// Render draws the work graph into a new RGBA image.
func Render(g graph.WorkGraph, width, height int) *image.RGBA {
	if width < 120 {
		width = 320
	}
	if height < 80 {
		height = 400
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	p := darkPalette
	draw.Draw(img, img.Bounds(), &image.Uniform{p.bg}, image.Point{}, draw.Src)

	y := marginY + lineH
	drawString(img, p.title, marginX, y, titleOrDefault(g))
	y += lineH + 6

	if len(g.Plan) > 0 {
		drawString(img, p.muted, marginX, y, "Plan")
		y += lineH
		for _, item := range g.Plan {
			mark := planMark(item.Status)
			col := phaseColor(p, item.Status)
			drawString(img, col, marginX, y, mark+" "+item.Text)
			y += lineH
		}
		y += 4
	}

	drawString(img, p.muted, marginX, y, "Workers")
	y += lineH

	roots := workersByParent(g.Workers, "")
	for _, w := range roots {
		y = drawWorkerTree(img, p, g.Workers, w, marginX, y, 0)
	}
	if len(g.Workers) == 0 {
		drawString(img, p.muted, marginX, y, "(empty)")
	}

	return img
}

func titleOrDefault(g graph.WorkGraph) string {
	if g.Title != "" {
		return g.Title
	}
	if g.RunID != "" {
		return "run " + truncate(g.RunID, 24)
	}
	return "agentflow"
}

func planMark(ph graph.Phase) string {
	switch ph {
	case graph.PhaseFinished:
		return "[x]"
	case graph.PhaseWorking:
		return "[~]"
	case graph.PhaseError:
		return "[!]"
	default:
		return "[ ]"
	}
}

func workersByParent(workers []graph.Worker, parent string) []graph.Worker {
	var out []graph.Worker
	for _, w := range workers {
		if w.ParentID == parent {
			out = append(out, w)
		}
	}
	if parent == "" && len(out) == 0 {
		return workers
	}
	return out
}

func drawWorkerTree(img *image.RGBA, p palette, all []graph.Worker, w graph.Worker, x, y, depth int) int {
	col := phaseColor(p, w.State)
	prefix := ""
	for i := 0; i < depth; i++ {
		prefix += "  "
	}
	label := fmt.Sprintf("%s● %s", prefix, w.Title)
	if w.Kind != "" {
		label += " (" + w.Kind + ")"
	}
	drawString(img, col, x+depth*indent, y, label)
	y += lineH
	if w.Msg != "" {
		drawString(img, p.muted, x+depth*indent+8, y, truncate(w.Msg, 48))
		y += lineH
	}
	for _, child := range workersByParent(all, w.ID) {
		y = drawWorkerTree(img, p, all, child, x, y, depth+1)
	}
	return y
}

func drawString(img *image.RGBA, col color.RGBA, x, y int, s string) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(0, 0),
	}
	// basicfont baseline
	d.Dot = fixed.P(x, y-2)
	d.DrawString(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}