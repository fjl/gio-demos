package todostore

import (
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
	eventQueue []Event

	eventsIn chan Event
	quitCh   chan struct{}
	wg       sync.WaitGroup
}

func NewStore(datadir string) *Store {
	s := &Store{
		dataDir:   datadir,
		eventsOut: make(chan Event, 256),
		eventsIn:  make(chan Event),
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

func (s *Store) Events() <-chan Event {
	return s.eventsOut
}

func (s *Store) AddItem(item Item) {
	s.handleEvent(&ItemAdded{ID: randomID(), Item: item})
}

func (s *Store) RemoveItem(id ID) {
	s.handleEvent(&ItemRemoved{ID: id})
}

func (s *Store) UpdateItem(id ID, item Item) {
	s.handleEvent(&ItemChanged{ID: id, Item: item})
}

func (s *Store) handleEvent(ev Event) {
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
		case <-s.quitCh:
			if s.dataFile != nil {
				s.dataFile.Close()
			}
			return
		}
	}
}

func (s *Store) enqueueOutputEvent(ev Event) {
	s.eventQueue = append(s.eventQueue, ev)
}

func (s *Store) queuedOutputEvent() (chan Event, Event) {
	if len(s.eventQueue) == 0 {
		return nil, nil
	}
	return s.eventsOut, s.eventQueue[0]
}

func (s *Store) popOutputEvent() {
	// log.Printf("sent event: %#v", s.eventQueue[0])
	copy(s.eventQueue, s.eventQueue[1:])
	s.eventQueue = s.eventQueue[:len(s.eventQueue)-1]
}

func (s *Store) writeEvent(ev Event) error {
	if err := s.initFile(); err != nil {
		return err
	}
	if err := writeEvent(s.writer, ev); err != nil {
		return err
	}
	return s.dataFile.Sync()
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
	log.Printf("file opened: %s", filename)
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
