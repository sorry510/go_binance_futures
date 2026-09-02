package contextengine

import (
	"strings"
	"unicode"
)

type TokenEstimator interface {
	Estimate(string) int
}

type HeuristicEstimator struct{}

func (HeuristicEstimator) Estimate(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	ascii, wide := 0, 0
	for _, r := range value {
		if r <= unicode.MaxASCII {
			ascii++
		} else {
			wide++
		}
	}
	// CJK and other non-ASCII text is commonly close to one token per rune;
	// English/code averages near four bytes per token. Slight overestimation is
	// intentional so budget trimming happens before provider rejection.
	return wide + (ascii+3)/4
}
