package scanner

import (
	"fmt"
	"reflect"
	"testing"
)

func TestSmartLocalV2RoundRobinCoversStableTop60(t *testing.T) {
	var rr SmartLocalV2RoundRobin
	pool := make([]PrefilterCandidate, 0, 60)
	for i := 1; i <= 60; i++ {
		pool = append(pool, PrefilterCandidate{Rank: i, Symbol: fmt.Sprintf("S%02dUSDT", i)})
	}

	for round := 0; round < 12; round++ {
		batch, state := rr.Next(pool, 5)
		if len(batch) != 5 {
			t.Fatalf("round %d batch size=%d", round+1, len(batch))
		}
		for i, candidate := range batch {
			wantRank := round*5 + i + 1
			if candidate.Rank != wantRank {
				t.Fatalf("round %d item %d rank=%d want=%d batch=%+v", round+1, i, candidate.Rank, wantRank, batch)
			}
		}
		if state.Sequence != uint64(round+1) {
			t.Fatalf("round %d sequence=%d", round+1, state.Sequence)
		}
	}

	batch, _ := rr.Next(pool, 5)
	got := []int{}
	for _, candidate := range batch {
		got = append(got, candidate.Rank)
	}
	if !reflect.DeepEqual(got, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("thirteenth batch=%v want first five again", got)
	}
}

func TestSmartLocalV2RoundRobinNewEntrantDoesNotStarveExistingUnserved(t *testing.T) {
	var rr SmartLocalV2RoundRobin
	initial := make([]PrefilterCandidate, 0, 10)
	for i := 1; i <= 10; i++ {
		initial = append(initial, PrefilterCandidate{Rank: i, Symbol: fmt.Sprintf("S%02dUSDT", i)})
	}
	first, _ := rr.Next(initial, 5)
	if first[0].Rank != 1 || first[4].Rank != 5 {
		t.Fatalf("first batch=%+v", first)
	}

	// A new top-ranked entrant appears before the next round. Existing unseen
	// symbols S06-S10 must still be served before the newcomer.
	nextPool := []PrefilterCandidate{{Rank: 1, Symbol: "NEWUSDT"}}
	for i := 6; i <= 10; i++ {
		nextPool = append(nextPool, PrefilterCandidate{Rank: i - 4, Symbol: fmt.Sprintf("S%02dUSDT", i)})
	}
	for i := 1; i <= 5; i++ {
		nextPool = append(nextPool, PrefilterCandidate{Rank: i + 6, Symbol: fmt.Sprintf("S%02dUSDT", i)})
	}

	second, _ := rr.Next(nextPool, 5)
	got := []string{}
	for _, candidate := range second {
		got = append(got, candidate.Symbol)
	}
	want := []string{"S06USDT", "S07USDT", "S08USDT", "S09USDT", "S10USDT"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("second batch=%v want=%v", got, want)
	}
}

func TestSmartLocalV2RoundRobinPeekDoesNotAdvance(t *testing.T) {
	var rr SmartLocalV2RoundRobin
	pool := []PrefilterCandidate{
		{Rank: 1, Symbol: "AAAUSDT"},
		{Rank: 2, Symbol: "BBBUSDT"},
		{Rank: 3, Symbol: "CCCUSDT"},
		{Rank: 4, Symbol: "DDDUSDT"},
		{Rank: 5, Symbol: "EEEUSDT"},
		{Rank: 6, Symbol: "FFFUSDT"},
	}

	peek1, state1 := rr.Peek(pool, 5)
	peek2, state2 := rr.Peek(pool, 5)
	if !reflect.DeepEqual(peek1, peek2) || state1.Sequence != 0 || state2.Sequence != 0 {
		t.Fatalf("peek advanced state: peek1=%+v peek2=%+v state1=%+v state2=%+v", peek1, peek2, state1, state2)
	}

	actual, state := rr.Next(pool, 5)
	if !reflect.DeepEqual(actual, peek1) || state.Sequence != 1 {
		t.Fatalf("actual=%+v peek=%+v state=%+v", actual, peek1, state)
	}

	next, _ := rr.Peek(pool, 5)
	if len(next) != 5 || next[0].Symbol != "FFFUSDT" {
		t.Fatalf("next peek should start with the only unserved symbol: %+v", next)
	}
}

func TestSmartLocalV2RoundRobinScopesAreIndependent(t *testing.T) {
	pool := []PrefilterCandidate{
		{Rank: 1, Symbol: "AUSDT"},
		{Rank: 2, Symbol: "BUSDT"},
		{Rank: 3, Symbol: "CUSDT"},
		{Rank: 4, Symbol: "DUSDT"},
		{Rank: 5, Symbol: "EUSDT"},
		{Rank: 6, Symbol: "FUSDT"},
	}
	tradeScope := "trade-test-scope"
	testScope := "paper-test-scope"

	trade1, stateTrade1 := NextSmartLocalV2BatchFor(tradeScope, pool, 5)
	test1, stateTest1 := NextSmartLocalV2BatchFor(testScope, pool, 5)
	if !reflect.DeepEqual(trade1, test1) {
		t.Fatalf("first batches differ: trade=%+v test=%+v", trade1, test1)
	}
	if stateTrade1.Sequence != 1 || stateTest1.Sequence != 1 {
		t.Fatalf("independent scopes must each start at sequence 1: trade=%+v test=%+v", stateTrade1, stateTest1)
	}

	trade2, _ := NextSmartLocalV2BatchFor(tradeScope, pool, 5)
	testPeek, statePeek := PeekSmartLocalV2BatchFor(testScope, pool, 5)
	if len(trade2) == 0 || trade2[0].Symbol != "FUSDT" {
		t.Fatalf("trade scope did not advance independently: %+v", trade2)
	}
	if len(testPeek) == 0 || testPeek[0].Symbol != "FUSDT" || statePeek.Sequence != 1 {
		t.Fatalf("test scope should still be at its own next batch: batch=%+v state=%+v", testPeek, statePeek)
	}
}
