package todostore

import (
	"container/list"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type ID string

func randomID() ID {
	s := make([]byte, 16)
	if _, err := crand.Read(s); err != nil {
		panic(err)
	}
	return ID(hex.EncodeToString(s))
}

// Item is a todo item.
type Item struct {
	Text string
	Done bool
}

type Store struct {
	dataDir  string
	dataFile *os.File
	reader   *json.Decoder
	writer   *json.Encoder

	eventsOut  chan Event
	eventQueue list.List

	eventsIn chan Event
	flushCh  chan struct{}
	quitCh   chan struct{}
	wg       sync.WaitGroup
}

func NewStore(datadir string) *Store {
	s := &Store{
		dataDir:   datadir,
		eventsOut: make(chan Event),
		eventsIn:  make(chan Event, 256),
		flushCh:   make(chan struct{}, 1),
		quitCh:    make(chan struct{}),
	}
	s.wg.Add(1)
	go s.mainLoop()
	return s
}

// Close closes the store and waits for items to be persisted.
func (s *Store) Close() {
	close(s.quitCh)
	s.wg.Wait()
}

// Events returns the event channel.
// The app reads this channel and applies the events to the UI.
func (s *Store) Events() <-chan Event {
	return s.eventsOut
}

// AddItem tells the store to add a new item.
func (s *Store) AddItem(item Item) {
	s.enqueueInputEvent(&ItemAdded{ID: randomID(), Item: item})
}

// RemoveItem tells the store to delete an item.
func (s *Store) RemoveItem(id ID) {
	s.enqueueInputEvent(&ItemRemoved{ID: id})
}

// UpdateItem tells the store to change an item.
func (s *Store) UpdateItem(id ID, item Item) {
	s.enqueueInputEvent(&ItemChanged{ID: id, Item: item})
}

// Persist tells the store to flush data to disk.
func (s *Store) Persist() {
	select {
	case s.flushCh <- struct{}{}:
	default:
	}
}

// enqueueInputEvent delivers an event from the app to mainLoop.
func (s *Store) enqueueInputEvent(ev Event) {
	select {
	case s.eventsIn <- ev:
	case <-s.quitCh:
	}
}

func (s *Store) mainLoop() {
	defer s.wg.Done()

	// Initial replay.
	err := s.initFile()
	if err != nil {
		s.enqueueOutputEvent(&IOError{Err: err})
	}

	// Handle events.
	for {
		sendEvChan, sendEv := s.queuedOutputEvent()
		select {
		case sendEvChan <- sendEv:
			s.popOutputEvent()

		case ev := <-s.eventsIn:
			if err := s.writeEvent(ev); err != nil {
				s.enqueueOutputEvent(&IOError{Err: err})
			} else {
				s.enqueueOutputEvent(ev)
			}

		case <-s.flushCh:
			if s.dataFile != nil {
				err := s.dataFile.Sync()
				log.Printf("data file flushed (err: %v)", err)
				if err != nil {
					s.enqueueOutputEvent(&IOError{Err: err})
				}
			}

		case <-s.quitCh:
			if s.dataFile != nil {
				err := s.dataFile.Close()
				log.Printf("data file closed (err: %v)", err)
			}
			return
		}
	}
}

func (s *Store) enqueueOutputEvent(ev Event) {
	s.eventQueue.PushBack(ev)
}

func (s *Store) queuedOutputEvent() (chan Event, Event) {
	first := s.eventQueue.Front()
	if first == nil {
		return nil, nil
	}
	return s.eventsOut, first.Value.(Event)
}

func (s *Store) popOutputEvent() {
	s.eventQueue.Remove(s.eventQueue.Front())
}

func (s *Store) writeEvent(ev Event) error {
	if err := s.initFile(); err != nil {
		return err
	}
	return writeEvent(s.writer, ev)
}

func (s *Store) initFile() error {
	if s.dataFile != nil {
		return nil // already exists
	}

	if err := os.MkdirAll(s.dataDir, 0700); err != nil {
		return err
	}
	filename := filepath.Join(s.dataDir, "events.json")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	log.Printf("data file opened: %s", filename)
	s.dataFile = f
	s.reader = json.NewDecoder(f)
	s.writer = json.NewEncoder(f)
	s.replay()
	return nil
}

// replay loads events from the data file and sends them.
func (s *Store) replay() {
	count := 0
	for {
		ev, err := readEvent(s.reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("decode error: %v", err)
			break
		}
		count++
		s.enqueueOutputEvent(ev)
	}
	log.Println("replay done:", count, "items")
}
