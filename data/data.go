package data

import "time"

// ByAge allows sorting Log Events by age.
type ByAge []*Event

func (a ByAge) Len() int           { return len(a) }
func (a ByAge) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAge) Less(i, j int) bool { return a[i].Ts.Before(a[j].Ts) }

// Log represents a Wires-X log.
type Log struct {
	Source       string
	Type         string
	ID           string
	WiresVersion string
	Events       []*Event
}

// Event represents a Wires-X log event / log line.
type Event struct {
	Raw string
	Ts  time.Time
	Msg string
}
