package main

import "gioui.org/widget"

const (
	filterAll int = iota
	filterActive
	filterCompleted
)

type item struct {
	text string
	done widget.Bool
}

type todos struct {
	items []*item
}

func (t *todos) add(text string) {
	t.items = append(t.items, &item{text: text})
}

func (t *todos) filter(filter int) []*item {
	if filter == filterAll {
		return t.items
	}
	var result []*item
	for _, item := range t.items {
		if filter == filterCompleted == item.done.Value {
			result = append(result, item)
		}
	}
	return result
}

func (t *todos) count() (todo, done int) {
	for _, item := range t.items {
		if item.done.Value {
			done++
		} else {
			todo++
		}
	}
	return todo, done
}

func (t *todos) clearDone() {
	t.items = t.filter(filterActive)
}
