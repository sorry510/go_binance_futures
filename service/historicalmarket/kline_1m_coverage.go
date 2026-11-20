package historicalmarket

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// CompleteKline1mChunkMonths checks all requested months with one metadata-only
// SQL query. A chunk is fast-path complete only when its contiguous range
// covers the requested minute bars. Missing/partial chunks must still go
// through KlineRangeComplete, which also checks ordinary 1m rows.
func (repo *Repository) CompleteKline1mChunkMonths(ctx context.Context, market, symbol string, start, end int64) (map[int64]bool, error) {
	covered := make(map[int64]bool)
	first, last, ok, err := expectedKlineBounds("1m", start, end)
	if err != nil || !ok {
		return covered, err
	}
	db, err := orm.GetDB("default")
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx,
		"SELECT month_start,start_time,end_time,point_count,encoding,compression "+
			"FROM market_klines_1m_chunks WHERE market=? AND symbol=? AND month_start>=? AND month_start<=?",
		market, strings.ToUpper(symbol), kline1mMonthStart(first).UnixMilli(), kline1mMonthStart(last).UnixMilli(),
	)
	if err != nil {
		return nil, fmt.Errorf("check 1m chunk coverage: %w", err)
	}
	defer rows.Close()
	const minuteMs = int64(time.Minute / time.Millisecond)
	for rows.Next() {
		var monthStart, chunkFirst, chunkLast int64
		var pointCount int
		var encoding, compression string
		if err := rows.Scan(&monthStart, &chunkFirst, &chunkLast, &pointCount, &encoding, &compression); err != nil {
			return nil, err
		}
		// Do not trust malformed metadata to skip fetching missing data.
		if encoding != kline1mChunkEncoding || compression != kline1mChunkCompression ||
			pointCount <= 0 || chunkFirst > chunkLast || chunkFirst%minuteMs != 0 ||
			chunkLast%minuteMs != 0 || (chunkLast-chunkFirst)/minuteMs+1 != int64(pointCount) {
			continue
		}
		month := time.UnixMilli(monthStart).UTC()
		_, monthEnd := kline1mMonthBounds(month)
		segmentFirst, segmentLast := first, last
		if segmentFirst < monthStart {
			segmentFirst = monthStart
		}
		monthLast := monthEnd - minuteMs + 1
		if segmentLast > monthLast {
			segmentLast = monthLast
		}
		if chunkFirst <= segmentFirst && chunkLast >= segmentLast {
			covered[monthStart] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return covered, nil
}
