package ui

import "testing"

func TestFormatUptime(t *testing.T) {
	cases := []struct {
		seconds int64
		want    string
	}{
		{90, "1m"},
		{3661, "1h 1m"},
		{90000, "1d 1h"},
	}

	for _, c := range cases {
		got := formatUptime(c.seconds)
		if got != c.want {
			t.Errorf("formatUptime(%d) = %q, want %q", c.seconds, got, c.want)
		}
	}
}
