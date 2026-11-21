package view

import (
	"encoding/json"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func TestWriteAfterPNG(t *testing.T) {
	const demoPath = "/Users/athulramesh/.local/state/agentflow/block_01a1fd58-3bc9-70bb-942a-6c3074769e1a.json"
	data, err := os.ReadFile(demoPath)
	if err != nil {
		t.Skip("demo graph not present:", err)
	}
	var g graph.WorkGraph
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatal(err)
	}
	img := RenderGraph(g, 420, 560)
	_ = os.MkdirAll("/tmp/agentflow-v1-polish", 0o755)
	out := "/tmp/agentflow-v1-polish/after.png"
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestDemoGraphTitleEllipsis(t *testing.T) {
	fs, err := getFonts()
	if err != nil {
		t.Fatal(err)
	}
	tp := &textPainter{fonts: fs}
	title := "Independently inspect this repository and explain its purpose, integrations (Cursor/Pi/Rex), and what the user-facing commands do."
	lines := tp.wrapTitle(title, false, fontSize, 260, maxTitleLines)
	if len(lines) > maxTitleLines {
		t.Fatalf("lines %d", len(lines))
	}
	last := lines[len(lines)-1]
	if !strings.HasSuffix(last, ell) {
		t.Fatalf("expected ellipsis on wrapped title, got %q", last)
	}
}

func TestRenderUsesVectorGlyphsNotBasicfont(t *testing.T) {
	g := graph.New("cursor-hooks")
	g.Workers = []graph.Worker{{ID: "1", Title: "ok", State: graph.PhaseFinished}}
	img := Render(g, 200, 200)
	// Finished worker row center ≈ marginTop + title block + section + itemPitch/2
	cy := int(marginTop + 34 + itemPitch + itemPitch/2)
	gx := int(marginL + iconCenterX)
	rgba := color.RGBAModel.Convert(img.At(gx, cy)).(color.RGBA)
	if rgba.G < 100 || rgba.R > 120 {
		t.Fatalf("expected finished glyph green at (%d,%d), got %#v", gx, cy, rgba)
	}
}