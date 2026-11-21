package view

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/vector"
)

type pathBuilder struct {
	img   *image.RGBA
	polys [][][2]float32
}

func newPath(img *image.RGBA) *pathBuilder {
	return &pathBuilder{img: img}
}

func (p *pathBuilder) poly(pts [][2]float64) {
	px := make([][2]float32, len(pts))
	for i, pt := range pts {
		px[i] = [2]float32{float32(pt[0]), float32(pt[1])}
	}
	p.polys = append(p.polys, px)
}

func (p *pathBuilder) fill(col color.RGBA) {
	p.fillWith(image.NewUniform(col))
}

func (p *pathBuilder) fillWith(src image.Image) {
	if len(p.polys) == 0 {
		return
	}
	minX, minY := float32(math.Inf(1)), float32(math.Inf(1))
	maxX, maxY := float32(math.Inf(-1)), float32(math.Inf(-1))
	for _, poly := range p.polys {
		for _, pt := range poly {
			minX, maxX = min(minX, pt[0]), max(maxX, pt[0])
			minY, maxY = min(minY, pt[1]), max(maxY, pt[1])
		}
	}
	box := image.Rect(
		int(math.Floor(float64(minX))), int(math.Floor(float64(minY))),
		int(math.Ceil(float64(maxX))), int(math.Ceil(float64(maxY))),
	).Intersect(p.img.Bounds())
	if box.Empty() {
		return
	}
	ras := vector.NewRasterizer(box.Dx(), box.Dy())
	ox, oy := float32(box.Min.X), float32(box.Min.Y)
	for _, poly := range p.polys {
		ras.MoveTo(poly[0][0]-ox, poly[0][1]-oy)
		for _, pt := range poly[1:] {
			ras.LineTo(pt[0]-ox, pt[1]-oy)
		}
		ras.ClosePath()
	}
	ras.Draw(p.img, box, src, image.Point{})
}

func arc(pts [][2]float64, cx, cy, rad, a0, a1 float64) [][2]float64 {
	n := int(math.Max(4, math.Ceil(math.Abs(a1-a0)*rad*4)))
	for i := 0; i <= n; i++ {
		a := a0 + (a1-a0)*float64(i)/float64(n)
		pts = append(pts, [2]float64{cx + rad*math.Cos(a), cy + rad*math.Sin(a)})
	}
	return pts
}

func roundRect(x0, y0, x1, y1, rad float64) [][2]float64 {
	rad = math.Max(0, math.Min(rad, math.Min(x1-x0, y1-y0)/2))
	var pts [][2]float64
	pts = arc(pts, x1-rad, y0+rad, rad, -math.Pi/2, 0)
	pts = arc(pts, x1-rad, y1-rad, rad, 0, math.Pi/2)
	pts = arc(pts, x0+rad, y1-rad, rad, math.Pi/2, math.Pi)
	pts = arc(pts, x0+rad, y0+rad, rad, math.Pi, 3*math.Pi/2)
	return pts
}

func reversed(pts [][2]float64) [][2]float64 {
	out := make([][2]float64, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}

func fillRing(img *image.RGBA, outer, inner [][2]float64, col color.RGBA) {
	p := newPath(img)
	p.poly(outer)
	p.poly(reversed(inner))
	p.fill(col)
}

func fillRoundRect(img *image.RGBA, x0, y0, x1, y1, rad float64, col color.RGBA) {
	p := newPath(img)
	p.poly(roundRect(x0, y0, x1, y1, rad))
	p.fill(col)
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 float64, col color.RGBA) {
	p := newPath(img)
	p.poly([][2]float64{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}})
	p.fill(col)
}

func fillCircle(img *image.RGBA, cx, cy, rad float64, col color.RGBA) {
	p := newPath(img)
	p.poly(arc(nil, cx, cy, rad, 0, 2*math.Pi))
	p.fill(col)
}