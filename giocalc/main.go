package main

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"strings"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	. "github.com/fjl/gio-demos/internal/cd"
)

var (
	digitColor       = color.NRGBA{90, 90, 90, 255}
	specialColor     = color.NRGBA{70, 70, 70, 255}
	opColor          = color.NRGBA{122, 90, 90, 255}
	activeOpColor    = color.NRGBA{160, 90, 90, 255}
	backgroundColor  = color.NRGBA{50, 50, 50, 255}
	resultColor      = color.NRGBA{255, 255, 255, 255}
	resultBackground = color.NRGBA{35, 35, 35, 255}

	designWidth  = unit.Dp(270)
	designHeight = unit.Dp(345)
	controlInset = unit.Dp(6)
	cornerRadius = unit.Dp(3.5)
)

// calcUI is the user interface of the calculator.
type calcUI struct {
	calc     calculator
	theme    *material.Theme
	buttons  [5][4]*button
	evFilter []event.Filter

	cornerRadius int
	gridSpacing  int
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
	ui.evFilter = ui.makeEventFilter()
	return ui
}

// digit creates a digit button.
func (ui *calcUI) digit(input string) *button {
	b := newButton(&ui.calc, input, digitColor)
	b.action = func() { ui.calc.digit(input) }
	b.op = opNop
	return b
}

// op creates an operation button.
func (ui *calcUI) op(op calcOp) *button {
	b := newButton(&ui.calc, op.String(), opColor)
	b.action = func() { ui.calc.run(op) }
	b.op = op
	return b
}

// special creates a special operation button.
func (ui *calcUI) special(name string, fn func()) *button {
	b := newButton(&ui.calc, name, specialColor)
	b.action = fn
	b.op = opNop
	return b
}

// Layout draws the UI.
func (ui *calcUI) Layout(gtx C) D {
	// Adapt design for screen size.
	scaleFactor := float32(gtx.Constraints.Max.X) / float32(gtx.Dp(designWidth))
	ui.cornerRadius = gtx.Dp(cornerRadius * unit.Dp(scaleFactor))
	ui.gridSpacing = gtx.Dp(controlInset * unit.Dp(scaleFactor))

	// Handle key events.
	ui.layoutInput(gtx)

	inset := layout.UniformInset(controlInset)
	return inset.Layout(gtx, func(gtx C) D {
		flex := layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceStart}
		return flex.Layout(gtx,
			layout.Flexed(20, func(gtx C) D {
				return inset.Layout(gtx, ui.layoutResult)
			}),
			layout.Flexed(70, func(gtx C) D {
				return inset.Layout(gtx, ui.layoutButtons)
			}),
		)
	})
}

func (ui *calcUI) layoutResult(gtx C) D {
	rect := image.Rectangle{Max: gtx.Constraints.Max}
	rr := clip.UniformRRect(rect, ui.cornerRadius)
	paint.FillShape(gtx.Ops, resultBackground, rr.Op(gtx.Ops))

	inset := layout.UniformInset(controlInset)
	return inset.Layout(gtx, ui.layoutResultText)
}

func (ui *calcUI) layoutResultText(gtx C) D {
	// Scale font based on height.
	fontSizePx := float32(gtx.Constraints.Max.Y) / 1.1
	fontSizeSp := unit.Sp(fontSizePx / gtx.Metric.PxPerSp)

	l := material.Label(ui.theme, fontSizeSp, ui.calc.text())
	l.Color = resultColor
	l.Alignment = text.End
	return shrinkToFit(gtx, l.Layout)
}

func (ui *calcUI) layoutButtons(gtx C) D {
	g := grid{
		rows:    len(ui.buttons),
		cols:    len(ui.buttons[0]),
		spacing: ui.gridSpacing,
	}
	return g.layout(gtx, func(row, col int, gtx C) D {
		if b := ui.buttons[row][col]; b != nil {
			return ui.layoutButton(gtx, b)
		}
		return D{}
	})
}

func (ui *calcUI) layoutButton(gtx C, b *button) D {
	if b.clicker.Clicked(gtx) && b.action != nil {
		b.action()
	}

	return b.clicker.Layout(gtx, func(gtx C) D {
		textSizePx := float32(gtx.Constraints.Max.Y) / 2.2
		textSizeSp := unit.Sp(textSizePx / gtx.Metric.PxPerSp)

		style := material.Button(ui.theme, &b.clicker, b.text)
		style.Background = b.color
		style.Inset = layout.Inset{}
		style.TextSize = textSizeSp
		style.CornerRadius = unit.Dp(float32(ui.cornerRadius) / gtx.Metric.PxPerDp)
		if b.calc.lastOp == b.op {
			style.Background = activeOpColor
		}
		return style.Layout(gtx)
	})
}

