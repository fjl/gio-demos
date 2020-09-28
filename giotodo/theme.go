package main

import (
	"image"
	"image/color"
	"math"

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
		Background color.RGBA
		MainPanel  color.RGBA
		Item       color.RGBA
		ItemDone   color.RGBA
		HintText   color.RGBA
		StatusText color.RGBA
		Border     color.RGBA
		Title      color.RGBA
		Cross      color.RGBA
		Checkmark  color.RGBA
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
	th.Color.Background = color.RGBA{245, 245, 245, 255}
	th.Color.MainPanel = color.RGBA{255, 255, 255, 255}
	th.Color.Item = color.RGBA{77, 77, 77, 255}
	th.Color.ItemDone = color.RGBA{217, 217, 217, 255}

	th.Color.HintText = color.RGBA{243, 234, 243, 255}
	th.Color.StatusText = color.RGBA{119, 119, 119, 255}
	th.Color.Border = color.RGBA{246, 246, 246, 255}
	th.Color.Title = color.RGBA{175, 47, 47, 100}
	th.Color.Cross = color.RGBA{175, 91, 94, 255}
	th.Color.Checkmark = color.RGBA{93, 194, 175, 255}

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
	Color         color.RGBA
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
		h := float32(dim.Size.Y) / 2
		rect := f32.Rect(0, h, float32(dim.Size.X), h+float32(gtx.Px(unit.Dp(1))))
		paint.PaintOp{Rect: rect}.Add(gtx.Ops)
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
	defer op.Push(gtx.Ops).Pop()

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
	disabled := gtx.Queue == nil
	if e.Editor.Len() > 0 {
		textColor := e.theme.Color.Item
		paint.ColorOp{Color: textColor}.Add(gtx.Ops)
		e.Editor.PaintText(gtx)
	} else {
		call.Add(gtx.Ops)
	}
	if !disabled {
		paint.ColorOp{Color: e.theme.Color.Item}.Add(gtx.Ops)
		e.Editor.PaintCaret(gtx)
	}
	return dims
}

// Items.

type itemStyle struct {
	Label   labelStyle
	Check   *widget.Bool
	Destroy *widget.Clickable
	theme   *todoTheme
}

// Item renders a todo item.
func (th *todoTheme) Item(txt string, check *widget.Bool, destroy *widget.Clickable) itemStyle {
	return itemStyle{
		Label:   th.ItemLabel(txt),
		Check:   check,
		Destroy: destroy,
		theme:   th,
	}
}

func (it *itemStyle) Layout(gtx layout.Context) layout.Dimensions {
	flex := layout.Flex{Alignment: layout.Middle}
	return flex.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(it.Check.Layout),
				layout.Stacked(it.drawCheckbox),
			)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			label := it.Label
			if it.Check.Value {
				label.Color = it.theme.Color.ItemDone
				label.StrikeThrough = true
			}
			return it.theme.Pad.Item.Layout(gtx, label.Layout)
		}),
	)
}

func (it *itemStyle) drawCheckbox(gtx layout.Context) layout.Dimensions {
	var (
		spx  = gtx.Px(it.theme.Size.Checkbox)
		size = image.Pt(spx, spx)
		rect = f32.Rect(0, 0, float32(spx), float32(spx))
	)
	if it.Check.Value {
		it.drawMark(gtx, rect, it.theme.Color.Checkmark)
		it.drawCircle(gtx, rect, mulAlpha(it.theme.Color.Checkmark, 150))
	} else {
		it.drawCircle(gtx, rect, it.theme.Color.Border)
	}
	return layout.Dimensions{Size: size}
}

func (it *itemStyle) drawCircle(gtx layout.Context, rect f32.Rectangle, color color.RGBA) {
	defer op.Push(gtx.Ops).Pop()

	r := rect.Dx() / 2
	w := float32(gtx.Px(unit.Sp(1)))
	b := clip.Border{Rect: rect, NE: r, NW: r, SE: r, SW: r, Width: w}
	b.Add(gtx.Ops)
	paint.ColorOp{Color: color}.Add(gtx.Ops)
	paint.PaintOp{Rect: rect}.Add(gtx.Ops)
}

func (it *itemStyle) drawMark(gtx layout.Context, rect f32.Rectangle, color color.RGBA) {
	defer op.Push(gtx.Ops).Pop()

	var (
		down = f32.Rect(0, 0, float32(gtx.Px(unit.Dp(7))), float32(gtx.Px(unit.Dp(2))))
		up   = f32.Rect(0, 0, float32(gtx.Px(unit.Dp(18))), float32(gtx.Px(unit.Dp(2))))
		rot1 = f32.Affine2D{}.Rotate(f32.Pt(0, 0), -math.Pi/1.34)
		rot2 = f32.Affine2D{}.Rotate(f32.Pt(0, 0), math.Pi/2.3)
	)
	paint.ColorOp{Color: color}.Add(gtx.Ops)
	op.Offset(f32.Pt(rect.Dx()/2.5, rect.Dy()-rect.Dy()/4.3)).Add(gtx.Ops)
	op.Affine(rot1).Add(gtx.Ops)
	paint.PaintOp{Rect: down}.Add(gtx.Ops)
	op.Affine(rot2).Add(gtx.Ops)
	paint.PaintOp{Rect: up}.Add(gtx.Ops)
}

// mulAlpha scales all color components by alpha/255.
func mulAlpha(c color.RGBA, alpha uint8) color.RGBA {
	a := uint16(alpha)
	return color.RGBA{
		A: uint8(uint16(c.A) * a / 255),
		R: uint8(uint16(c.R) * a / 255),
		G: uint8(uint16(c.G) * a / 255),
		B: uint8(uint16(c.B) * a / 255),
	}
}

// Buttons.

type buttonStyle struct {
	Label  labelStyle
	Border color.RGBA
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
	b.Button.Layout(gtx)
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(b.Button.Layout),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return b.theme.Pad.Button.Layout(gtx, b.Label.Layout)
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return b.drawBorder(gtx, b.Border)
		}),
	)
}

func (b buttonStyle) drawBorder(gtx layout.Context, color color.RGBA) layout.Dimensions {
	defer op.Push(gtx.Ops).Pop()

	var (
		radius = b.theme.Size.CornerRadius
		r      = float32(gtx.Px(radius))
		rect   = f32.Rectangle{Min: f32.Pt(0, 0), Max: layout.FPt(gtx.Constraints.Min)}
		w      = float32(gtx.Px(unit.Dp(1)))
		border = clip.Border{Rect: rect, Width: w, SE: r, SW: r, NE: r, NW: r}
	)
	if b.Active {
		border.Add(gtx.Ops)
		paint.ColorOp{color}.Add(gtx.Ops)
		paint.PaintOp{rect}.Add(gtx.Ops)
	}
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

func fill(gtx layout.Context, color color.RGBA) {
	defer op.Push(gtx.Ops).Pop()
	rect := f32.Rectangle{Min: f32.Pt(0, 0), Max: layout.FPt(gtx.Constraints.Max)}
	paint.ColorOp{color}.Add(gtx.Ops)
	paint.PaintOp{rect}.Add(gtx.Ops)
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
