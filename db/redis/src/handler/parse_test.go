package handler

import "testing"

func TestParseMessageKeepsJSONArgumentsWhole(t *testing.T) {
	// The old parser split on whitespace and refused anything longer than
	// two fields, so an ADDUSER carrying JSON was rejected outright.
	payload := `{"id":"abc","country":"KR","path":"/blog"}`
	cmd, arg, err := parseMessage("ADDUSER " + payload + "\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "ADDUSER" {
		t.Errorf("command = %q", cmd)
	}
	if arg != payload {
		t.Errorf("arg = %q, want %q", arg, payload)
	}
}

func TestParseMessageHandlesABareVerb(t *testing.T) {
	for _, line := range []string{"REAL-TIME\n", "REAL-TIME \n", "  REAL-TIME  "} {
		cmd, arg, err := parseMessage(line)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", line, err)
		}
		if cmd != "REAL-TIME" || arg != "" {
			t.Errorf("%q -> (%q, %q)", line, cmd, arg)
		}
	}
}

func TestParseMessageStillReadsTheOldTwoFieldForm(t *testing.T) {
	cmd, arg, err := parseMessage("ADDUSER 1.2.3.4:5678\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "ADDUSER" || arg != "1.2.3.4:5678" {
		t.Errorf("got (%q, %q)", cmd, arg)
	}
}

func TestParseMessageRejectsAnEmptyLine(t *testing.T) {
	if _, _, err := parseMessage("\n"); err == nil {
		t.Error("expected an error for an empty line")
	}
}
