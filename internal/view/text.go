package view

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"unicode/utf8"

	"github.com/athulramesh/rex-agentflow/internal/graph"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const (
	fontSize      = 14.0
	sectionSize   = 13.0
	subSize       = 12.0
	tracking      = -0.011
	baselineOff   = 5.0
	maxTitleLines = 1 // workers use short labels; plans share the same paint path
)

type textPainter struct {
	fonts *fontSet
	scale float64 // device pixels per point; 0 means 1
}

func (tp *textPainter) sc() float64 {
	if tp.scale < 1 {
		return 1
	}
	return tp.scale
}

func (tp *textPainter) measure(face font.Face, s string, track float64) float64 {
	var w fixed.Int26_6
	prev := rune(-1)
	for _, c := range s {
		if prev >= 0 {
			w += face.Kern(prev, c)
		}
		adv, _ := face.GlyphAdvance(c)
		w += adv + fixed.Int26_6(track*64)
		prev = c
	}
	return float64(w) / 64
}

// draw paints s at (x,y) in layout points. size and maxW are also in points.
func (tp *textPainter) draw(img *image.RGBA, s string, x, y float64, bold bool, size float64, col color.RGBA, maxW float64) {
	s = strings.ReplaceAll(s, "\t", " ")
	sc := tp.sc()
	face := tp.fonts.face(bold, size*sc)
	track := tracking * size * sc
	shown, _, _ := tp.fit(s, bold, size, maxW)
	src := image.NewUniform(col)
	dot := fixed.Point26_6{X: fixed.Int26_6(x * sc * 64), Y: fixed.Int26_6(y * sc * 64)}
	prev := rune(-1)
	for _, c := range shown {
		if prev >= 0 {
			dot.X += face.Kern(prev, c)
		}
		dr, mask, mp, adv, ok := face.Glyph(dot, c)
		if ok {
			draw.DrawMask(img, dr, src, image.Point{}, mask, mp, draw.Over)
		}
		dot.X += adv + fixed.Int26_6(track*64)
		prev = c
	}
}

const ell = "…"

func (tp *textPainter) fit(s string, bold bool, size float64, maxW float64) (shown string, kept int, truncated bool) {
	s = strings.ReplaceAll(s, "\t", " ")
	sc := tp.sc()
	face := tp.fonts.face(bold, size*sc)
	track := tracking * size * sc
	limit := maxW * sc
	if maxW <= 0 || tp.measure(face, s, track) <= limit {
		return s, len(s), false
	}
	for len(s) > 0 {
		_, sz := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-sz]
		if tp.measure(face, trimRightSpaces(s)+ell, track) <= limit {
			break
		}
	}
	s = trimRightSpaces(s)
	if i := strings.LastIndex(s, " "); i > len(s)/2 {
		s = trimRightSpaces(s[:i])
	}
	return s + ell, len(s), true
}

func trimRightSpaces(s string) string {
	for len(s) > 0 {
		c, size := utf8.DecodeLastRuneInString(s)
		if c != ' ' && c != ',' && c != '.' && c != ';' && c != ':' && c != '-' {
			break
		}
		s = s[:len(s)-size]
	}
	return s
}

// wrapTitle breaks s into at most maxLines lines; overflow merges into the last line with an ellipsis.
// maxW is in layout points.
func (tp *textPainter) wrapTitle(s string, bold bool, size float64, maxW float64, maxLines int) []string {
	if maxW <= 0 {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	sc := tp.sc()
	face := tp.fonts.face(bold, size*sc)
	track := tracking * size * sc
	limit := maxW * sc
	var lines []string
	cur := ""
	for _, w := range words {
		trial := w
		if cur != "" {
			trial = cur + " " + w
		}
		if tp.measure(face, trial, track) <= limit {
			cur = trial
			continue
		}
		if cur != "" {
			lines = append(lines, cur)
			cur = w
		} else {
			shown, _, _ := tp.fit(w, bold, size, maxW)
			lines = append(lines, shown)
			cur = ""
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) <= maxLines {
		return lines
	}
	tail := strings.Join(lines[maxLines-1:], " ")
	shown, _, _ := tp.fit(tail, bold, size, maxW)
	return append(lines[:maxLines-1], shown)
}

func workerTextColor(p palette, ph graph.Phase) color.RGBA {
	if ph == graph.PhaseFinished {
		return p.muted
	}
	return p.text
}
