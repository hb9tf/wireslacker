package reader

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/finfinack/wireslack/data"
)

const (
	timeFormat = "2006/01/02 15:04:05"
)

var (
	httpLogTypeRE = regexp.MustCompile("<title>(.*)</title>")
	httpInfoRE    = regexp.MustCompile("<body><a href=\".*\">(WIRES-X .*)")
	httpNodeRE    = regexp.MustCompile("NODE: <b>(.*) , (.*\\([0-9]+\\)) </b>")
	httpRoomRE    = regexp.MustCompile("ROOM: <b>(.*) , (.*\\([0-9]+\\)) </b>")
	logMsgRE      = regexp.MustCompile("([0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2})[[:space:]]+(.*)")
)

type Log interface {
	Read() (*data.Log, error)
}

type HTTP struct {
	target string
}

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
		if match := httpInfoRE.FindStringSubmatch(l); len(match) > 1 {
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

func New(target string) (Log, error) {
	if strings.HasPrefix(target, "http") {
		return &HTTP{
			target,
		}, nil
	}
	return nil, fmt.Errorf("no reader for %q not implemented, provide an alternative target", target)
}
