package main

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
)

// todoTheme defines the TodoMVC style.
type todoTheme struct {
	Shaper text.Shaper
	Color  struct {
		Background color.NRGBA
		MainPanel  color.NRGBA
		Item       color.NRGBA
		ItemDone   color.NRGBA
		HintText   color.NRGBA
		StatusText color.NRGBA
		Error      color.NRGBA
		Selection  color.NRGBA
		Border     color.NRGBA
		Title      color.NRGBA
		Cross      color.NRGBA
		Checkmark  color.NRGBA
	}
	Size struct {
		ItemText     unit.Value
		StatusText   unit.Value
		CornerRadius unit.Value
		Checkbox     unit.Value
		MainWidth    unit.Value
	}
	Pad struct {
		Main   layout.Inset
		Button layout.Inset
		Item   layout.Inset
	}
	Font struct {
		Item     text.Font
		ItemHint text.Font
		Status   text.Font
	}
}

func newTodoTheme(fonts []text.FontFace) *todoTheme {
	th := &todoTheme{Shaper: text.NewCache(fonts)}

	// Colors.
	th.Color.Background = color.NRGBA{245, 245, 245, 255}
	th.Color.MainPanel = color.NRGBA{255, 255, 255, 255}
	th.Color.Item = color.NRGBA{77, 77, 77, 255}
	th.Color.ItemDone = color.NRGBA{217, 217, 217, 255}

	th.Color.HintText = color.NRGBA{243, 234, 243, 255}
	th.Color.StatusText = color.NRGBA{119, 119, 119, 255}
	th.Color.Error = color.NRGBA{255, 119, 119, 255}
	th.Color.Border = color.NRGBA{235, 235, 235, 255}
	th.Color.Title = color.NRGBA{175, 47, 47, 100}
	th.Color.Cross = color.NRGBA{175, 91, 94, 255}
	th.Color.Checkmark = color.NRGBA{93, 194, 175, 255}
	th.Color.Selection = color.NRGBA{93, 194, 175, 100}

	// Sizes.
	th.Size.ItemText = unit.Dp(26)
	th.Size.StatusText = unit.Dp(14)
	th.Size.CornerRadius = unit.Dp(3)
	th.Size.Checkbox = unit.Dp(30)
	th.Size.MainWidth = unit.Dp(550)

	// Padding.
	th.Pad.Main = layout.UniformInset(unit.Dp(12))
	th.Pad.Button = layout.Inset{
		Top:    unit.Dp(4),
		Bottom: unit.Dp(4),
		Left:   unit.Dp(8),
		Right:  unit.Dp(8),
	}
	th.Pad.Item = layout.Inset{
		Top:    unit.Dp(8),
		Bottom: unit.Dp(8),
		Left:   unit.Dp(16),
		Right:  unit.Dp(8),
	}

	// Fonts.
	th.Font.Item.Style = text.Regular
	th.Font.ItemHint.Style = text.Italic
	th.Font.Status.Style = text.Regular

	return th
}

// Labels.

type labelStyle struct {
	Text          string
	Color         color.NRGBA
	Font          text.Font
	TextSize      unit.Value
	StrikeThrough bool
	Alignment     text.Alignment
	theme         *todoTheme
}

// StatusLabel makes a label with status bar style.
func (th *todoTheme) StatusLabel(txt string) labelStyle {
	return labelStyle{
		Text:     txt,
		Color:    th.Color.StatusText,
		Font:     th.Font.Status,
		TextSize: th.Size.StatusText,
		theme:    th,
	}
}

// ItemLabel makes a label that shows todo item text.
func (th *todoTheme) ItemLabel(txt string) labelStyle {
	return labelStyle{
		Text:     txt,
		Color:    th.Color.Item,
		Font:     th.Font.Item,
		TextSize: th.Size.ItemText,
		theme:    th,
	}
}

func (l labelStyle) Layout(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = image.Point{}

	// Draw text.
	paint.ColorOp{Color: l.Color}.Add(gtx.Ops)
	dim := widget.Label{MaxLines: 1}.Layout(gtx, l.theme.Shaper, l.Font, l.TextSize, l.Text)

	// Draw strike.
	if l.StrikeThrough {
		h := dim.Size.Y / 2
		rect := clip.Rect(image.Rect(0, h, dim.Size.X, h+gtx.Px(unit.Dp(2))))
		paint.FillShape(gtx.Ops, l.Color, rect.Op())
	}
	return dim
}

// Editor.

type editorStyle struct {
	Hint   string
	Editor *widget.Editor
	theme  *todoTheme
}

// Editor renders an item editor.
func (th *todoTheme) Editor(ed *widget.Editor, hint string) editorStyle {
	return editorStyle{Hint: hint, Editor: ed, theme: th}
}

