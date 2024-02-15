package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/fjl/gio-demos/giotodo/internal/todostore"

	. "github.com/fjl/gio-demos/internal/cd"
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

	// Item editing.
	itemBeingEdited    *item
	itemEditor         widget.Editor
	editFocusRequested bool
	initialFocus       bool
}

func newTodoUI(theme *todoTheme, model *todoModel) *todoUI {
	ui := &todoUI{
		todos:     model,
		filter:    filterAll,
		theme:     theme,
		mainInput: widget.Editor{Submit: true, SingleLine: true, InputHint: key.HintText},
		list:      layout.List{Axis: layout.Vertical},
	}
	return ui
}

// Layout draws the app.
func (ui *todoUI) Layout(gtx C) D {
	// Set focus to the input line initially.
	if !ui.initialFocus {
		gtx.Execute(key.FocusCmd{Tag: &ui.mainInput})
		ui.initialFocus = true
	}

	// Process submissions.
	for {
		e, ok := ui.mainInput.Update(gtx)
		if !ok {
			break
		}
		switch e := e.(type) {
		case widget.SubmitEvent:
			newItem := strings.TrimSpace(e.Text)
			if newItem != "" {
				ui.submit(newItem)
			}
		}
	}
	// Process clear.
	if ui.clear.Clicked(gtx) {
		ui.todos.clearDone()
	}
	// Process filter selection.
	switch {
	case ui.all.Clicked(gtx):
		ui.filter = filterAll
	case ui.active.Clicked(gtx):
		ui.filter = filterActive
	case ui.completed.Clicked(gtx):
		ui.filter = filterCompleted
	}

	// Draw.
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return ui.theme.Pad.Main.Layout(gtx, ui.layoutInput)
		}),
		layout.Flexed(1.0, func(gtx C) D {
			return ui.layoutItems(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return ui.theme.Pad.Main.Layout(gtx, ui.layoutStatusBar)
		}),
	)
}

// layoutInput draws the main input line.
func (ui *todoUI) layoutInput(gtx C) D {
	ed := ui.theme.Editor(&ui.mainInput, "What needs to be done?")
	return ed.Layout(gtx)
}

// layoutItems draws the current items.
func (ui *todoUI) layoutItems(gtx C) D {
	items := ui.todos.filteredItems(ui.filter)

	// Process other item actions.
	for _, item := range items {
		if doubleClicked(&item.click, gtx) {
			ui.startItemEdit(gtx, item)
		}
		if item.done.Update(gtx) {
			ui.todos.itemUpdated(item)
		}
		if item.remove.Clicked(gtx) {
			ui.todos.remove(item)
		}
	}

	if ui.itemBeingEdited != nil {
		// Item editing should end (and the item be updated) when itemEditor loses focus.
		// Requesting focus uses events, and a simple check for Focused(itemEditor) returning
		// false doesn't work: it would also trigger when editing just started but focus
		// hasn't been granted yet. To make it work, we call endItemEdit only when focus is
		// not being requested.
		foc := gtx.Focused(&ui.itemEditor)
		switch {
		case foc && ui.editFocusRequested:
			ui.editFocusRequested = false
		case !foc && !ui.editFocusRequested:
			ui.endItemEdit()
		}
		// Submit events also end the edit operation.
		for {
			e, ok := ui.itemEditor.Update(gtx)
			if !ok {
				break
			}
			switch e.(type) {
			case widget.SubmitEvent:
				ui.endItemEdit()
			}
		}
	}

	// Draw the list.
	return ui.list.Layout(gtx, len(items), func(gtx C, i int) D {
		item := items[i]
		var e *widget.Editor
		if item == ui.itemBeingEdited {
			e = &ui.itemEditor
		}
		w := ui.theme.Item(item, e)
		return w.Layout(gtx)
	})
}

func doubleClicked(c *widget.Clickable, gtx C) (clicked bool) {
	for {
		cl, ok := c.Update(gtx)
		if !ok {
			break
		}
		if cl.NumClicks >= 2 {
			clicked = true
		}
	}
	return clicked
}

// layoutStatusBar draws the status bar at the bottom.
func (ui *todoUI) layoutStatusBar(gtx C) D {
	doneCount := ui.todos.doneCount()
	count := ui.todos.len() - doneCount

	flex := layout.Flex{
		Axis:    layout.Horizontal,
		Spacing: layout.SpaceBetween,
	}
	return flex.Layout(gtx,
		layout.Rigid(func(gtx C) D {
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
		layout.Flexed(1.0, func(gtx C) D {
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
		layout.Rigid(func(gtx C) D {
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

func (ui *todoUI) startItemEdit(gtx C, item *item) {
	if ui.itemBeingEdited == item {
		// Already editing this item.
		return
	} else if ui.itemBeingEdited != nil {
		// Cancel previous edit first.
		ui.endItemEdit()
	}

	fmt.Println("start editing item:", item.text)

	// Configure the editor.
	ui.itemBeingEdited = item
	ui.itemEditor = widget.Editor{Submit: true, SingleLine: true, InputHint: key.HintText}
	ui.itemEditor.SetText(item.text)
	length := ui.itemEditor.Len()
	ui.itemEditor.SetCaret(length, length)
	gtx.Execute(key.FocusCmd{Tag: &ui.itemEditor})
	ui.editFocusRequested = true
}

func (ui *todoUI) endItemEdit() {
	if ui.itemBeingEdited == nil {
		return
	}
	text := ui.itemEditor.Text()
	fmt.Println("end editing item:", text)
	ui.itemBeingEdited.text = text
	ui.todos.itemUpdated(ui.itemBeingEdited)
	ui.itemBeingEdited = nil
	ui.editFocusRequested = true
}

func main() {
	go func() {
		var (
			theme    = newTodoTheme()
			title    = app.Title("GioTodo")
			size     = app.Size(theme.Size.PrefWidth, unit.Dp(600))
			minSize  = app.MinSize(theme.Size.MinWidth, unit.Dp(250))
			statusBg = app.StatusColor(theme.Color.Background)
			navBg    = app.NavigationColor(theme.Color.Background)
			window   = app.NewWindow(title, size, minSize, statusBg, navBg)
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
	datadir, err := app.DataDir()
	if err != nil {
		return err
	}

	var (
		storedir = filepath.Join(datadir, "giotodo")
		store    = todostore.NewStore(storedir, w.Invalidate)
		model    = newTodoModel(store)
		ui       = newTodoUI(theme, model)
		ops      op.Ops
	)
	defer store.Close()

	for {
		for _, e := range store.Events() {
			model.handleStoreEvent(e)
			w.Invalidate()
		}
		e := w.NextEvent()
		switch e := e.(type) {
		case app.StageEvent:
			if e.Stage == app.StagePaused {
				store.Persist()
			}
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			paint.Fill(gtx.Ops, ui.theme.Color.MainPanel)
			ui.Layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}
