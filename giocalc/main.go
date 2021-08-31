package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"runtime"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var (
	designWidth     = unit.Dp(270)
	designHeight    = unit.Dp(286)
	digitColor      = color.NRGBA{90, 90, 90, 255}
	specialColor    = color.NRGBA{70, 70, 70, 255}
	opColor         = color.NRGBA{122, 90, 90, 255}
	activeOpColor   = color.NRGBA{160, 90, 90, 255}
	backgroundColor = color.NRGBA{50, 50, 50, 255}
	resultColor     = color.NRGBA{255, 255, 255, 255}
	controlInset    = unit.Dp(6)
	resultHeight    = unit.Dp(80)
)

// calcUI is the user interface of the calculator.
type calcUI struct {
	calc    calculator
	theme   *material.Theme
	buttons [5][4]*button
}

func newUI(theme *material.Theme) *calcUI {
	ui := &calcUI{theme: theme}
	reset := ui.special("AC", ui.calc.reset)
	sign := ui.special("±", ui.calc.flipSign)
	percent := ui.special("%", ui.calc.percent)
	decimal := ui.special(".", func() { ui.calc.digit(".") })
	ui.buttons = [5][4]*button{
		{reset, sign, percent, ui.op(opDiv)},
		{ui.digit("7"), ui.digit("8"), ui.digit("9"), ui.op(opMul)},
		{ui.digit("4"), ui.digit("5"), ui.digit("6"), ui.op(opSub)},
		{ui.digit("1"), ui.digit("2"), ui.digit("3"), ui.op(opAdd)},
		{ui.digit("0"), nil, decimal, ui.op(opEq)},
	}
	return ui
}

// digit creates a digit button.
func (ui *calcUI) digit(input string) *button {
	b := newButton(&ui.calc, ui.theme, input, digitColor)
	b.action = func() { ui.calc.digit(input) }
	b.op = opNop
	return b
}

// op creates an operation button.
func (ui *calcUI) op(op calcOp) *button {
	b := newButton(&ui.calc, ui.theme, op.String(), opColor)
	b.action = func() { ui.calc.run(op) }
	b.op = op
	return b
}

// special creates a special operation button.
func (ui *calcUI) special(name string, fn func()) *button {
	b := newButton(&ui.calc, ui.theme, name, specialColor)
	b.action = fn
	b.op = opNop
	return b
}

func (ui *calcUI) Layout(gtx layout.Context) layout.Dimensions {
	scale := float32(gtx.Constraints.Max.X) / float32(gtx.Px(designWidth))
	tr := f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(scale, scale))
	op.Affine(tr).Add(gtx.Ops)
	gtx.Constraints.Min.X = gtx.Px(designWidth)
	gtx.Constraints.Max.X = gtx.Px(designWidth)
	gtx.Constraints.Max.Y = int(float32(gtx.Constraints.Max.Y)/scale + 0.5)
	height := float32(gtx.Constraints.Max.Y)
	inset := layout.Inset{}
	if height > float32(gtx.Px(designHeight)) {
		inset.Top = unit.Px(height - float32(gtx.Px(designHeight)))
	}
	return inset.Layout(gtx, ui.layout)
}

// layout draws the UI.
func (ui *calcUI) layout(gtx layout.Context) layout.Dimensions {
	// Draw the result and buttons.
	inset := layout.UniformInset(controlInset)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, gtx.Px(resultHeight)))
			return inset.Layout(gtx, ui.layoutResult)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return inset.Layout(gtx, ui.layoutButtons)
		}),
	)
}

func (ui *calcUI) layoutResult(gtx layout.Context) layout.Dimensions {
	l := material.Label(ui.theme, unit.Px(float32(gtx.Constraints.Max.Y)), ui.calc.text())
	l.Color = resultColor
	l.Alignment = text.End
	return shrinkToFit(gtx, l.Layout)
}

func (ui *calcUI) layoutButtons(gtx layout.Context) layout.Dimensions {
	g := grid{rows: len(ui.buttons), cols: len(ui.buttons[0]), spacing: controlInset}
	return g.layout(gtx, func(row, col int, gtx layout.Context) layout.Dimensions {
		if b := ui.buttons[row][col]; b != nil {
			return b.Layout(gtx)
		}
		return layout.Dimensions{}
	})
}

// handleKey handles a key event.
func (ui *calcUI) handleKey(e key.Event) {
	if e.State == key.Release {
		return
	}
	switch e.Name {
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", ".":
		ui.calc.digit(e.Name)
	case "+":
		ui.calc.run(opAdd)
	case "-":
		if e.Modifiers.Contain(key.ModAlt) {
			ui.calc.flipSign()
		} else {
			ui.calc.run(opSub)
		}
	case "*":
		ui.calc.run(opMul)
	case "/":
		ui.calc.run(opDiv)
	case "%":
		ui.calc.percent()
	case "=", key.NameEnter, key.NameReturn:
		ui.calc.run(opEq)
	case key.NameDeleteBackward, key.NameDeleteForward:
		ui.calc.rubout()
	case key.NameEscape:
		ui.calc.reset()
	}
}

// button is a clickable button.
type button struct {
	calc    *calculator
	clicker widget.Clickable
	style   material.ButtonStyle
	action  func()
	op      calcOp
}

func newButton(calc *calculator, theme *material.Theme, text string, color color.NRGBA) *button {
	b := &button{calc: calc}
	b.style = material.Button(theme, &b.clicker, text)
	b.style.Background = color
	return b
}

// Layout draws the button.
func (b *button) Layout(gtx layout.Context) layout.Dimensions {
	// Check for mouse events.
	b.clicker.Layout(gtx)
	if b.clicker.Clicked() && b.action != nil {
		b.action()
	}
	// Draw the button.
	style := b.style
	if b.calc.lastOp == b.op {
		style.Background = activeOpColor
	}
	return style.Layout(gtx)
}

func main() {
	var (
		min = app.MinSize(designWidth, designHeight)
		// max   = app.MaxSize(designWidth, designHeight)
		size  = app.Size(designWidth, designHeight)
		title = app.Title("GioCalc")
	)
	go func() {
		w := app.NewWindow(min, size, title)
		if err := loop(w); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}

// loop is the main loop of the app.
func loop(w *app.Window) error {
	var (
		th  = material.NewTheme(gofont.Collection())
		ui  = newUI(th)
		ops op.Ops
	)

	for e := range w.Events() {
		switch e := e.(type) {
		case system.DestroyEvent:
			return e.Err
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			paint.Fill(gtx.Ops, backgroundColor)
			ui.Layout(gtx)
			e.Frame(gtx.Ops)
		case key.Event:
			switch {
			case isCopy(e):
				w.WriteClipboard(ui.calc.text())
			case isPaste(e):
				w.ReadClipboard()
			default:
				ui.handleKey(e)
				w.Invalidate()
			}
		case clipboard.Event:
			ui.calc.parse(e.Text)
			w.Invalidate()
		}
	}
	return nil
}

func isCopy(e key.Event) bool {
	mod := key.ModCtrl
	if runtime.GOOS == "darwin" {
		mod = key.ModCommand
	}
	return e.Name == "C" && e.Modifiers.Contain(mod)
}

func isPaste(e key.Event) bool {
	mod := key.ModCtrl
	if runtime.GOOS == "darwin" {
		mod = key.ModCommand
	}
	return e.Name == "V" && e.Modifiers.Contain(mod)
}
