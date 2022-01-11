package main

import (
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// grid lays out widgets in an equally-spaced grid.
type grid struct {
	rows, cols int
	spacing    unit.Value
}

type gridWidget func(int, int, layout.Context) layout.Dimensions

// layout places the grid elements by calling widget for each row/column.
func (g *grid) layout(gtx layout.Context, widget gridWidget) layout.Dimensions {
	if g.cols == 0 || g.rows == 0 {
		return layout.Dimensions{}
	}

	var (
		size  = gtx.Constraints.Max
		w, h  = float32(size.X), float32(size.Y)
		space = float32(gtx.Px(g.spacing))
	)
	w = (w - float32(g.cols-1)*space) / float32(g.cols)
	h = (h - float32(g.rows-1)*space) / float32(g.rows)

	cellSize := image.Pt(int(w), int(h))
	gtx.Constraints = layout.Exact(cellSize)
	for row := 0; row < g.rows; row++ {
		for col := 0; col < g.cols; col++ {
			pos := f32.Point{
				X: float32(col)*w + float32(col)*space,
				Y: float32(row)*h + float32(row)*space,
			}
			offset := op.Offset(pos).Push(gtx.Ops)
			widget(row, col, gtx)
			offset.Pop()
		}
	}
	return layout.Dimensions{Size: size}
}

// shrinkToFit renders w, scaling down if it doesn't fit into the available width.
func shrinkToFit(gtx layout.Context, w layout.Widget) layout.Dimensions {
	// Render w with near-infinite width.
	macro := op.Record(gtx.Ops)
	wide := gtx
	wide.Constraints.Max.X = 10e6
	dim := w(wide)
	call := macro.Stop()

	// If it's too wide, push scale transform before drawing.
	if dim.Size.X > gtx.Constraints.Max.X {
		maxWidth := float32(gtx.Constraints.Max.X)
		scale := maxWidth / float32(dim.Size.X)
		origin := f32.Pt(0, float32(gtx.Constraints.Max.Y))
		tr := f32.Affine2D{}.Scale(origin, f32.Pt(scale, scale))
		stack := op.Affine(tr).Push(gtx.Ops)
		defer stack.Pop()

		// Scale dim, too.
		dim.Size = ptf(tr.Transform(layout.FPt(dim.Size)))
	}
	// Draw w.
	call.Add(gtx.Ops)
	return dim
}

// ptf converts f32.Point to image.Point.
// It's kind of the inverse of layout.FPt.
func ptf(pt f32.Point) image.Point {
	return image.Point{
		X: int(math.Ceil(float64(pt.X))),
		Y: int(math.Ceil(float64(pt.Y))),
	}
}