// layoutInput registers the global key handler.
func (ui *calcUI) layoutInput(gtx C) {
	event.Op(gtx.Ops, ui)
	for {
		ev, ok := gtx.Event(ui.evFilter...)
		if !ok {
			break
		}
		switch ev := ev.(type) {
		case key.Event:
			switch {
			case isCopy(ev):
				text := io.NopCloser(strings.NewReader(ui.calc.text()))
				gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: text})
			case isPaste(ev):
				gtx.Execute(clipboard.ReadCmd{Tag: ui})
			default:
				ui.handleKey(ev)
			}
		case transfer.DataEvent:
			r := ev.Open()
			text, _ := io.ReadAll(io.LimitReader(r, 4096))
			r.Close()
			ui.calc.parse(string(text))
		default:
			fmt.Printf("unhandled event type %T\n", ev)
		}
	}
}

func (ui *calcUI) makeEventFilter() []event.Filter {
	return []event.Filter{
		// Clipboard events.
		transfer.TargetFilter{Target: ui, Type: "application/text"},

		// Steal tab key events from Gio to avoid automatic focus handling.
		key.Filter{Name: key.NameTab, Optional: key.ModShift},

		// Button keys
		key.Filter{Name: "0"},
		key.Filter{Name: "1"},
		key.Filter{Name: "2"},
		key.Filter{Name: "3"},
		key.Filter{Name: "4"},
		key.Filter{Name: "5"},
		key.Filter{Name: "6"},
		key.Filter{Name: "7"},
		key.Filter{Name: "8"},
		key.Filter{Name: "9"},
		key.Filter{Name: ".", Optional: key.ModShift},
		key.Filter{Name: ",", Optional: key.ModShift},
		key.Filter{Name: "+", Optional: key.ModShift},
		key.Filter{Name: "*", Optional: key.ModShift},
		key.Filter{Name: "-", Optional: key.ModAlt | key.ModShift},
		key.Filter{Name: "/", Optional: key.ModShift},
		key.Filter{Name: "%", Optional: key.ModShift},
		key.Filter{Name: "=", Optional: key.ModShift},
		key.Filter{Name: key.NameReturn},
		key.Filter{Name: key.NameEnter},
		key.Filter{Name: key.NameEscape},
		key.Filter{Name: key.NameDeleteBackward},
		key.Filter{Name: key.NameDeleteForward},

		// Copy/Paste
		key.Filter{Name: "C", Required: key.ModShortcut},
		key.Filter{Name: "V", Required: key.ModShortcut},
	}
}

func isCopy(e key.Event) bool {
	return e.Name == "C" && e.Modifiers.Contain(key.ModShortcut)
}

func isPaste(e key.Event) bool {
	return e.Name == "V" && e.Modifiers.Contain(key.ModShortcut)
}

// handleKey handles a key event.
func (ui *calcUI) handleKey(e key.Event) {
	if e.State == key.Release {
		return
	}

	switch e.Name {
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", ".":
		ui.calc.digit(string(e.Name))
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
	calc   *calculator
	op     calcOp
	text   string
	action func()

	color   color.NRGBA
	clicker widget.Clickable
}

func newButton(calc *calculator, text string, color color.NRGBA) *button {
	return &button{calc: calc, text: text, color: color}
}

func main() {
	var (
		size     = app.Size(designWidth, designHeight)
		statusBg = app.StatusColor(backgroundColor)
		sysBg    = app.NavigationColor(backgroundColor)
		title    = app.Title("GioCalc")
		portrait = app.PortraitOrientation.Option()
	)
	go func() {
		w := app.NewWindow(statusBg, sysBg, size, title, portrait)
		// w.Option(app.MaxSize(designWidth, designHeight))
		w.Option(app.MinSize(designWidth, designHeight))

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
	// Use Go font because the design is kind of built for it.
	theme := material.NewTheme()
	gofonts := gofont.Collection()
	theme.Shaper = text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofonts))

	var (
		ui  = newUI(theme)
		ops op.Ops
	)
	for {
		e := w.NextEvent()
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			paint.Fill(gtx.Ops, backgroundColor)
			ui.Layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
	return nil
}
