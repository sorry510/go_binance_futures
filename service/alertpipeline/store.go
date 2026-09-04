package alertpipeline

import (
	"context"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
	signalservice "go_binance_futures/service/signal"
)

type TraceRepository interface {
	Save(context.Context, Trace) error
}

type ORMTraceStore struct{ Alias string }

func (store ORMTraceStore) ormer() orm.Ormer {
	if strings.TrimSpace(store.Alias) != "" {
		return orm.NewOrmUsingDB(store.Alias)
	}
	return orm.NewOrm()
}

func (store ORMTraceStore) Save(ctx context.Context, trace Trace) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(trace.SignalID) == "" {
		return nil
	}
	o := store.ormer()
	var row models.AgentAlertPipelineTrace
	err := o.QueryTable(new(models.AgentAlertPipelineTrace)).Filter("signal_id", trace.SignalID).One(&row)
	if err != nil && err != orm.ErrNoRows {
		return err
	}
	createdAt := trace.CreatedAt
	if createdAt <= 0 {
		createdAt = time.Now().UnixMilli()
	}
	if row.ID > 0 && row.CreatedAt > 0 {
		createdAt = row.CreatedAt
	}
	row.EventID = trace.EventID
	row.SignalID = trace.SignalID
	row.TaskID = trace.TaskID
	row.NotificationID = trace.NotificationID
	row.Symbol = trace.Symbol
	row.Type = string(trace.Type)
	row.Severity = string(trace.Severity)
	row.Action = trace.Action
	row.Status = trace.Status
	if trace.Fallback {
		row.Fallback = 1
	} else {
		row.Fallback = 0
	}
	row.Error = trace.Error
	row.CreatedAt = createdAt
	row.UpdatedAt = trace.UpdatedAt
	if row.UpdatedAt <= 0 {
		row.UpdatedAt = time.Now().UnixMilli()
	}
	if row.ID == 0 {
		_, err = o.Insert(&row)
		return err
	}
	_, err = o.Update(&row)
	return err
}

type TraceListOptions struct {
	Page            int
	Limit           int
	Symbol          string
	Type            string
	Severity        string
	Status          string
	Fallback        *int
	HasNotification *bool
	StartTime       int64
	EndTime         int64
}

type TraceListItem struct {
	Trace
	TaskStatus   string               `json:"task_status,omitempty"`
	TaskStage    string               `json:"task_stage,omitempty"`
	TaskError    string               `json:"task_error,omitempty"`
	Notification *models.Notification `json:"notification,omitempty"`
}

type TraceListResult struct {
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
	Total int64           `json:"total"`
	List  []TraceListItem `json:"list"`
}

func (store ORMTraceStore) List(ctx context.Context, options TraceListOptions) (TraceListResult, error) {
	if err := ctx.Err(); err != nil {
		return TraceListResult{}, err
	}
	if options.Page < 1 {
		options.Page = 1
	}
	if options.Limit < 1 {
		options.Limit = 20
	}
	if options.Limit > 100 {
		options.Limit = 100
	}
	o := store.ormer()
	query := o.QueryTable(new(models.AgentAlertPipelineTrace))
	if value := strings.ToUpper(strings.TrimSpace(options.Symbol)); value != "" {
		query = query.Filter("symbol", value)
	}
	if value := strings.TrimSpace(options.Type); value != "" {
		query = query.Filter("type", value)
	}
	if value := strings.TrimSpace(options.Severity); value != "" {
		query = query.Filter("severity", value)
	}
	if value := strings.TrimSpace(options.Status); value != "" {
		query = query.Filter("status", value)
	}
	if options.Fallback != nil {
		query = query.Filter("fallback", *options.Fallback)
	}
	if options.HasNotification != nil {
		if *options.HasNotification {
			query = query.Filter("notification_id__gt", 0)
		} else {
			query = query.Filter("notification_id", 0)
		}
	}
	if options.StartTime > 0 {
		query = query.Filter("created_at__gte", options.StartTime)
	}
	if options.EndTime > 0 {
		query = query.Filter("created_at__lte", options.EndTime)
	}
	total, err := query.Count()
	if err != nil {
		return TraceListResult{}, err
	}
	var rows []models.AgentAlertPipelineTrace
	if _, err := query.OrderBy("-created_at").Limit(options.Limit, (options.Page-1)*options.Limit).All(&rows); err != nil {
		return TraceListResult{}, err
	}
	items := make([]TraceListItem, 0, len(rows))
	notificationIDs := make([]interface{}, 0)
	taskIDs := make([]interface{}, 0)
	for _, row := range rows {
		if row.NotificationID > 0 {
			notificationIDs = append(notificationIDs, row.NotificationID)
		}
		if strings.TrimSpace(row.TaskID) != "" {
			taskIDs = append(taskIDs, row.TaskID)
		}
	}
	notificationByID := map[int64]models.Notification{}
	if len(notificationIDs) > 0 {
		var values []models.Notification
		if _, err := o.QueryTable(new(models.Notification)).Filter("id__in", notificationIDs...).All(&values); err != nil {
			return TraceListResult{}, err
		}
		for _, value := range values {
			notificationByID[value.ID] = value
		}
	}
	taskByID := map[string]models.AgentTask{}
	if len(taskIDs) > 0 {
		var values []models.AgentTask
		if _, err := o.QueryTable(new(models.AgentTask)).Filter("id__in", taskIDs...).All(&values, "ID", "Status", "Stage", "Error"); err != nil {
			return TraceListResult{}, err
		}
		for _, value := range values {
			taskByID[value.ID] = value
		}
	}
	for _, row := range rows {
		trace := Trace{EventID: row.EventID, SignalID: row.SignalID, TaskID: row.TaskID, NotificationID: row.NotificationID, Symbol: row.Symbol, Type: signalType(row.Type), Severity: signalSeverity(row.Severity), Action: row.Action, Status: row.Status, Fallback: row.Fallback == 1, Error: row.Error, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
		item := TraceListItem{Trace: trace}
		if value, ok := taskByID[row.TaskID]; ok {
			item.TaskStatus, item.TaskStage, item.TaskError = value.Status, value.Stage, value.Error
		}
		if value, ok := notificationByID[row.NotificationID]; ok {
			copy := value
			item.Notification = &copy
		}
		items = append(items, item)
	}
	return TraceListResult{Page: options.Page, Limit: options.Limit, Total: total, List: items}, nil
}

func signalType(value string) signalservice.Type         { return signalservice.Type(value) }
func signalSeverity(value string) signalservice.Severity { return signalservice.Severity(value) }
