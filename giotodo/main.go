package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/fjl/gio-demos/giotodo/internal/todostore"
)

type todoUI struct {
	todos  *todoModel
	filter itemFilter

	// UI elements.
	theme     *todoTheme
	mainInput widget.Editor
	list      layout.List
	all       widget.Clickable
	active    widget.Clickable
	completed widget.Clickable
	clear     widget.Clickable
}

func newTodoUI(theme *todoTheme, model *todoModel) *todoUI {
	ui := &todoUI{
		todos:     model,
		filter:    filterAll,
		theme:     theme,
		mainInput: widget.Editor{Submit: true, SingleLine: true, InputHint: key.HintText},
		list:      layout.List{Axis: layout.Vertical},
	}
	ui.mainInput.Focus()
	return ui
}

// Layout draws the app.
func (ui *todoUI) Layout(gtx layout.Context) layout.Dimensions {
	// Process submissions.
	for _, e := range ui.mainInput.Events() {
		switch e := e.(type) {
		case widget.SubmitEvent:
			newItem := strings.TrimSpace(e.Text)
			if newItem != "" {
				ui.submit(newItem)
			}
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
	items := ui.todos.filteredItems(ui.filter)

	// Process item actions.
	for _, item := range items {
		if item.done.Changed() {
			ui.todos.setItemDone(item, item.done.Value)
		}
		if item.remove.Clicked() {
			ui.todos.remove(item)
		}
	}

	// Draw the list.
	return ui.list.Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
		item := ui.theme.Item(items[i])
		return item.Layout(gtx)
	})
}

// layoutStatusBar draws the status bar at the bottom.
func (ui *todoUI) layoutStatusBar(gtx layout.Context) layout.Dimensions {
	doneCount := ui.todos.doneCount()
	count := ui.todos.len() - doneCount

	flex := layout.Flex{
		Axis:    layout.Horizontal,
		Spacing: layout.SpaceBetween,
	}
	return flex.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := ui.theme.StatusLabel("")
			if ui.todos.lastError != nil {
				label.Text = ui.todos.lastError.Error()
				label.Color = ui.theme.Color.Error
			} else {
				if ui.filter == filterCompleted {
					label.Text = fmt.Sprintf("%d done.", doneCount)
				} else {
					label.Text = fmt.Sprintf("%d to do.", count)
				}
			}
			label.Alignment = text.Start
			return ui.theme.Pad.Button.Layout(gtx, label.Layout)
		}),
		layout.Flexed(1.0, func(gtx layout.Context) layout.Dimensions {
			all := ui.theme.StatusButton(&ui.all, "All", ui.filter == filterAll)
			active := ui.theme.StatusButton(&ui.active, "Active", ui.filter == filterActive)
			completed := ui.theme.StatusButton(&ui.completed, "Done", ui.filter == filterCompleted)
			flex := layout.Flex{Alignment: layout.Baseline, Spacing: layout.SpaceSides}
			return flex.Layout(gtx,
				layout.Rigid(all.Layout),
				layout.Rigid(active.Layout),
				layout.Rigid(completed.Layout),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			clear := ui.theme.Clickable(&ui.clear, "Clear")
			clear.Label.Alignment = text.End
			return showIf(doneCount > 0, gtx, clear.Layout)
		}),
	)
}

// submit is called when a todo item is submitted.
func (ui *todoUI) submit(line string) {
	ui.mainInput.SetText("")
	ui.todos.add(line)
}

func main() {
	go func() {
		var (
			theme    = newTodoTheme(gofont.Collection())
			title    = app.Title("GioTodo")
			size     = app.Size(theme.Size.PrefWidth, unit.Dp(600))
			statusBg = app.StatusColor(theme.Color.Background)
			navBg    = app.NavigationColor(theme.Color.Background)
			window   = app.NewWindow(title, size, statusBg, navBg)
		)
		window.Option(app.MinSize(theme.Size.MinWidth, unit.Dp(250)))

		if err := loop(window, theme); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

// loop is the main loop of the app.
func loop(w *app.Window, theme *todoTheme) error {
	datadir, err := app.DataDir()
	if err != nil {
		return err
	}
	store := todostore.NewStore(filepath.Join(datadir, "giotodo"))
	defer store.Close()

	var (
		model = newTodoModel(store)
		ui    = newTodoUI(theme, model)
		ops   op.Ops
	)
	for {
		select {
		case e := <-store.Events():
			model.handleStoreEvent(e)
			w.Invalidate()
		case e := <-w.Events():
			switch e := e.(type) {
			case system.StageEvent:
				if e.Stage == system.StagePaused {
					store.Persist()
				}
			case system.DestroyEvent:
				return e.Err
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)
				paint.Fill(gtx.Ops, ui.theme.Color.MainPanel)
				ui.Layout(gtx)
				e.Frame(gtx.Ops)
			}
		}
	}
}
