package bird

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Routes struct {
	Imported  int
	Filtered  int
	Exported  int
	Preferred int
}

type ProtocolState struct {
	Name   string
	Proto  string
	Table  string
	State  string
	Since  time.Time
	Info   string
	Routes *Routes
}

func trimRepeatingSpace(s string) string {
	space := regexp.MustCompile(`\s+`)
	return space.ReplaceAllString(s, " ")
}

func parseRoutes(s string) (*Routes, error) {
	out := &Routes{
		Imported:  -1,
		Filtered:  -1,
		Exported:  -1,
		Preferred: -1,
	}

	routesRegex := regexp.MustCompile(`(.*)Routes:(.*)`)
	routes := routesRegex.FindString(s)
	routes = trimDupSpace(routes)
	routes = trimRepeatingSpace(routes)

	routeTokens := strings.Split(routes, "Routes: ")
	if len(routeTokens) < 2 {
		return out, nil
	}

	routesParts := strings.Split(routeTokens[1], ", ")

	for r := range routesParts {
		parts := strings.Split(routesParts[r], " ")
		num, err := strconv.ParseInt(parts[0], 10, 32)
		if err != nil {
			return nil, err
		}
		switch parts[1] {
		case "imported":
			out.Imported = int(num)
		case "filtered":
			out.Filtered = int(num)
		case "exported":
			out.Exported = int(num)
		case "preferred":
			out.Preferred = int(num)
		}
	}

	return out, nil
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
		Name:  headerParts[0],
		Proto: headerParts[1],
		Table: headerParts[2],
		State: headerParts[3],
		Since: since,
		Info:  trimDupSpace(strings.Join(headerParts[6:], " ")),
	}

	routes, err := parseRoutes(p)
	if err != nil {
		return nil, err
	}
	protocolState.Routes = routes

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
