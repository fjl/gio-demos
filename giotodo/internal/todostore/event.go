package todostore

import (
	"encoding/json"
	"fmt"
)

type ItemAdded struct {
	ID   ID
	Item Item
}

type ItemRemoved struct {
	ID ID
}

type ItemChanged struct {
	ID   ID
	Item Item
}

type IOError struct {
	Err error
}

type Event interface {
	evType() string
}

func (*ItemAdded) evType() string   { return "add" }
func (*ItemRemoved) evType() string { return "remove" }
func (*ItemChanged) evType() string { return "change" }
func (*IOError) evType() string     { return "ioerror" }

type jsonEvent struct {
	Type  string `json:"type"`
	Event Event  `json:"event"`
}

func writeEvent(enc *json.Encoder, ev Event) error {
	jsev := &jsonEvent{Type: ev.evType(), Event: ev}
	return enc.Encode(jsev)
}

func readEvent(dec *json.Decoder) (Event, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('{') {
		return nil, fmt.Errorf("unexpected JSON token %v, expected '{'", tok)
	}

	var (
		evtype = ""
		event  Event
	)
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch keyTok.(string) {
		case "type":
			evtype, err = readEventType(dec)
			if err != nil {
				return nil, err
			}
		case "event":
			if evtype == "" {
				return nil, fmt.Errorf("key \"type\" must precede \"event\"")
			}
			event, err = makeEvent(evtype)
			if err != nil {
				return nil, err
			}
			if err := dec.Decode(event); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown key %q", keyTok)
		}
	}

	// read '}'
	_, err = dec.Token()
	return event, err
}

func readEventType(dec *json.Decoder) (string, error) {
	typeTok, err := dec.Token()
	if err != nil {
		return "", err
	}
	typ, ok := typeTok.(string)
	if !ok {
		return "", fmt.Errorf("expected string for \"type\", got %v", typeTok)
	}
	return typ, nil
}

func makeEvent(evtype string) (Event, error) {
	switch evtype {
	case (&ItemAdded{}).evType():
		return new(ItemAdded), nil
	case (&ItemRemoved{}).evType():
		return new(ItemRemoved), nil
	case (&ItemChanged{}).evType():
		return new(ItemChanged), nil
	default:
		return nil, fmt.Errorf("unknown event type %q", evtype)
	}
}
