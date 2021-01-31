package main

import (
	"image"

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

// layout places the grid elements by calling widget for each row/column. This only really
// works well if spacing is non-zero because the cells are placed at integer coordinates.
// The grid will look slighly uneven with too little spacing.
func (g *grid) layout(gtx layout.Context, widget gridWidget) layout.Dimensions {
	var (
		size  = gtx.Constraints.Max
		w, h  = float32(size.X), float32(size.Y)
		space = float32(gtx.Px(g.spacing))
	)
	if g.cols > 0 {
		w = (w - float32(g.cols-1)*space) / float32(g.cols)
	}
	if g.rows > 0 {
		h = (h - float32(g.rows-1)*space) / float32(g.rows)
	}

	for row := 0; row < g.rows; row++ {
		for col := 0; col < g.cols; col++ {
			pos := image.Point{
				X: int(float32(col)*w + float32(col)*space),
				Y: int(float32(row)*h + float32(row)*space),
			}
			size := image.Pt(int(w), int(h))
			stk := op.Save(gtx.Ops)
			op.Offset(layout.FPt(pos)).Add(gtx.Ops)
			gtx.Constraints = layout.Exact(size)
			widget(row, col, gtx)
			stk.Load()
		}
	}
	return layout.Dimensions{Size: size}
}

// shrinkToFit renders w, scaling down if it doesn't fit into the available width.
func shrinkToFit(gtx layout.Context, w layout.Widget) layout.Dimensions {
	defer op.Save(gtx.Ops).Load()

	// Render w with near-infinite width.
	macro := op.Record(gtx.Ops)
	wide := gtx
	wide.Constraints.Max.X = 10e6
	dim := w(wide)
	call := macro.Stop()

	// Scale down if it exceeds the available space.
	if dim.Size.X > gtx.Constraints.Max.X {
		scale := float32(gtx.Constraints.Max.X) / float32(dim.Size.X)
		origin := f32.Pt(0, float32(gtx.Constraints.Max.Y))
		tr := f32.Affine2D{}.Scale(origin, f32.Pt(scale, scale))
		op.Affine(tr).Add(gtx.Ops)
	}
	call.Add(gtx.Ops)
	return layout.Dimensions{Size: gtx.Constraints.Max}
}
