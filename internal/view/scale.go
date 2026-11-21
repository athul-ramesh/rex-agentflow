package view

import (
	"math"
	"os"
	"strconv"
)

// pickScale is device pixels per layout point, matching rexpad: a ~17pt
// terminal cell at Retina (~34px) yields 2x so 14pt type is 28 device pixels.
func pickScale(flag float64, ws winSize) float64 {
	if flag > 0 {
		return flag
	}
	if v, err := strconv.ParseFloat(os.Getenv("AGENTFLOW_SCALE"), 64); err == nil && v > 0 {
		return v
	}
	if ws.rows <= 0 || ws.ypix <= 0 {
		return 1
	}
	cellH := float64(ws.ypix) / float64(ws.rows)
	return math.Max(1, math.Round(cellH/17*2)/2)
}
