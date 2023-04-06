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

	. "github.com/fjl/gio-demos/internal/cd"
)

// todoTheme defines the TodoMVC style.
type todoTheme struct {
	Shaper *text.Shaper
	Color  struct {
		Background color.NRGBA
		MainPanel  color.NRGBA
		Item       color.NRGBA
		ItemDone   color.NRGBA
		ItemEditBG color.NRGBA
		HintText   color.NRGBA
		StatusText color.NRGBA
		Error      color.NRGBA
		Selection  color.NRGBA
		Border     color.NRGBA
		Title      color.NRGBA
		Checkmark  color.NRGBA
		Remove     color.NRGBA
		RemoveBG   color.NRGBA
	}
	Size struct {
		ItemText     unit.Sp
		StatusText   unit.Sp
		CornerRadius unit.Dp
		Checkbox     unit.Dp
		Remove       unit.Dp
		MinWidth     unit.Dp
		MaxWidth     unit.Dp
		PrefWidth    unit.Dp
	}
	Pad struct {
		Main     layout.Inset
		MainItem layout.Inset
		Button   layout.Inset
		Remove   layout.Inset
		Item     layout.Inset
	}
	Font struct {
		Item     text.Font
		ItemHint text.Font
		Status   text.Font
	}
}

func newTodoTheme(fonts []text.FontFace) *todoTheme {
	th := &todoTheme{Shaper: text.NewShaper(fonts)}

	// Colors.
	th.Color.Background = color.NRGBA{245, 245, 245, 255}
	th.Color.MainPanel = color.NRGBA{255, 255, 255, 255}
	th.Color.Selection = color.NRGBA{93, 194, 175, 100}

	th.Color.HintText = color.NRGBA{243, 234, 243, 255}
	th.Color.StatusText = color.NRGBA{119, 119, 119, 255}
	th.Color.Error = color.NRGBA{255, 119, 119, 255}
	th.Color.Border = color.NRGBA{200, 200, 200, 130}
	th.Color.Title = color.NRGBA{175, 47, 47, 100}

	th.Color.Item = color.NRGBA{77, 77, 77, 255}
	th.Color.ItemDone = color.NRGBA{217, 217, 217, 255}
	th.Color.ItemEditBG = color.NRGBA{77, 77, 77, 18}
	th.Color.Checkmark = color.NRGBA{93, 194, 175, 255}
	th.Color.Remove = color.NRGBA{175, 91, 94, 255}
	th.Color.RemoveBG = th.Color.Remove
	th.Color.RemoveBG.A = 30

	// Sizes.
	th.Size.ItemText = 26
	th.Size.StatusText = 14
	th.Size.CornerRadius = 3
	th.Size.Checkbox = 30
	th.Size.Remove = 20
	th.Size.MinWidth = 350
	th.Size.PrefWidth = 550
	th.Size.MaxWidth = 700

	// Padding.
	th.Pad.Main = layout.UniformInset(unit.Dp(12))
	th.Pad.MainItem = layout.Inset{
		Top:    0,
		Bottom: 0,
		Left:   th.Pad.Main.Left,
		Right:  th.Pad.Main.Right,
	}
	th.Pad.Button = layout.Inset{
		Top:    unit.Dp(4),
		Bottom: unit.Dp(4),
		Left:   unit.Dp(8),
		Right:  unit.Dp(8),
	}
	th.Pad.Remove = layout.Inset{
		Top:    unit.Dp(6),
		Bottom: unit.Dp(6),
		Left:   unit.Dp(6),
		Right:  unit.Dp(6),
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
	TextSize      unit.Sp
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

// Layout draws the label.
func (l *labelStyle) Layout(gtx C) D {
	textMaterial := recording(gtx.Ops, func() {
		paint.ColorOp{Color: l.Color}.Add(gtx.Ops)
	})

	// Draw the text. Use minimum dimensions of 0 here to get the true size.
	mingtx := gtx
	mingtx.Constraints.Min = image.ZP
	label := widget.Label{MaxLines: 1}
	dim := label.Layout(mingtx, l.theme.Shaper, l.Font, l.TextSize, l.Text, textMaterial)

	// Draw strikethrough.
	if l.StrikeThrough {
		h := dim.Size.Y / 2
		rect := clip.Rect(image.Rect(0, h, dim.Size.X, h+gtx.Dp(2)))
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

func (e *editorStyle) Layout(gtx C) D {
	hintMaterial := recording(gtx.Ops, func() {
		paint.ColorOp{Color: e.theme.Color.HintText}.Add(gtx.Ops)
	})
	textMaterial := recording(gtx.Ops, func() {
		paint.ColorOp{Color: e.theme.Color.Item}.Add(gtx.Ops)
	})
	selectionMaterial := recording(gtx.Ops, func() {
		paint.ColorOp{Color: e.theme.Color.Selection}.Add(gtx.Ops)
	})

	// Draw hint label.
	var dims D
	showHint := recording(gtx.Ops, func() {
		tl := widget.Label{Alignment: e.Editor.Alignment}
		dims = tl.Layout(gtx, e.theme.Shaper, e.theme.Font.ItemHint, e.theme.Size.ItemText, e.Hint, hintMaterial)
	})

	// Expand minimum dimensions to fit the hint label.
	if w := dims.Size.X; gtx.Constraints.Min.X < w {
		gtx.Constraints.Min.X = w
	}
	if h := dims.Size.Y; gtx.Constraints.Min.Y < h {
		gtx.Constraints.Min.Y = h
	}

	// Draw editor.
	dims = e.Editor.Layout(gtx, e.theme.Shaper, e.theme.Font.Item, e.theme.Size.ItemText, textMaterial, selectionMaterial)
	if e.Editor.Len() == 0 {
		showHint.Add(gtx.Ops)
	}
	return dims
}

// Items.

type itemStyle struct {
	item    *item
	theme   *todoTheme
	label   labelStyle
	editor  editorStyle
	editing bool
}

// Item renders a todo item.
// When edit is non-nil, the item text is editable.
func (th *todoTheme) Item(item *item, edit *widget.Editor) itemStyle {
	s := itemStyle{
		item:  item,
		theme: th,
	}
	if edit != nil {
		s.editing = true
		s.editor = th.Editor(edit, "")
	} else {
		s.label = th.ItemLabel(item.text)
	}
	return s
}

// Layout draws an item.
func (it *itemStyle) Layout(gtx C) D {
	// Layout the item to get dimensions.
	r := op.Record(gtx.Ops)
	dim := it.theme.Pad.MainItem.Layout(gtx, func(gtx C) D {
		return it.item.click.Layout(gtx, it.layoutRow)
	})
	mac := r.Stop()

	// Put background under item when editing.
	if it.editing {
		bg := clip.Rect(image.Rectangle{Max: dim.Size})
		paint.FillShape(gtx.Ops, it.theme.Color.ItemEditBG, bg.Op())
	}

	// Now draw item over.
	mac.Add(gtx.Ops)
	return dim
}

// layoutRow draws an item.
func (it *itemStyle) layoutRow(gtx C) D {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		// Checkbox.
		layout.Rigid(func(gtx C) D {
			sz := gtx.Dp(it.theme.Size.Checkbox)
			gtx.Constraints = layout.Exact(image.Pt(sz, sz))
			return it.item.done.Layout(gtx, it.layoutCheckbox)
		}),
		// Item text.
		layout.Flexed(1, it.layoutText),
		// Remove button.
		layout.Rigid(func(gtx C) D {
			if !it.item.click.Hovered() {
				// Remove is visible only when mouse is on the item.
				return D{}
			}
			sz := gtx.Dp(it.theme.Size.Remove)
			gtx.Constraints = layout.Exact(image.Pt(sz, sz))
			return it.layoutRemoveButton(gtx)
		}),
	)
}

// layoutText draws the item text.
func (it *itemStyle) layoutText(gtx C) D {
	var textWidget layout.Widget
	if it.editing {
		textWidget = it.editor.Layout
	} else {
		label := it.label
		if it.item.done.Value {
			label.Color = it.theme.Color.ItemDone
			label.StrikeThrough = true
		}
		textWidget = label.Layout
	}

	dim := it.theme.Pad.Item.Layout(gtx, textWidget)

	// Label returns the minimum required size, but should really grab all available
	// space to make flex alignment work. Forcefully expand size to max width here.
	dim.Size.X = gtx.Constraints.Max.X
	return dim
}

// layoutCheckbox draws the checkbox.
func (it *itemStyle) layoutCheckbox(gtx C) D {
	var (
		spx    = gtx.Constraints.Min.X
		size   = image.Pt(spx, spx)
		rect   = image.Rectangle{Max: size}
		circle color.NRGBA
	)
	if it.item.done.Value {
		it.drawMark(gtx, rect, it.theme.Color.Checkmark)
		circle = it.theme.Color.Checkmark
		circle.A = 125
	} else {
		circle = it.theme.Color.Border
	}
	it.drawCircle(gtx, rect, circle)
	return D{Size: size}
}

// drawCircle draws the checkmark button outline.
func (it *itemStyle) drawCircle(gtx C, rect image.Rectangle, color color.NRGBA) {
	w := gtx.Dp(1.3)
	rect = rect.Inset(w) // Ensure outline is fully within rect.
	fillPath(gtx, clip.Ellipse(rect).Path(gtx.Ops), color, w)
}

// drawMark draws the checkmark button icon.
func (it *itemStyle) drawMark(gtx C, rect image.Rectangle, color color.NRGBA) {
	var (
		path  clip.Path
		w, h  = float32(rect.Dx()), float32(rect.Dy())
		start = f32.Pt(w-w/4, h/4)
		low   = f32.Pt(w/2.3, h-h/4.6)
		end   = f32.Pt(w/4, h-h/2.4)
	)
	path.Begin(gtx.Ops)
	path.MoveTo(start)
	path.LineTo(low)
	path.LineTo(end)
	fillPath(gtx, path.End(), color, gtx.Dp(1.8))
}

// layoutRemoveButton draws the item remove button.
func (it *itemStyle) layoutRemoveButton(gtx C) D {
	return it.item.remove.Layout(gtx, func(gtx C) D {
		// Add a background when hovering over the actual button.
		if it.item.remove.Hovered() {
			rect := image.Rectangle{Max: gtx.Constraints.Min}
			rr := clip.UniformRRect(rect, gtx.Dp(it.theme.Size.CornerRadius))
			paint.FillShape(gtx.Ops, it.theme.Color.RemoveBG, rr.Op(gtx.Ops))
		}
		return it.theme.Pad.Remove.Layout(gtx, it.layoutRemoveCross)
	})
}

// layoutRemoveCross draws the remove button icon.
func (it *itemStyle) layoutRemoveCross(gtx C) D {
	var (
		spx   = gtx.Constraints.Min.X
		size  = image.Pt(spx, spx)
		rect  = image.Rectangle{Max: size}
		color = it.theme.Color.Remove
		path  clip.Path
	)
	path.Begin(gtx.Ops)
	path.MoveTo(layout.FPt(rect.Min))
	path.LineTo(layout.FPt(rect.Max))
	fillPath(gtx, path.End(), color, gtx.Dp(1.8))
	path.Begin(gtx.Ops)
	path.MoveTo(layout.FPt(image.Pt(rect.Min.X, rect.Max.Y)))
	path.LineTo(layout.FPt(image.Pt(rect.Max.X, rect.Min.Y)))
	fillPath(gtx, path.End(), color, gtx.Dp(1.8))

	return D{Size: size}
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

func (b *buttonStyle) Layout(gtx C) D {
	border := widget.Border{CornerRadius: b.theme.Size.CornerRadius, Width: 1}
	if b.Active {
		border.Color = b.Border
	}

	return b.Button.Layout(gtx, func(gtx C) D {
		return border.Layout(gtx, func(gtx C) D {
			return b.theme.Pad.Button.Layout(gtx, b.Label.Layout)
		})
	})
}

// showIf draws w if cond is true.
func showIf(cond bool, gtx C, w layout.Widget) D {
	m := op.Record(gtx.Ops)
	dim := w(gtx)
	call := m.Stop()
	if cond {
		call.Add(gtx.Ops)
	}
	return dim
}

// fillPath draws the line of p using the given color and stroke width.
func fillPath(gtx C, p clip.PathSpec, color color.NRGBA, width int) {
	w := float32(width)
	paint.FillShape(gtx.Ops, color, clip.Stroke{Path: p, Width: w}.Op())
}

func recording(ops *op.Ops, f func()) op.CallOp {
	rec := op.Record(ops)
	f()
	return rec.Stop()
}
