package model

import (
	"encoding/json"
	"strings"
)

const (
	MessageControlExtrasKey = "monita::messageControls"
	MessageControlAssign    = "assign"
	MessageControlResolve   = "resolve"
	MessageControlAttach    = "attach"
)

var allowedMessageControls = map[string]struct{}{
	MessageControlAssign:  {},
	MessageControlResolve: {},
	MessageControlAttach:  {},
}

// MessageControlsFromExtras returns the normalized per-message interaction controls
// carried in Monita message extras.
func MessageControlsFromExtras(extras []byte) []string {
	if len(extras) == 0 {
		return nil
	}

	var values map[string]json.RawMessage
	if err := json.Unmarshal(extras, &values); err != nil {
		return nil
	}
	raw, ok := values[MessageControlExtrasKey]
	if !ok {
		return nil
	}

	var requested []string
	if err := json.Unmarshal(raw, &requested); err != nil {
		return nil
	}

	seen := make(map[string]struct{}, len(requested))
	controls := make([]string, 0, len(requested))
	for _, value := range requested {
		control := strings.ToLower(strings.TrimSpace(value))
		if _, allowed := allowedMessageControls[control]; !allowed {
			continue
		}
		if _, exists := seen[control]; exists {
			continue
		}
		seen[control] = struct{}{}
		controls = append(controls, control)
	}
	return controls
}

// MessageControlEnabled reports whether one optional interaction is enabled for
// this individual message.
func MessageControlEnabled(message *Message, control string) bool {
	if message == nil {
		return false
	}
	control = strings.ToLower(strings.TrimSpace(control))
	for _, enabled := range MessageControlsFromExtras(message.Extras) {
		if enabled == control {
			return true
		}
	}
	return false
}
