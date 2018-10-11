package data

import "time"

// ByAge allows sorting Log Events by age.
type ByAge []*Event

func (a ByAge) Len() int           { return len(a) }
func (a ByAge) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAge) Less(i, j int) bool { return a[i].Ts.Before(a[j].Ts) }

// Log represents a Wires-X log.
type Log struct {
	// Source is where the log was polled from.
	Source string
	// Type defines what log this is (i.e. node log, room log, etd).
	Type string
	// ID is the idenfitier for the node or room this log is for.
	ID string
	// WiresVersion exposes the Wires-X software version of the server.
	WiresVersion string

	// Specific to Node Log
	// ConnectedTo is the node, the repeater is connected to.
	ConnectedTo string

	// Events are all the events listed in the log.
	Events []*Event
}

// Event represents a Wires-X log event / log line.
type Event struct {
	Raw string
	Ts  time.Time
	Msg string
}

// ActiveNodes represents the Active Nodes list provided by Yaesu.
type ActiveNodes struct {
	// LastUpdate is the timestamp of the last update parsed from the polled file.
	LastUpdate time.Time
	// Nodes is a list of all active nodes represented in the file.
	Nodes []*Node
}

type Node struct {
	ID       string
	DTMFID   string
	Callsign string
	AnaDig   string
	Location *Location
	Freq     string
	SQL      string
	Comment  string
}

type Location struct {
	City    string
	State   string
	Country string
	Lat     string
	Lon     string
}
