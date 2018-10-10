package reader

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/finfinack/wireslacker/data"
)

const (
	// timeFormat is the date/time format used in Wires-X logs.
	timeFormat = "2006/01/02 15:04:05"
)

var (
	// httpLogTypeRE is the regexp used to capture the name of an HTTP/S based log.
	httpLogTypeRE = regexp.MustCompile("<title>(.*)</title>")
	// httpVersionRE is the regexp used to determine the Wires-X version of an HTTP/S based log.
	httpVersionRE = regexp.MustCompile("<body><a href=\".*\">(WIRES-X .*)")
	// httpNodeRE is the regexp used to find the node info of an HTTP/S based log.
	httpNodeRE = regexp.MustCompile("NODE: <b>(.*) , (.*\\([0-9]+\\)) </b>")
	// httpRoomRE is the regexp used to find the room info of an HTTP/S based log.
	httpRoomRE = regexp.MustCompile("ROOM: <b>(.*) , (.*\\([0-9]+\\)) </b>")
	// logMsgRE is the regexp used to match a log event (timestamp plus message).
	logMsgRE = regexp.MustCompile("([0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2})[[:space:]]+(.*)")
)

// Log is an interface to provide access to Wires-X logs.
type Log interface {
	// Read polls the log and parses it into data.Log format.
	Read() (*data.Log, error)
}

// New creates a new Log reader matching the provided target.
func New(target string) (Log, error) {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return &HTTP{
			target,
		}, nil
	}
	return nil, fmt.Errorf("no reader for %q not implemented, provide an alternative target", target)
}

// HTTP implements the Log interface and reads the log from an HTTP/S target.
type HTTP struct {
	target string
}

// read grabs the raw log from the target and returns it as a string.
func (r *HTTP) read() (string, error) {
	response, err := http.Get(r.target)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Read polls the log and parses it into data.Log format.
func (r *HTTP) Read() (*data.Log, error) {
	s, err := r.read()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(s, "<br>")

	log := &data.Log{
		Source: r.target,
		Events: []*data.Event{},
	}
	for _, l := range lines {
		// General info
		if match := httpLogTypeRE.FindStringSubmatch(l); len(match) > 1 {
			log.Type = match[1]
			continue
		}
		if match := httpVersionRE.FindStringSubmatch(l); len(match) > 1 {
			log.WiresVersion = match[1]
			continue
		}

		// Info depending on the log type to determine ID
		if match := httpNodeRE.FindStringSubmatch(l); len(match) > 1 {
			log.ID = fmt.Sprintf("%s, %s", match[1], match[2])
			continue
		}
		if match := httpRoomRE.FindStringSubmatch(l); len(match) > 1 {
			log.ID = fmt.Sprintf("%s, %s", match[1], match[2])
			continue
		}

		// Actual message parsing
		if match := logMsgRE.FindStringSubmatch(l); len(match) > 1 {
			ts, err := time.Parse(timeFormat, match[1])
			if err != nil {
				continue
			}
			log.Events = append(log.Events, &data.Event{
				Raw: l,
				Ts:  ts,
				Msg: match[2],
			})
		}
	}
	return log, nil
}
