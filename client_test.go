package stock

import "testing"

func TestHandlesCommand(t *testing.T) {
	tests := []struct {
		registered string
		command    string
		want       bool
	}{
		{"stock", "stock", true},
		{"stock", "stonks", true},
		{"stock", "stockdev", false},
		{"stockdev", "stockdev", true},
		{"stockdev", "stock", false},
		{"stockdev", "stonks", false},
	}
	for _, tt := range tests {
		t.Run(tt.registered+"/"+tt.command, func(t *testing.T) {
			c := &SeabirdClient{command: tt.registered}
			if got := c.handles(tt.command); got != tt.want {
				t.Errorf("handles(%q) with %q registered = %v, want %v", tt.command, tt.registered, got, tt.want)
			}
		})
	}
}
