package model

import (
	"encoding/json"
	"testing"
)

func TestMessageControlsFromExtras(t *testing.T) {
	extras, err := json.Marshal(map[string]any{
		MessageControlExtrasKey: []string{"assign", "resolve", "attach", "assign", "unknown"},
	})
	if err != nil {
		t.Fatal(err)
	}

	controls := MessageControlsFromExtras(extras)
	expected := []string{"assign", "resolve", "attach"}
	if len(controls) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, controls)
	}
	for i := range expected {
		if controls[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, controls)
		}
	}
}

func TestMessageControlEnabledDefaultsOff(t *testing.T) {
	message := &Message{}
	if MessageControlEnabled(message, MessageControlAssign) {
		t.Fatal("assign unexpectedly enabled without message controls")
	}
	if MessageControlEnabled(message, MessageControlResolve) {
		t.Fatal("resolve unexpectedly enabled without message controls")
	}
	if MessageControlEnabled(message, MessageControlAttach) {
		t.Fatal("attach unexpectedly enabled without message controls")
	}
}
