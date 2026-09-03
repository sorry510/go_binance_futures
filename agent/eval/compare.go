package eval

import "go_binance_futures/agent/replay"

type DimensionDelta struct {
	Name  string  `json:"name"`
	From  float64 `json:"from"`
	To    float64 `json:"to"`
	Delta float64 `json:"delta"`
}

type Comparison struct {
	CaseName           string                     `json:"case_name"`
	ScoreFrom          float64                    `json:"score_from"`
	ScoreTo            float64                    `json:"score_to"`
	ScoreDelta         float64                    `json:"score_delta"`
	VersionDifferences []replay.VersionDifference `json:"version_differences,omitempty"`
	Dimensions         []DimensionDelta           `json:"dimensions,omitempty"`
}

func Compare(from, to Report) Comparison {
	result := Comparison{CaseName: to.CaseName, ScoreFrom: from.Score, ScoreTo: to.Score, ScoreDelta: to.Score - from.Score, VersionDifferences: replay.CompareVersions(from.Identity, to.Identity)}
	left := map[string]float64{}
	for _, item := range from.Dimensions {
		if item.Applicable {
			left[item.Name] = item.Score
		}
	}
	for _, item := range to.Dimensions {
		if item.Applicable {
			previous := left[item.Name]
			if previous != item.Score {
				result.Dimensions = append(result.Dimensions, DimensionDelta{Name: item.Name, From: previous, To: item.Score, Delta: item.Score - previous})
			}
		}
	}
	return result
}
