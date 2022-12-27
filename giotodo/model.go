package main

import (
	"container/list"
	"fmt"

	"gioui.org/widget"
	"github.com/fjl/gio-demos/giotodo/internal/todostore"
)

const (
	filterInvalid itemFilter = iota
	filterAll
	filterActive
	filterCompleted
)

type itemFilter int

type item struct {
	id   todostore.ID
	elem *list.Element
	text string

	// UI state.
	done   widget.Bool
	remove widget.Clickable
	click  widget.Clickable
}

type todoModel struct {
	store     *todostore.Store
	items     map[todostore.ID]*item
	all       *list.List
	lastError error

	// This is the cache for filteredItems.
	cachedList       []*item
	cachedListFilter itemFilter
}

func newTodoModel(store *todostore.Store) *todoModel {
	return &todoModel{
		store: store,
		all:   list.New(),
		items: make(map[todostore.ID]*item),
	}
}

func (m *todoModel) handleStoreEvent(e todostore.Event) {
	switch e := e.(type) {
	case *todostore.ItemAdded:
		it := &item{id: e.ID, text: e.Item.Text}
		it.done.Value = e.Item.Done
		it.elem = m.all.PushBack(it)
		m.items[e.ID] = it
		if m.cachedListFilter.match(it) {
			m.cachedList = append(m.cachedList, it)
		}

	case *todostore.ItemRemoved:
		it := m.items[e.ID]
		m.all.Remove(it.elem)
		delete(m.items, e.ID)
		if m.cachedListFilter.match(it) {
			m.cachedListFilter = filterInvalid
		}

	case *todostore.ItemChanged:
		it := m.items[e.ID]
		it.done.Value = e.Item.Done
		it.text = e.Item.Text
		m.cachedListFilter = filterInvalid

	case *todostore.IOError:
		m.lastError = e.Err
	}
}

func (m *todoModel) len() int {
	return m.all.Len()
}

func (m *todoModel) doneCount() int {
	count := 0
	for _, it := range m.items {
		if it.done.Value {
			count++
		}
	}
	return count
}

// filteredItems returns all items that match the given filter.
func (m *todoModel) filteredItems(filter itemFilter) []*item {
	if filter == filterInvalid {
		panic("filteredItems(filterInvalid)")
	}
	if filter == m.cachedListFilter {
		return m.cachedList // unchanged
	}

	m.cachedList = m.cachedList[:0]
	m.cachedListFilter = filter
	for elem := m.all.Front(); elem != nil; elem = elem.Next() {
		it := elem.Value.(*item)
		if filter.match(it) {
			m.cachedList = append(m.cachedList, it)
		}
	}
	return m.cachedList
}

// match tells whether an item matches the filter.
func (f itemFilter) match(it *item) bool {
	switch f {
	case filterInvalid:
		return false
	case filterAll:
		return true
	case filterActive:
		return !it.done.Value
	case filterCompleted:
		return it.done.Value
	default:
		panic(fmt.Errorf("invalid filter %d", f))
	}
}

func (m *todoModel) setItemDone(it *item, done bool) {
	m.store.UpdateItem(it.id, todostore.Item{
		Text: it.text,
		Done: it.done.Value,
	})
}

func (m *todoModel) clearDone() {
	for _, it := range m.items {
		if it.done.Value {
			m.store.RemoveItem(it.id)
		}
	}
}

func (m *todoModel) add(text string) {
	m.store.AddItem(todostore.Item{Text: text})
}

func (m *todoModel) remove(it *item) {
	m.store.RemoveItem(it.id)
}
