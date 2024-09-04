package store

type Sequence uint64

type EventType byte

const (
	_ EventType = iota
	EventDelete
	EventPut
)

type Event struct {
	Sequence   Sequence
	EventType  EventType
	Key, Value string
}