func (e *editorStyle) Layout(gtx layout.Context) layout.Dimensions {
	defer op.Save(gtx.Ops).Load()

	// Draw label.
	macro := op.Record(gtx.Ops)
	paint.ColorOp{Color: e.theme.Color.HintText}.Add(gtx.Ops)
	tl := widget.Label{Alignment: e.Editor.Alignment}
	dims := tl.Layout(gtx, e.theme.Shaper, e.theme.Font.ItemHint, e.theme.Size.ItemText, e.Hint)
	call := macro.Stop()
	if w := dims.Size.X; gtx.Constraints.Min.X < w {
		gtx.Constraints.Min.X = w
	}
	if h := dims.Size.Y; gtx.Constraints.Min.Y < h {
		gtx.Constraints.Min.Y = h
	}

	// Draw editor.
	dims = e.Editor.Layout(gtx, e.theme.Shaper, e.theme.Font.Item, e.theme.Size.ItemText)
	if e.Editor.SelectionLen() > 0 {
		paint.ColorOp{Color: e.theme.Color.Selection}.Add(gtx.Ops)
		e.Editor.PaintSelection(gtx)
	}
	if e.Editor.Len() > 0 {
		paint.ColorOp{Color: e.theme.Color.Item}.Add(gtx.Ops)
		e.Editor.PaintText(gtx)
	} else {
		call.Add(gtx.Ops)
	}
	disabled := gtx.Queue == nil
	if !disabled {
		paint.ColorOp{Color: e.theme.Color.Item}.Add(gtx.Ops)
		e.Editor.PaintCaret(gtx)
	}
	return dims
}

// Items.

type itemStyle struct {
	item  *item
	Label labelStyle
	theme *todoTheme
}

// Item renders a todo item.
func (th *todoTheme) Item(txt string, item *item) itemStyle {
	return itemStyle{
		item:  item,
		Label: th.ItemLabel(txt),
		theme: th,
	}
}

// Layout draws a complete item.
func (it *itemStyle) Layout(gtx layout.Context) layout.Dimensions {
	var (
		cbsize  = gtx.Px(it.theme.Size.Checkbox)
		cbconst = layout.Exact(image.Pt(cbsize, cbsize))
	)

	flex := layout.Flex{Alignment: layout.Middle}
	return flex.Layout(gtx,
		// Checkbox.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = cbconst     // Constant size.
			it.item.done.Layout(gtx)      // Handle clicks.
			return it.layoutCheckbox(gtx) // Draw.
		}),
		// Item text.
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			// Stack the text click here to track hovering over the text.
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(it.item.textClick.Layout),
				layout.Stacked(it.layoutText),
			)
		}),
		// Remove button.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = cbconst  // Constant size (same as checkbox).
			it.item.remove.Layout(gtx) // Handle clicks.
			// Draw cross when hovering over the item.
			hovered := it.item.textClick.Hovered() || it.item.remove.Hovered()
			dim := layout.Dimensions{Size: cbconst.Max}
			if hovered {
				inset := layout.UniformInset(unit.Dp(10))
				dim = inset.Layout(gtx, it.layoutCross)
			}
			return dim
		}),
	)
}

// layoutText draws the item text.
func (it *itemStyle) layoutText(gtx layout.Context) layout.Dimensions {
	label := it.Label
	if it.item.done.Value {
		label.Color = it.theme.Color.ItemDone
		label.StrikeThrough = true
	}
	dim := it.theme.Pad.Item.Layout(gtx, label.Layout)
	// Label returns minimum required size, but should really
	// grab all available space to make the alignment work, so
	// forcefully expand size to max width here.
	dim.Size.X = gtx.Constraints.Max.X
	return dim
}

// layoutCheckbox draws the checkbox.
func (it *itemStyle) layoutCheckbox(gtx layout.Context) layout.Dimensions {
	var (
		spx  = gtx.Constraints.Max.X
		size = image.Pt(spx, spx)
		rect = f32.Rectangle{Max: layout.FPt(size)}
	)
	if it.item.done.Value {
		circleColor := it.theme.Color.Checkmark
		circleColor.A = 125
		it.drawMark(gtx, rect, it.theme.Color.Checkmark)
		it.drawCircle(gtx, rect, circleColor)
	} else {
		it.drawCircle(gtx, rect, it.theme.Color.Border)
	}
	return layout.Dimensions{Size: size}
}

// drawCircle draws the checkmark button outline.
func (it *itemStyle) drawCircle(gtx layout.Context, rect f32.Rectangle, color color.NRGBA) {
	w := float32(gtx.Px(unit.Sp(1)))
	circle := clip.Circle{
		Center: f32.Pt(rect.Max.X/2, rect.Max.Y/2),
		Radius: rect.Dx() / 2,
	}
	stroke := clip.Stroke{
		Path:  circle.Path(gtx.Ops),
		Style: clip.StrokeStyle{Width: w},
	}
	paint.FillShape(gtx.Ops, color, stroke.Op())
}

