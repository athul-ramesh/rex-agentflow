package view

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

type palette struct {
	bg, title, text, muted                    color.RGBA
	working, blocked, finished, error, queued color.RGBA
	divider, activeBg, check                  color.RGBA
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
	divider:  color.RGBA{58, 60, 64, 255},
	activeBg: color.RGBA{37, 49, 72, 255},
	check:    color.RGBA{255, 255, 255, 255},
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

func mix(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

// Layout constants are in points. RenderScaled multiplies by device scale.
const (
	marginL     = 12.0
	marginR     = 12.0
	marginTop   = 14.0
	itemPitch   = 28.0
	wrapLineH   = 18.0
	subLineH    = 16.0
	iconCenterX = 8.0
	textX       = 21.0
	depthIndent = 14.0
	activeInset = 6.0
)

// Render draws at 1x (tests and simple callers).
func Render(g graph.WorkGraph, width, height int) *image.RGBA {
	return RenderScaled(g, width, height, 1)
}

// RenderScaled draws the work graph into width×height device pixels.
// scale is device pixels per layout point (2 on a typical Retina pane).
func RenderScaled(g graph.WorkGraph, width, height int, scale float64) *image.RGBA {
	if width < 120 {
		width = 320
	}
	if height < 80 {
		height = 400
	}
	if scale < 1 {
		scale = 1
	}
	fs, err := getFonts()
	if err != nil {
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(img, img.Bounds(), &image.Uniform{darkPalette.bg}, image.Point{}, draw.Src)
		return img
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	pal := darkPalette
	draw.Draw(img, img.Bounds(), &image.Uniform{pal.bg}, image.Point{}, draw.Src)

	tp := &textPainter{fonts: fs, scale: scale}
	rows := buildLayout(g)
	contentW := float64(width)/scale - marginL - marginR
	textLeft := marginL + textX
	s := scale

	y := marginTop

	for _, row := range rows {
		switch row.Kind {
		case rowTitle:
			tp.draw(img, row.Text, marginL, y+baselineOff, true, fontSize, pal.title, 0)
			y += 10
			fillRect(img, marginL*s, (y+6)*s, (float64(width)/scale-marginR)*s, (y+7)*s, pal.divider)
			y += 24

		case rowSection:
			y += 4
			tp.draw(img, row.Text, marginL, y+baselineOff, true, sectionSize, pal.muted, 0)
			y += itemPitch - 4

		case rowPlan, rowWorker:
			gx := marginL + iconCenterX + float64(row.Depth)*depthIndent
			tx := textLeft + float64(row.Depth)*depthIndent
			maxW := contentW - textX - float64(row.Depth)*depthIndent

			lines := tp.wrapTitle(row.Text, false, fontSize, maxW, maxTitleLines)
			subH := 0.0
			if row.Sub != "" {
				subH = subLineH
			}
			extra := float64(len(lines)-1) * wrapLineH
			rowH := itemPitch + extra + subH
			cy := y + itemPitch/2 + extra/2

			if row.Active {
				fillRoundRect(img,
					(marginL-activeInset)*s, (cy-itemPitch/2+2)*s,
					(float64(width)/scale-marginR+activeInset)*s, (cy+itemPitch/2-2+extra+subH)*s,
					6*s, pal.activeBg)
			}

			drawPhaseGlyph(img, pal, gx*s, cy*s, row.Phase, row.Progress, s)

			col := pal.text
			if row.Kind == rowWorker {
				col = workerTextColor(pal, row.Phase)
			} else if row.Phase == graph.PhaseFinished {
				col = pal.muted
			}

			line0Y := cy - float64(len(lines)-1)*wrapLineH/2 + baselineOff
			for i, line := range lines {
				tp.draw(img, line, tx, line0Y+float64(i)*wrapLineH, false, fontSize, col, maxW)
			}

			if row.Sub != "" {
				sy := line0Y + float64(len(lines))*wrapLineH - 6
				tp.draw(img, row.Sub, tx, sy, false, subSize, pal.muted, maxW)
			}

			y += rowH

		case rowEmpty:
			tp.draw(img, row.Text, textLeft, y+baselineOff, false, fontSize, pal.muted, 0)
			y += itemPitch

		case rowFooter:
			y += 10
			fillRect(img, marginL*s, y*s, (float64(width)/scale-marginR)*s, (y+1)*s, pal.divider)
			y += 12
			tp.draw(img, row.Text, marginL, y+baselineOff, false, subSize, pal.muted, 0)
			y += itemPitch
		}
	}

	return img
}
