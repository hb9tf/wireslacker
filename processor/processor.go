package processor

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/finfinack/wireslacker/data"
)

var (
	// lastSeen is used to keep the timestamp of the last event that was processed.
	// Only if a newer event is seen, it is processed/posted.
	lastSeen = time.Now()
	// timePostFormat is the date/time format presented in the Slack post.
	timePostFormat = "2006-01-02 15:04:05"

	// filterMsg is a list of strings against which the log messages are compared
	// and if the log message contains any of them, the log message is ignored.
	// This is primarily to filter boring or noisy stuff.
	filterMsg = []string{
		"Browser connected from", // each poll creates such an entry, ignore them
	}
)

// NewSlacker creates a new Slacker for the provided webhook.
func NewSlacker(webhook string, dry bool) *Slacker {
	return &Slacker{
		webhook,
		&http.Client{},
		dry,
	}
}

// Slacker is a super simple Slack bot which allows to post messages using a webhook.
type Slacker struct {
	webhook string
	client  *http.Client
	dry     bool
}

// Post sends the provided message to the webhook, posting it in the channel.
func (s *Slacker) Post(msg string) error {
	body := []byte(fmt.Sprintf(`{"text":"%s"}`, msg))
	req, err := http.NewRequest("POST", s.webhook, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if s.dry {
		log.Printf("DRY-MODE: Slack message: %v\n", req)
		return nil
	}
	_, err = s.client.Do(req)
	return err
}

// filter is a simple message filter which decides whether to drop a provided event.
// All events which are older than the lastSeen are ignored as well as messages
// which contains any of the filter strings.
func filter(evt *data.Event) bool {
	if !evt.Ts.After(lastSeen) {
		return true
	}
	for _, fm := range filterMsg {
		if strings.Contains(evt.Msg, fm) {
			return true
		}
	}
	return false
}

// Run iterates over all logs provided in the log channel and posts new messages using the Slacker provided.
func Run(logChan chan *data.Log, slkr *Slacker) {
	for log := range logChan {
		sort.Sort(data.ByAge(log.Events))
		for _, evt := range log.Events {
			if filter(evt) {
				continue
			}
			lastSeen = evt.Ts
			fmt.Printf("New message from %s (%s): %v\n", log.ID, log.Type, evt)
			slkr.Post(fmt.Sprintf("%s (%s) @ %s: %s", log.ID, log.Type, evt.Ts.Format(timePostFormat), evt.Msg))
		}
	}
}
