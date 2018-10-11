package processor

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/finfinack/wireslacker/data"
	"github.com/finfinack/wireslacker/resolver"
)

const (
	httpPOST        = "POST"
	httpContentType = "Content-Type"
	httpJSON        = "application/json"
)

var (
	// timePostFormat is the date/time format presented in the Slack post.
	timePostFormat = "2006-01-02 15:04:05"

	// filterMsg is a list of strings against which the log messages are compared
	// and if the log message contains any of them, the log message is ignored.
	// This is primarily to filter boring or noisy stuff.
	filterMsg = []string{
		"Browser connected from", // each poll creates such an entry, ignore them
	}

	inCallRE    = regexp.MustCompile("\\*-\\*-\\* In-Call from No.([0-9]+) \\*-\\*-\\*")
	callStartRE = regexp.MustCompile("\\*-\\*-\\* Call Start No.([0-9]+) \\*-\\*-\\*")
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
	req, err := http.NewRequest(httpPOST, s.webhook, bytes.NewBuffer(body))
	req.Header.Set(httpContentType, httpJSON)
	if s.dry {
		log.Printf("DRY-MODE: Slack message: %v", req)
		return nil
	}
	_, err = s.client.Do(req)
	return err
}

// filter is a simple message filter which decides whether to drop a provided event.
func filter(evt *data.Event, notBefore time.Time) bool {
	// Filter all events which are older than notBefore (avoid posting the same thing twice).
	if !evt.Ts.After(notBefore) {
		return true
	}
	// Filter all events containing any of the filter strings.
	for _, fm := range filterMsg {
		if strings.Contains(evt.Msg, fm) {
			return true
		}
	}
	return false
}

// enrich is a simple function to pass all events through and add more information if available.
func enrich(evt *data.Event) {
	// Attempt to resolve some information about calling nodes.
	if match := inCallRE.FindStringSubmatch(evt.Msg); len(match) > 1 {
		n := resolver.FindNode("", match[1], "")
		if n != nil {
			evt.Msg = fmt.Sprintf("*-*-* In-Call from %s (%s, %s, %s) *-*-*", n.ID, n.Location.City, n.Location.State, n.Location.Country)
			log.Printf("V: Enriched in-call event with location: %v", evt)
		}
	}
	if match := callStartRE.FindStringSubmatch(evt.Msg); len(match) > 1 {
		n := resolver.FindNode("", match[1], "")
		if n != nil {
			evt.Msg = fmt.Sprintf("*-*-* Call Start %s (%s, %s, %s) *-*-*", n.ID, n.Location.City, n.Location.State, n.Location.Country)
			log.Printf("V: Enriched call start event with location: %v", evt)
		}
	}
}

// Run iterates over all logs provided in the log channel and posts new messages using the Slacker provided.
func Run(logChan chan *data.Log, slkr *Slacker, verbose bool) {
	logCount := 0
	notBefore := time.Now()
	for evtLog := range logChan {
		logCount++
		evtCount := 0
		evtFltrCount := 0
		sort.Sort(data.ByAge(evtLog.Events))
		var lastTs time.Time
		for _, evt := range evtLog.Events {
			evtCount++
			if filter(evt, notBefore) {
				evtFltrCount++
				continue
			}
			lastTs = evt.Ts

			log.Printf("New message from %s (%s): %v", evtLog.ID, evtLog.Type, evt)
			enrich(evt)
			slkr.Post(fmt.Sprintf("%s (%s) @ %s: %s", evtLog.ID, evtLog.Type, evt.Ts.Format(timePostFormat), evt.Msg))
		}
		if lastTs.After(notBefore) {
			notBefore = lastTs
		}
		if verbose {
			log.Printf("V: Processed log #%d, total of %d events, filtered %d", logCount, evtCount, evtFltrCount)
		}
	}
}
