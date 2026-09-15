package logcleanup

import (
	"context"
	"fmt"

	"github.com/beego/beego/v2/client/orm"
)

type Target struct {
	Name       string `json:"name"`
	Table      string `json:"table"`
	TimeColumn string `json:"time_column"`
}

type Result struct {
	Name    string `json:"name"`
	Table   string `json:"table"`
	Deleted int64  `json:"deleted"`
}

var targets = []Target{
	{Name: "agent observations", Table: "agent_observations", TimeColumn: "created_at"},
	{Name: "agent change events", Table: "agent_change_events", TimeColumn: "created_at"},
	{Name: "agent task events", Table: "agent_task_events", TimeColumn: "event_time"},
	{Name: "alert pipeline traces", Table: "agent_alert_pipeline_traces", TimeColumn: "created_at"},
	{Name: "notifications", Table: "notifications", TimeColumn: "create_time"},
}

func Targets() []Target {
	result := make([]Target, len(targets))
	copy(result, targets)
	return result
}

func Cleanup(ctx context.Context, cutoff int64) ([]Result, error) {
	if cutoff <= 0 {
		return nil, fmt.Errorf("cleanup cutoff must be positive")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	o := orm.NewOrm()
	tx, err := o.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin log cleanup: %w", err)
	}
	results := make([]Result, 0, len(targets))
	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		res, err := tx.Raw("DELETE FROM "+target.Table+" WHERE "+target.TimeColumn+" < ?", cutoff).Exec()
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("cleanup %s: %w", target.Table, err)
		}
		deleted, err := res.RowsAffected()
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("cleanup %s rows affected: %w", target.Table, err)
		}
		results = append(results, Result{Name: target.Name, Table: target.Table, Deleted: deleted})
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit log cleanup: %w", err)
	}
	return results, nil
}
