package view

import (
	"image/color"
	"testing"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func TestPickScaleRetinaCell(t *testing.T) {
	ws := winSize{rows: 40, cols: 80, xpix: 80 * 16, ypix: 40 * 34}
	got := pickScale(0, ws)
	if got != 2 {
		t.Fatalf("scale %v want 2", got)
	}
}

func TestPickScaleOneXCell(t *testing.T) {
	ws := winSize{rows: 40, cols: 80, xpix: 80 * 10, ypix: 40 * 17}
	got := pickScale(0, ws)
	if got != 1 {
		t.Fatalf("scale %v want 1", got)
	}
}

func TestNormalizeWinSizePromotesHistoric8x16(t *testing.T) {
	ws := normalizeWinSize(winSize{rows: 24, cols: 80, xpix: 80 * 8, ypix: 24 * 16})
	w, h, scale := frameSize(ws)
	if w != 80*16 || h != 24*34 {
		t.Fatalf("frame %dx%d", w, h)
	}
	if scale != 2 {
		t.Fatalf("scale %v want 2", scale)
	}
}

func TestFrameSizeWholeCells(t *testing.T) {
	ws := winSize{rows: 51, cols: 52, xpix: 52*14 + 3, ypix: 51*32 + 5}
	w, h, scale := frameSize(ws)
	if w != 52*14 || h != 51*32 {
		t.Fatalf("frame %dx%d want %dx%d", w, h, 52*14, 51*32)
	}
	if scale != 2 {
		t.Fatalf("scale %v want 2", scale)
	}
}

func TestRenderScaledUsesLargerFace(t *testing.T) {
	g := graph.New("test")
	g.Title = "agentflow"
	g.Workers = []graph.Worker{{ID: "1", Title: "main", State: graph.PhaseFinished}}
	img1 := RenderScaled(g, 400, 400, 1)
	img2 := RenderScaled(g, 800, 800, 2)
	if img2.Bounds().Dx() != 800 || img1.Bounds().Dx() != 400 {
		t.Fatalf("sizes %v %v", img1.Bounds(), img2.Bounds())
	}
	// At 2x the finished glyph center is twice as far from the origin.
	cy1 := int(marginTop + 34 + itemPitch + itemPitch/2)
	gx1 := int(marginL + iconCenterX)
	cy2, gx2 := cy1*2, gx1*2
	c1 := color.RGBAModel.Convert(img1.At(gx1, cy1)).(color.RGBA)
	c2 := color.RGBAModel.Convert(img2.At(gx2, cy2)).(color.RGBA)
	if c1.G < 100 || c2.G < 100 {
		t.Fatalf("expected green glyphs at 1x=%#v 2x=%#v", c1, c2)
	}
}
