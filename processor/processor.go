package processor

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/finfinack/wireslack/data"
)

var (
	lastSeen       = time.Now()
	timePostFormat = "2006-01-02 15:04:05"

	filterMsg = []string{
		"Browser connected from",
	}
)

func NewSlacker(webhook string, dry bool) *Slacker {
	return &Slacker{
		webhook,
		&http.Client{},
		dry,
	}
}

type Slacker struct {
	webhook string
	client  *http.Client
	dry     bool
}

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
