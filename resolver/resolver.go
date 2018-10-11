package resolver

import (
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hb9tf/wireslacker/data"
)

const (
	activeNodesURL = "https://www.yaesu.com/jp/en/wires-x/id/active_node.php"

	// updateTimeFormat is the date/time format used in the Active Nodes list.
	updateTimeFormat = "02 Jan 2006 15:04:05 MST"
)

var (
	// updateInterval defines how often the nodes list should be refreshed.
	updateInterval = time.Duration(20 * time.Minute)
	// httpTimeout defines how long to wait for a response before giving up.
	httpTimeout = time.Duration(30 * time.Second)

	// updateTimeRE is the regexp used to determine the last update time of the list.
	updateTimeRE = regexp.MustCompile("<p class=.*><span>Update every .*</span> <span>(.*)</span></p>")
	// nodeRE is the regexp used to parse the node information.
	nodeRE = regexp.MustCompile("dataList\\[[0-9]+\\] = {id:\"(.*)\", dtmf_id:\"([0-9]+)\", call_sign:\"(.*)\", ana_dig:\"(.*)\", city:\"(.*)\", state:\"(.*)\", country:\"(.*)\", freq:\"(.*)\", sql:\"(.*)\", lat:\"(.*)\", lon:\"(.*)\", comment:\"(.*)\"};")

	activeNodes   *data.ActiveNodes
	activeNodesMu = &sync.RWMutex{}
)

func read() (string, error) {
	client := &http.Client{
		Timeout: httpTimeout,
	}
	response, err := client.Get(activeNodesURL)
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

func readAndDecodeNodes(verbose bool) (*data.ActiveNodes, error) {
	s, err := read()
	if err != nil {
		return nil, err
	}
	if verbose {
		log.Printf("V: Read %d bytes from %q", len(s), activeNodesURL)
	}
	lines := strings.Split(s, "\n")

	an := &data.ActiveNodes{
		LastUpdate: time.Now(),
		Nodes:      []*data.Node{},
	}
	for _, l := range lines {
		if match := updateTimeRE.FindStringSubmatch(l); len(match) > 1 {
			ts, err := time.Parse(updateTimeFormat, match[1])
			if err != nil {
				continue
			}
			an.LastUpdate = ts
			continue
		}
		if match := nodeRE.FindStringSubmatch(l); len(match) > 1 {
			n := &data.Node{
				ID:       html.UnescapeString(match[1]),
				DTMFID:   html.UnescapeString(match[2]),
				Callsign: html.UnescapeString(match[3]),
				AnaDig:   html.UnescapeString(match[4]),
				Location: &data.Location{
					City:    html.UnescapeString(match[5]),
					State:   html.UnescapeString(match[6]),
					Country: html.UnescapeString(match[7]),
					Lat:     html.UnescapeString(match[10]),
					Lon:     html.UnescapeString(match[11]),
				},
				Freq:    html.UnescapeString(match[8]),
				SQL:     html.UnescapeString(match[9]),
				Comment: html.UnescapeString(match[12]),
			}
			an.Nodes = append(an.Nodes, n)
		}
	}
	return an, nil
}

// Update reads a list of all active nodes from the Yaesu server and updates the cached list locally.
func Update(verbose bool) error {
	an, err := readAndDecodeNodes(verbose)
	if err != nil {
		return err
	}

	activeNodesMu.Lock()
	activeNodes = an
	activeNodesMu.Unlock()

	return nil
}

// AutoUpdate is a blocking function which updates the list of active nodes every updateInterval.
func AutoUpdate(verbose bool) error {
	if err := Update(verbose); err != nil {
		log.Printf("Unable to update nodes (temporarily?): %v", err)
	}
	for _ = range time.Tick(updateInterval) {
		if err := Update(verbose); err != nil {
			log.Printf("Unable to update nodes (temporarily?): %v", err)
			continue // we don't want to abort in this case and retry later
		}
	}
	return nil
}

// FindNode searches through the list of active nodes for the given parameters and returns the
// first node which matches. It returns nil if no node matched.
func FindNode(id, dtmfid, callsign string) *data.Node {
	if activeNodes == nil {
		return nil
	}
	activeNodesMu.RLock()
	activeNodesMu.RUnlock()
	for _, n := range activeNodes.Nodes {
		if id != "" && n.ID == id {
			return n
		}
		if dtmfid != "" && n.DTMFID == dtmfid {
			return n
		}
		if callsign != "" && n.Callsign == callsign {
			return n
		}
	}
	return nil
}
