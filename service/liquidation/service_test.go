package liquidation

import "testing"

func TestParseTimestampNormalizesSeconds(t *testing.T) {
	if got := ParseTimestamp("1700000000"); got != 1700000000000 {
		t.Fatalf("got %d", got)
	}
	if got := ParseTimestamp("1700000000000"); got != 1700000000000 {
		t.Fatalf("got %d", got)
	}
}
