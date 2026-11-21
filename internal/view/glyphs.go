package view

import (
	"image"
	"math"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

const (
	glyphRadius    = 7.5
	glyphRingWidth = 1.65
	glyphSweepDef  = 0.4
)

// drawPhaseGlyph paints at device-pixel coordinates. scale multiplies glyph radii.
func drawPhaseGlyph(img *image.RGBA, p palette, cx, cy float64, ph graph.Phase, progress *int, scale float64) {
	if scale < 1 {
		scale = 1
	}
	col := phaseColor(p, ph)
	r := glyphRadius * scale
	rw := glyphRingWidth * scale
	switch ph {
	case graph.PhaseFinished:
		fillCircle(img, cx, cy, r, col)
		drawCheck(img, p, cx-r, cy-r, scale)
	case graph.PhaseWorking:
		outer := arc(nil, cx, cy, r, 0, 2*math.Pi)
		inner := arc(nil, cx, cy, r-rw, 0, 2*math.Pi)
		fillRing(img, outer, inner, col)
		sweep := glyphSweepDef
		if progress != nil {
			sweep = float64(*progress) / 100.0
			if sweep < 0 {
				sweep = 0
			}
			if sweep > 1 {
				sweep = 1
			}
		}
		if sweep > 0.001 {
			start := -math.Pi / 2
			pts := [][2]float64{{cx, cy}}
			pts = arc(pts, cx, cy, r-rw*1.8, start, start+2*math.Pi*sweep)
			pb := newPath(img)
			pb.poly(pts)
			pb.fill(col)
		}
	case graph.PhaseError:
		fillCircle(img, cx, cy, r, col)
		drawBang(img, p, cx, cy, scale)
	default:
		muted := p.muted
		outer := arc(nil, cx, cy, r, 0, 2*math.Pi)
		inner := arc(nil, cx, cy, r-rw, 0, 2*math.Pi)
		fillRing(img, outer, inner, muted)
	}
}

func drawCheck(img *image.RGBA, p palette, bx, by float64, scale float64) {
	pts := [][2]float64{
		{bx + 3.0*scale, by + 6.2*scale},
		{bx + 5.1*scale, by + 8.3*scale},
		{bx + 9.0*scale, by + 3.8*scale},
	}
	w := 0.75 * scale
	pb := newPath(img)
	for i := 0; i+1 < len(pts); i++ {
		a, b := pts[i], pts[i+1]
		dx, dy := b[0]-a[0], b[1]-a[1]
		l := math.Hypot(dx, dy)
		if l < 1e-6 {
			continue
		}
		nx, ny := -dy/l*w, dx/l*w
		ex, ey := dx/l*w*0.6, dy/l*w*0.6
		pb.poly([][2]float64{
			{a[0] - nx - ex, a[1] - ny - ey}, {b[0] - nx + ex, b[1] - ny + ey},
			{b[0] + nx + ex, b[1] + ny + ey}, {a[0] + nx - ex, a[1] + ny - ey},
		})
	}
	pb.fill(p.check)
}

func drawBang(img *image.RGBA, p palette, cx, cy float64, scale float64) {
	fillRect(img, cx-1*scale, cy-4.5*scale, cx+1*scale, cy+1.5*scale, p.check)
	fillCircle(img, cx, cy+4*scale, 1.2*scale, p.check)
}
