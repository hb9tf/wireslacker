package processor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hb9tf/wireslacker/data"
	"github.com/hb9tf/wireslacker/resolver"
)

const (
	httpPOST        = "POST"
	httpContentType = "Content-Type"
	httpJSON        = "application/json"

	slackColorGood = "good"
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

	// Mixed node and room RE
	callStartRE   = regexp.MustCompile("Call Start No.([0-9]+)")
	connectedToRE = regexp.MustCompile("Connected to (.+)\\(([0-9]+)\\)\\.")
	// Node only RE
	nodeInCallRE = regexp.MustCompile("In-Call from No.([0-9]+)")
	// Room only RE
	nodeInRE  = regexp.MustCompile("(.+)\\(([0-9]+)\\) IN\\.")
	nodeOutRE = regexp.MustCompile("(.+)\\(([0-9]+)\\) OUT\\.")
)

// NewSlacker creates a new Slacker for the provided webhook.
func NewSlacker(webhook string, dry bool, verbose bool) *Slacker {
	return &Slacker{
		webhook,
		&http.Client{},
		dry,
		verbose,
	}
}

// Slacker is a super simple Slack bot which allows to post messages using a webhook.
type Slacker struct {
	webhook string
	client  *http.Client
	dry     bool
	verbose bool
}

// Post sends the provided message to the webhook, posting it in the channel.
func (s *Slacker) Post(msg *data.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(httpPOST, s.webhook, bytes.NewBuffer(data))
	req.Header.Set(httpContentType, httpJSON)
	if s.verbose {
		log.Printf("V: Posting Slack message: %v", req)
	}
	if s.dry {
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
func enrich(evtLog *data.Log, evt *data.Event, msg *data.Message, verbose bool) *data.Message {
	// Attempt to resolve some information about calling nodes.
	var n *data.Node
	if match := nodeInCallRE.FindStringSubmatch(evt.Msg); len(match) > 1 {
		n = resolver.FindNode("", match[1], "")
	} else if match := callStartRE.FindStringSubmatch(evt.Msg); len(match) > 1 {
		n = resolver.FindNode(match[1], match[2], "")
	} else if match := connectedToRE.FindStringSubmatch(evt.Msg); len(match) > 1 {
		n = resolver.FindNode("", match[1], "")
	}
	if n != nil {
		loc := "n/a"
		if n.Location != nil {
			loc = fmt.Sprintf("%s, %s, %s", n.Location.City, n.Location.State, n.Location.Country)
			if n.Location.Lat != float64(0) && n.Location.Lon != float64(0) {
				loc = fmt.Sprintf("<https://www.google.com/maps/@%f,%f%s>", n.Location.Lat, n.Location.Lon, loc)
			}
		}
		text := []string{
			fmt.Sprintf("%s (%s):", n.ID, n.Mode),
			fmt.Sprintf("Location: %s", loc),
		}
		if n.Freq != "" {
			text = append(text, fmt.Sprintf("Frequency: %s (%s)", n.Freq, n.SQL))
		}
		if n.Comment != "" {
			text = append(text, fmt.Sprintf("Comment: %s", n.Comment))
		}
		msg.Attachments[0].Text = strings.Join(text, "\n")
		msg.Attachments[0].Color = slackColorGood
		if verbose {
			log.Printf("V: Enriched message with node information: %v", msg)
		}
	}

	// Attempt to resolve some information about rooms.
	var r *data.Room
	if match := callStartRE.FindStringSubmatch(evt.Msg); len(match) > 1 {
		r = resolver.FindRoom(match[1], match[2], "")
	} else if match := connectedToRE.FindStringSubmatch(evt.Msg); len(match) > 1 {
		r = resolver.FindRoom("", match[1], "")
	} else if match := nodeInRE.FindStringSubmatch(evt.Msg); len(match) > 1 {
		r = resolver.FindRoom(match[1], match[2], "")
	} else if match := nodeOutRE.FindStringSubmatch(evt.Msg); len(match) > 1 {
		r = resolver.FindRoom(match[1], match[2], "")
	}
	if r != nil {
		loc := "n/a"
		if r.Location != nil {
			loc = fmt.Sprintf("%s, %s, %s", r.Location.City, r.Location.State, r.Location.Country)
		}
		text := []string{
			fmt.Sprintf("%s: %s", r.ID, r.Name),
			fmt.Sprintf("Location: %s", loc),
		}
		if r.Comment != "" {
			text = append(text, fmt.Sprintf("Comment: %s", r.Comment))
		}
		msg.Attachments[0].Text = strings.Join(text, "\n")
		msg.Attachments[0].Color = slackColorGood
		if verbose {
			log.Printf("V: Enriched message with room information: %v", msg)
		}
	}

	return msg
}

func getSlackMsg(evtLog *data.Log, evt *data.Event, verbose bool) *data.Message {
	msg := &data.Message{
		Attachments: []data.Attachment{
			{
				Pretext: fmt.Sprintf(
					"%s: %s",
					evtLog.ID,
					evt.Msg),
				Ts: json.Number(evt.Ts.Unix()),
			},
		},
	}
	return enrich(evtLog, evt, msg, verbose)
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
			slkr.Post(getSlackMsg(evtLog, evt, verbose))
		}
		if lastTs.After(notBefore) {
			notBefore = lastTs
		}
		if verbose {
			log.Printf("V: Processed log #%d, total of %d events, filtered %d", logCount, evtCount, evtFltrCount)
		}
	}
}
