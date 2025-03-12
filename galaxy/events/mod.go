package events

type Event interface {

}

type EventType uint16

const (
  EvUnused = iota
  EvNewFood
  EvNewPlayer
  EvPlayerMove
  EvPlayerGrow
  EvDestroyFood
  EvDestroyPlayer
)
