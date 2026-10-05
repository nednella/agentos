package project

import (
	"slices"
	"strconv"
	"strings"
)

var defaultLanes = map[string]string{
	"ready": "ready", "needs-plan": "plan", "needs-human": "you", "idea": "idea",
}

var defaultCommands = map[string]string{
	"ready": "/work {n}", "plan": "/investigate {n}", "inbox": "", "idea": "",
}

// laneOrder ranks lanes when an issue's labels map to several.
var laneOrder = []string{"ready", "plan", "you", "idea"}

// Lane places an issue by its labels: the first lane in laneOrder that a label
// maps to, "inbox" when there are no labels at all, else "idea".
func (p Project) Lane(labels []string) string {
	lanes := p.Lanes
	if len(lanes) == 0 {
		lanes = defaultLanes
	}
	var hit []string
	for _, l := range labels {
		hit = append(hit, lanes[l])
	}
	for _, lane := range laneOrder {
		if slices.Contains(hit, lane) {
			return lane
		}
	}
	if len(labels) == 0 {
		return "inbox"
	}
	return "idea"
}

func (p Project) command(key string) string {
	if c, ok := p.Commands[key]; ok {
		return c
	}
	return defaultCommands[key]
}

// IssueCommand is what a session for an issue in the lane types into the agent, or "".
func (p Project) IssueCommand(lane string, number int) string {
	if lane != "ready" && lane != "plan" && lane != "inbox" && lane != "idea" {
		return ""
	}
	return strings.ReplaceAll(p.command(lane), "{n}", strconv.Itoa(number))
}
