package scanner

import (
	"sort"
	"sync"
)

const DefaultSmartLocalV2BatchSize = 5

type SmartLocalV2RotationState struct {
	PoolSize     int      `json:"pool_size"`
	BatchSize    int      `json:"batch_size"`
	Sequence     uint64   `json:"sequence"`
	BatchSymbols []string `json:"batch_symbols"`
	BatchRanks   []int    `json:"batch_ranks"`
}

type SmartLocalV2RoundRobin struct {
	mu           sync.Mutex
	sequence     uint64
	seenSequence uint64
	lastServed   map[string]uint64
	firstSeen    map[string]uint64
}

var (
	smartLocalV2RoundRobinMu sync.Mutex
	smartLocalV2RoundRobins  = map[string]*SmartLocalV2RoundRobin{}
)

func NextSmartLocalV2Batch(candidates []PrefilterCandidate, batchSize int) ([]PrefilterCandidate, SmartLocalV2RotationState) {
	return NextSmartLocalV2BatchFor(string(SmartLocalV2ModeTrade), candidates, batchSize)
}

func PeekSmartLocalV2Batch(candidates []PrefilterCandidate, batchSize int) ([]PrefilterCandidate, SmartLocalV2RotationState) {
	return PeekSmartLocalV2BatchFor(string(SmartLocalV2ModeTrade), candidates, batchSize)
}

func NextSmartLocalV2BatchFor(scope string, candidates []PrefilterCandidate, batchSize int) ([]PrefilterCandidate, SmartLocalV2RotationState) {
	return smartLocalV2RoundRobinFor(scope).Next(candidates, batchSize)
}

func PeekSmartLocalV2BatchFor(scope string, candidates []PrefilterCandidate, batchSize int) ([]PrefilterCandidate, SmartLocalV2RotationState) {
	return smartLocalV2RoundRobinFor(scope).Peek(candidates, batchSize)
}

func smartLocalV2RoundRobinFor(scope string) *SmartLocalV2RoundRobin {
	if scope == "" {
		scope = string(SmartLocalV2ModeTrade)
	}
	smartLocalV2RoundRobinMu.Lock()
	defer smartLocalV2RoundRobinMu.Unlock()
	if rr := smartLocalV2RoundRobins[scope]; rr != nil {
		return rr
	}
	rr := &SmartLocalV2RoundRobin{}
	smartLocalV2RoundRobins[scope] = rr
	return rr
}

func (r *SmartLocalV2RoundRobin) Next(candidates []PrefilterCandidate, batchSize int) ([]PrefilterCandidate, SmartLocalV2RotationState) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ensureState(candidates)
	batch := r.pick(candidates, batchSize)
	if len(batch) > 0 {
		r.sequence++
		for _, candidate := range batch {
			r.lastServed[candidate.Symbol] = r.sequence
		}
	}
	return batch, rotationState(candidates, batch, batchSize, r.sequence)
}

func (r *SmartLocalV2RoundRobin) Peek(candidates []PrefilterCandidate, batchSize int) ([]PrefilterCandidate, SmartLocalV2RotationState) {
	r.mu.Lock()
	defer r.mu.Unlock()

	lastServed := make(map[string]uint64, len(candidates))
	firstSeen := make(map[string]uint64, len(candidates))
	for symbol, value := range r.lastServed {
		lastServed[symbol] = value
	}
	for symbol, value := range r.firstSeen {
		firstSeen[symbol] = value
	}
	seenSequence := r.seenSequence
	for _, candidate := range candidates {
		if _, ok := lastServed[candidate.Symbol]; !ok {
			lastServed[candidate.Symbol] = 0
			seenSequence++
			firstSeen[candidate.Symbol] = seenSequence
		}
	}
	batch := pickSmartLocalV2Batch(candidates, batchSize, lastServed, firstSeen)
	return batch, rotationState(candidates, batch, batchSize, r.sequence)
}

func (r *SmartLocalV2RoundRobin) ensureState(candidates []PrefilterCandidate) {
	if r.lastServed == nil {
		r.lastServed = make(map[string]uint64, len(candidates))
	}
	if r.firstSeen == nil {
		r.firstSeen = make(map[string]uint64, len(candidates))
	}
	active := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		active[candidate.Symbol] = true
		if _, ok := r.lastServed[candidate.Symbol]; !ok {
			r.lastServed[candidate.Symbol] = 0
			r.seenSequence++
			r.firstSeen[candidate.Symbol] = r.seenSequence
		}
	}
	for symbol := range r.lastServed {
		if !active[symbol] {
			delete(r.lastServed, symbol)
			delete(r.firstSeen, symbol)
		}
	}
}

func (r *SmartLocalV2RoundRobin) pick(candidates []PrefilterCandidate, batchSize int) []PrefilterCandidate {
	return pickSmartLocalV2Batch(candidates, batchSize, r.lastServed, r.firstSeen)
}

func pickSmartLocalV2Batch(candidates []PrefilterCandidate, batchSize int, lastServed, firstSeen map[string]uint64) []PrefilterCandidate {
	if batchSize <= 0 {
		batchSize = DefaultSmartLocalV2BatchSize
	}
	if batchSize > len(candidates) {
		batchSize = len(candidates)
	}
	if batchSize == 0 {
		return []PrefilterCandidate{}
	}

	ordered := append([]PrefilterCandidate(nil), candidates...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := lastServed[ordered[i].Symbol]
		right := lastServed[ordered[j].Symbol]
		if left != right {
			return left < right
		}
		leftSeen := firstSeen[ordered[i].Symbol]
		rightSeen := firstSeen[ordered[j].Symbol]
		if leftSeen != rightSeen {
			return leftSeen < rightSeen
		}
		if ordered[i].Rank != ordered[j].Rank {
			return ordered[i].Rank < ordered[j].Rank
		}
		return ordered[i].Symbol < ordered[j].Symbol
	})
	return append([]PrefilterCandidate(nil), ordered[:batchSize]...)
}

func rotationState(pool, batch []PrefilterCandidate, batchSize int, sequence uint64) SmartLocalV2RotationState {
	if batchSize <= 0 {
		batchSize = DefaultSmartLocalV2BatchSize
	}
	symbols := make([]string, 0, len(batch))
	ranks := make([]int, 0, len(batch))
	for _, candidate := range batch {
		symbols = append(symbols, candidate.Symbol)
		ranks = append(ranks, candidate.Rank)
	}
	return SmartLocalV2RotationState{
		PoolSize:     len(pool),
		BatchSize:    batchSize,
		Sequence:     sequence,
		BatchSymbols: symbols,
		BatchRanks:   ranks,
	}
}
