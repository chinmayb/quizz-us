package ws

import (
	"encoding/json"
	"testing"

	pb "github.com/chinmayb/quizz-us/gen/go/api"
)

func TestParseWebSocketMessage_Action(t *testing.T) {
	payload := []byte(`{"id":"player-1","code":"ABC123","action":"join"}`)

	msg, err := parseWebSocketMessage(payload)
	if err != nil {
		t.Fatalf("parseWebSocketMessage returned error: %v", err)
	}

	if msg.GetId() != "player-1" {
		t.Fatalf("expected id 'player-1', got %q", msg.GetId())
	}

	if msg.GetCode() != "ABC123" {
		t.Fatalf("expected code 'ABC123', got %q", msg.GetCode())
	}

	action := msg.GetAction()
	if action != pb.GamePlayAction_JOIN {
		t.Fatalf("expected action JOIN, got %v", action)
	}
}

func TestParseWebSocketMessage_Command(t *testing.T) {
	payload := []byte(`{"id":"player-1","code":"ABC123","command":{"id":"q1","player_answer":"42"}}`)

	msg, err := parseWebSocketMessage(payload)
	if err != nil {
		t.Fatalf("parseWebSocketMessage returned error: %v", err)
	}

	command := msg.GetCommand()
	if command.GetId() != "q1" {
		t.Fatalf("expected command id 'q1', got %q", command.GetId())
	}
	if command.GetPlayerAnswer() != "42" {
		t.Fatalf("expected player answer '42', got %q", command.GetPlayerAnswer())
	}
}

func TestParseWebSocketMessage_Invalid(t *testing.T) {
	payload := []byte(`{"code":"ABC123","action":"JOIN"}`)

	if _, err := parseWebSocketMessage(payload); err == nil {
		t.Fatalf("expected error when id is missing")
	}
}

func TestBuildWebSocketPayload_Action(t *testing.T) {
	msg := &pb.GamePlay{
		Id:   "player-1",
		Code: "ABC123",
		Cmd:  &pb.GamePlay_Action{Action: pb.GamePlayAction_HEARTBEAT},
	}

	payload, err := buildWebSocketPayload(msg)
	if err != nil {
		t.Fatalf("buildWebSocketPayload returned error: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	var action string
	if err := json.Unmarshal(decoded["action"], &action); err != nil {
		t.Fatalf("failed to unmarshal action: %v", err)
	}

	if action != "HEARTBEAT" {
		t.Fatalf("expected action HEARTBEAT, got %s", action)
	}
}

func TestBuildWebSocketPayload_Command(t *testing.T) {
	msg := &pb.GamePlay{
		Id:   "player-1",
		Code: "ABC123",
		Cmd: &pb.GamePlay_Command{Command: &pb.GamePlayCommand{
			Id:           "q1",
			PlayerAnswer: "42",
		}},
	}

	payload, err := buildWebSocketPayload(msg)
	if err != nil {
		t.Fatalf("buildWebSocketPayload returned error: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if len(decoded["command"]) == 0 {
		t.Fatalf("expected command field to be present")
	}

	var command map[string]interface{}
	if err := json.Unmarshal(decoded["command"], &command); err != nil {
		t.Fatalf("failed to unmarshal command payload: %v", err)
	}

	if command["id"] != "q1" {
		t.Fatalf("expected command id 'q1', got %v", command["id"])
	}

	if command["player_answer"] != "42" {
		t.Fatalf("expected player_answer '42', got %v", command["player_answer"])
	}
}
