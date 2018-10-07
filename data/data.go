package data

import "time"

type ByAge []*Event

func (a ByAge) Len() int           { return len(a) }
func (a ByAge) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAge) Less(i, j int) bool { return a[i].Ts.Before(a[j].Ts) }

type Log struct {
	Source       string
	Type         string
	ID           string
	WiresVersion string
	Events       []*Event
}

type Event struct {
	Raw string
	Ts  time.Time
	Msg string
}
