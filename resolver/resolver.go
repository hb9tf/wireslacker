package resolver

import (
	"fmt"
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
	activeRoomsURL = "https://www.yaesu.com/jp/en/wires-x/id/active_room.php"

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
	// roomRE is the regexp used to parse the room information.
	roomRE = regexp.MustCompile("dataList\\[[0-9]+\\] = {id:\"(.*)\", dtmp:\"([0-9]+)\", act:\"(.*)\", room_name:\"(.*)\", city:\"(.*)\", state:\"(.*)\", country:\"(.*)\", comment:\"(.*)\"};")

	latRE = regexp.MustCompile("([NS]):([0-9]+) ([0-9]+)' ([0-9]+)")
	lonRE = regexp.MustCompile("([EW]):([0-9]+) ([0-9]+)' ([0-9]+)")

	activeNodes   *data.ActiveNodes
	activeNodesMu = &sync.RWMutex{}
	activeRooms   *data.ActiveRooms
	activeRoomsMu = &sync.RWMutex{}
)

func convertLatLon(lat, lon string) (string, string, error) {
	matchLat := latRE.FindStringSubmatch(lat)
	if len(matchLat) < 2 {
		return "", "", fmt.Errorf("unable to determine latitude: %s", lat)
	}
	matchLon := lonRE.FindStringSubmatch(lon)
	if len(matchLon) < 2 {
		return "", "", fmt.Errorf("unable to determine longitude: %s", lon)
	}
	return fmt.Sprintf("%s %s %s %s", matchLat[2], matchLat[3], matchLat[4], matchLat[1]), fmt.Sprintf("%s %s %s %s", matchLon[2], matchLon[3], matchLon[4], matchLon[1]), nil
}

func read(target string) (string, error) {
	client := &http.Client{
		Timeout: httpTimeout,
	}
	response, err := client.Get(target)
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

func readAndDecodeRooms(verbose bool) (*data.ActiveRooms, error) {
	s, err := read(activeRoomsURL)
	if err != nil {
		return nil, err
	}
	if verbose {
		log.Printf("V: Read %d bytes from %q", len(s), activeRoomsURL)
	}
	lines := strings.Split(s, "\n")

	ar := &data.ActiveRooms{
		LastUpdate: time.Now(),
		Rooms:      []*data.Room{},
	}
	for _, l := range lines {
		if match := updateTimeRE.FindStringSubmatch(l); len(match) > 1 {
			ts, err := time.Parse(updateTimeFormat, match[1])
			if err != nil {
				continue
			}
			ar.LastUpdate = ts
			continue
		}
		if match := roomRE.FindStringSubmatch(l); len(match) > 1 {
			r := &data.Room{
				ID:     html.UnescapeString(match[1]),
				DTMFID: html.UnescapeString(match[2]),
				Act:    html.UnescapeString(match[3]),
				Name:   html.UnescapeString(match[4]),
				Location: &data.Location{
					City:    html.UnescapeString(match[5]),
					State:   html.UnescapeString(match[6]),
					Country: html.UnescapeString(match[7]),
				},
				Comment: html.UnescapeString(match[8]),
			}
			ar.Rooms = append(ar.Rooms, r)
		}
	}
	return ar, nil
}

func readAndDecodeNodes(verbose bool) (*data.ActiveNodes, error) {
	s, err := read(activeNodesURL)
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
			lat, lon, err := convertLatLon(html.UnescapeString(match[10]), html.UnescapeString(match[11]))
			if err != nil {
				lat = ""
				lon = ""
			}
			n := &data.Node{
				ID:       html.UnescapeString(match[1]),
				DTMFID:   html.UnescapeString(match[2]),
				Callsign: html.UnescapeString(match[3]),
				Mode:     html.UnescapeString(match[4]),
				Location: &data.Location{
					City:    html.UnescapeString(match[5]),
					State:   html.UnescapeString(match[6]),
					Country: html.UnescapeString(match[7]),
					Lat:     lat,
					Lon:     lon,
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

// Update reads a list of all active nodes and rooms from the Yaesu server and updates the cached list locally.
func Update(verbose bool) error {
	an, err := readAndDecodeNodes(verbose)
	if err != nil {
		return err
	}

	activeNodesMu.Lock()
	activeNodes = an
	activeNodesMu.Unlock()

	ar, err := readAndDecodeRooms(verbose)
	if err != nil {
		return err
	}

	activeRoomsMu.Lock()
	activeRooms = ar
	activeRoomsMu.Unlock()

	return nil
}

// AutoUpdate is a blocking function which updates the list of active nodes and rooms every updateInterval.
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

// FindRoom searches through the list of active rooms for the given parameters and returns the
// first room which matches. It returns nil if no room matched.
func FindRoom(id, dtmfid, name string) *data.Room {
	if activeRooms == nil {
		return nil
	}
	activeRoomsMu.RLock()
	activeRoomsMu.RUnlock()
	for _, r := range activeRooms.Rooms {
		if id != "" && r.ID == id {
			return r
		}
		if dtmfid != "" && r.DTMFID == dtmfid {
			return r
		}
		if name != "" && r.Name == name {
			return r
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
