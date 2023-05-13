package bird

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ProtocolState struct {
	Name  string
	Proto string
	Table string
	State string
	Since time.Time
	Info  string

	Imported  int
	Exported  int
	Preferred int
}

func trimRepeatingSpace(s string) string {
	space := regexp.MustCompile(`\s+`)
	return space.ReplaceAllString(s, " ")
}

// ParseOne parses a single protocol
func ParseOne(p string) (*ProtocolState, error) {
	// Remove lines that start with BIRD
	birdRegex := regexp.MustCompile(`BIRD.*ready.*`)
	p = birdRegex.ReplaceAllString(p, "")
	tableHeaderRegex := regexp.MustCompile(`Name.*Info`)
	p = tableHeaderRegex.ReplaceAllString(p, "")

	// Remove leading and trailing newlines
	p = strings.Trim(p, "\n")
	header := strings.Split(p, "\n")[0]
	header = trimRepeatingSpace(header)
	headerParts := strings.Split(header, " ")

	if len(headerParts) < 6 {
		return nil, fmt.Errorf("invalid header len %d: %+v (%s)", len(headerParts), headerParts, header)
	}

	// Parse since timestamp
	since, err := time.Parse(time.DateTime, headerParts[4]+" "+headerParts[5])
	if err != nil {
		return nil, err
	}

	// Parse header
	protocolState := &ProtocolState{
		Name:      headerParts[0],
		Proto:     headerParts[1],
		Table:     headerParts[2],
		State:     headerParts[3],
		Since:     since,
		Info:      trimDupSpace(strings.Join(headerParts[6:], " ")),
		Imported:  -1,
		Exported:  -1,
		Preferred: -1,
	}

	// Get line starting with Routes
	routesRegex := regexp.MustCompile(`(.*)Routes:(.*)`)
	routes := routesRegex.FindString(p)
	routes = trimDupSpace(routes)
	routes = trimRepeatingSpace(routes)
	routesParts := strings.Split(routes, " ")

	if len(routesParts) == 7 {
		imported, err := strconv.ParseInt(routesParts[1], 10, 32)
		if err != nil {
			return nil, err
		}
		protocolState.Imported = int(imported)

		exported, err := strconv.ParseInt(routesParts[3], 10, 32)
		if err != nil {
			return nil, err
		}
		protocolState.Exported = int(exported)

		preferred, err := strconv.ParseInt(routesParts[5], 10, 32)
		if err != nil {
			return nil, err
		}
		protocolState.Preferred = int(preferred)
	}

	return protocolState, nil
}

// Parse parses a list of protocols
func Parse(p string) ([]*ProtocolState, error) {
	protocols := strings.Split(p, "\n\n")
	protocolStates := make([]*ProtocolState, len(protocols))
	for i, protocol := range protocols {
		protocolState, err := ParseOne(protocol)
		if err != nil {
			return nil, err
		}
		protocolStates[i] = protocolState
	}
	return protocolStates, nil
}
