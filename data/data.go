package data

import (
	"encoding/json"
	"time"
)

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

// ActiveRooms represents the Active Rooms list provided by Yaesu.
type ActiveRooms struct {
	// LastUpdate is the timestamp of the last update parsed from the polled file.
	LastUpdate time.Time
	// Rooms is a list of all active rooms represented in the file.
	Rooms []*Room
}

type Room struct {
	ID       string
	Act      string
	DTMFID   string
	Name     string
	Location *Location
	Comment  string
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
	Mode     string
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

type Attachment struct {
	Color    string `json:"color,omitempty"`
	Fallback string `json:"fallback"`

	CallbackID string `json:"callback_id,omitempty"`
	ID         int    `json:"id,omitempty"`

	AuthorID      string `json:"author_id,omitempty"`
	AuthorName    string `json:"author_name,omitempty"`
	AuthorSubname string `json:"author_subname,omitempty"`
	AuthorLink    string `json:"author_link,omitempty"`
	AuthorIcon    string `json:"author_icon,omitempty"`

	Title     string `json:"title,omitempty"`
	TitleLink string `json:"title_link,omitempty"`
	Pretext   string `json:"pretext,omitempty"`
	Text      string `json:"text"`

	ImageURL string `json:"image_url,omitempty"`
	ThumbURL string `json:"thumb_url,omitempty"`

	//Fields     []AttachmentField  `json:"fields,omitempty"`
	//Actions    []AttachmentAction `json:"actions,omitempty"`
	MarkdownIn []string `json:"mrkdwn_in,omitempty"`

	Footer     string `json:"footer,omitempty"`
	FooterIcon string `json:"footer_icon,omitempty"`

	Ts json.Number `json:"ts,omitempty"`
}

type Message struct {
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}
