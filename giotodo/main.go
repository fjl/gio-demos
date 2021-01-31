package main

import (
	"fmt"
	"image"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
)

type todoUI struct {
	todos  todos
	filter int

	theme     *todoTheme
	mainInput widget.Editor
	list      layout.List
	all       widget.Clickable
	active    widget.Clickable
	completed widget.Clickable
	clear     widget.Clickable
}

func newTodoUI(theme *todoTheme) *todoUI {
	ui := &todoUI{
		theme: theme,
		mainInput: widget.Editor{
			SingleLine: true,
			Submit:     true,
		},
		list: layout.List{Axis: layout.Vertical},
	}
	ui.todos.add("foo")
	ui.todos.add("bar")
	ui.todos.items[0].done.Value = true
	return ui
}

// Layout draws the app.
func (ui *todoUI) Layout(gtx layout.Context) layout.Dimensions {
	// Process submissions.
	for _, e := range ui.mainInput.Events() {
		switch e := e.(type) {
		case widget.SubmitEvent:
			ui.submit(e.Text)
		}
	}
	// Process clear.
	if ui.clear.Clicked() {
		ui.todos.clearDone()
	}
	// Process filter selection.
	switch {
	case ui.all.Clicked():
		ui.filter = filterAll
	case ui.active.Clicked():
		ui.filter = filterActive
	case ui.completed.Clicked():
		ui.filter = filterCompleted
	}

	// Draw.
	gtx.Constraints.Min.X = gtx.Px(ui.theme.Size.MainWidth)
	gtx.Constraints.Max.X = gtx.Px(ui.theme.Size.MainWidth)
	rect := clip.Rect{Max: gtx.Constraints.Max}
	paint.FillShape(gtx.Ops, ui.theme.Color.MainPanel, rect.Op())
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.theme.Pad.Main.Layout(gtx, ui.layoutInput)
		}),
		layout.Flexed(1.0, func(gtx layout.Context) layout.Dimensions {
			return ui.theme.Pad.Main.Layout(gtx, ui.layoutItems)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.theme.Pad.Main.Layout(gtx, ui.layoutStatusBar)
		}),
	)
}

// layoutInput draws the main input line.
func (ui *todoUI) layoutInput(gtx layout.Context) layout.Dimensions {
	ed := ui.theme.Editor(&ui.mainInput, "What needs to be done?")
	return ed.Layout(gtx)
}

// layoutItems draws the current items.
func (ui *todoUI) layoutItems(gtx layout.Context) layout.Dimensions {
	items := ui.todos.filter(ui.filter)
	return ui.list.Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
		item := items[i]
		box := ui.theme.Item(item.text, &item.done, nil)
		return box.Layout(gtx)
	})
}

// layoutStatusBar draws the status bar at the bottom.
func (ui *todoUI) layoutStatusBar(gtx layout.Context) layout.Dimensions {
	count, doneCount := ui.todos.count()

	dim := layout.SW.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		l := ui.theme.StatusLabel("1 item left")
		l.Alignment = text.Start
		if count != 1 {
			l.Text = fmt.Sprintf("%d items left", count)
		}
		return ui.theme.Pad.Button.Layout(gtx, l.Layout)
	})
	layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		all := ui.theme.StatusButton(&ui.all, "All", ui.filter == filterAll)
		active := ui.theme.StatusButton(&ui.active, "Active", ui.filter == filterActive)
		completed := ui.theme.StatusButton(&ui.completed, "Completed", ui.filter == filterCompleted)
		flex := layout.Flex{Alignment: layout.Baseline, Spacing: layout.SpaceEvenly}
		return flex.Layout(gtx,
			layout.Rigid(all.Layout),
			layout.Rigid(active.Layout),
			layout.Rigid(completed.Layout),
		)
	})
	layout.SE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		clear := ui.theme.Clickable(&ui.clear, "Clear completed")
		clear.Label.Alignment = text.End
		return showIf(doneCount > 0, gtx, clear.Layout)
	})
	return layout.Dimensions{
		Size:     image.Pt(gtx.Constraints.Max.X, dim.Size.Y),
		Baseline: dim.Baseline,
	}
}

// submit is called when a todo item is submitted.
func (ui *todoUI) submit(line string) {
	ui.mainInput.SetText("")
	ui.todos.add(line)
}

func main() {
	go func() {
		var (
			theme  = newTodoTheme(gofont.Collection())
			title  = app.Title("GioTodo")
			min    = app.MinSize(theme.Size.MainWidth, unit.Dp(200))
			window = app.NewWindow(min, title)
		)
		if err := loop(window, theme); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

// loop is the main loop of the app.
func loop(w *app.Window, theme *todoTheme) error {
	var (
		ui  = newTodoUI(theme)
		ops op.Ops
	)
	for {
		select {
		case e := <-w.Events():
			switch e := e.(type) {
			case system.DestroyEvent:
				return e.Err
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)
				paint.Fill(gtx.Ops, ui.theme.Color.Background)
				layout.Center.Layout(gtx, ui.Layout)
				e.Frame(gtx.Ops)
			}
		}
	}
}