// drawMark draws the checkmark button icon.
func (it *itemStyle) drawMark(gtx layout.Context, rect f32.Rectangle, color color.NRGBA) {
	var (
		path  clip.Path
		start = f32.Pt(rect.Dx()-rect.Dx()/4, rect.Dy()/4)
		low   = f32.Pt(rect.Dx()/2.3, rect.Dy()-rect.Dy()/4.6)
		end   = f32.Pt(rect.Dx()/4, rect.Dy()-rect.Dy()/2.4)
	)
	path.Begin(gtx.Ops)
	path.MoveTo(start)
	path.LineTo(low)
	path.LineTo(end)
	paint.FillShape(gtx.Ops, color, clip.Stroke{
		Path:  path.End(),
		Style: clip.StrokeStyle{Width: float32(gtx.Px(unit.Dp(1.8)))},
	}.Op())
}

// layoutCross draws the remove button.
func (it *itemStyle) layoutCross(gtx layout.Context) layout.Dimensions {
	// Draw.
	var (
		spx  = gtx.Constraints.Max.X
		size = image.Pt(spx, spx)
		rect = f32.Rectangle{Max: layout.FPt(size)}
	)
	it.drawCross(gtx, rect)
	return layout.Dimensions{Size: size}
}

// drawCross draws the remove button icon.
func (it *itemStyle) drawCross(gtx layout.Context, rect f32.Rectangle) {
	var color = it.theme.Color.Cross

	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(f32.Pt(rect.Min.X, rect.Min.Y))
	path.LineTo(f32.Pt(rect.Max.X, rect.Max.Y))
	paint.FillShape(gtx.Ops, color, clip.Stroke{
		Path:  path.End(),
		Style: clip.StrokeStyle{Width: float32(gtx.Px(unit.Dp(1.8)))},
	}.Op())

	path.Begin(gtx.Ops)
	path.MoveTo(f32.Pt(rect.Min.X, rect.Max.Y))
	path.LineTo(f32.Pt(rect.Max.X, rect.Min.Y))
	paint.FillShape(gtx.Ops, color, clip.Stroke{
		Path:  path.End(),
		Style: clip.StrokeStyle{Width: float32(gtx.Px(unit.Dp(1.8)))},
	}.Op())
}

// Buttons.

type buttonStyle struct {
	Label  labelStyle
	Border color.NRGBA
	Active bool
	Button *widget.Clickable
	theme  *todoTheme
}

// StatusButton makes a button with a border.
// The border is shown when 'active' is true.
func (th *todoTheme) StatusButton(click *widget.Clickable, txt string, active bool) buttonStyle {
	return buttonStyle{
		Label:  th.StatusLabel(txt),
		Border: th.Color.Title,
		Button: click,
		Active: active,
		theme:  th,
	}
}

// Clickable makes a button with no border.
func (th *todoTheme) Clickable(click *widget.Clickable, txt string) buttonStyle {
	return buttonStyle{
		Label:  th.StatusLabel(txt),
		Button: click,
		theme:  th,
	}
}

func (b buttonStyle) Layout(gtx layout.Context) layout.Dimensions {
	inset := layout.UniformInset(unit.Dp(1))
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Stack{}.Layout(gtx,
			// Handle clicks.
			layout.Expanded(b.Button.Layout),
			// Draw text.
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return b.theme.Pad.Button.Layout(gtx, b.Label.Layout)
			}),
			// Draw border.
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				if b.Active {
					return b.drawBorder(gtx, b.Border)
				} else if b.Button.Hovered() {
					color := b.Border
					color.A -= color.A / 3
					return b.drawBorder(gtx, color)
				}
				return layout.Dimensions{}
			}),
		)
	})
}

func (b buttonStyle) drawBorder(gtx layout.Context, color color.NRGBA) layout.Dimensions {
	var (
		radius = b.theme.Size.CornerRadius
		r      = float32(gtx.Px(radius))
		w      = float32(gtx.Px(unit.Dp(1)))
		rect   = f32.Rectangle{Min: f32.Pt(0, 0), Max: layout.FPt(gtx.Constraints.Min)}
		rr     = clip.RRect{Rect: rect, SE: r, SW: r, NE: r, NW: r}
		border = clip.Stroke{Path: rr.Path(gtx.Ops), Style: clip.StrokeStyle{Width: w}}
	)
	paint.FillShape(gtx.Ops, color, border.Op())
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

// showIf draws w if cond is true.
func showIf(cond bool, gtx layout.Context, w layout.Widget) layout.Dimensions {
	m := op.Record(gtx.Ops)
	dim := w(gtx)
	call := m.Stop()
	if cond {
		call.Add(gtx.Ops)
	}
	return dim
}

// rigidInset makes a rigid flex child with uniform inset.
func rigidInset(inset unit.Value, w layout.Widget) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(inset).Layout(gtx, w)
	})
}
