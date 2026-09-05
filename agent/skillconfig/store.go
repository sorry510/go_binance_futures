package skillconfig

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

type Store struct {
	Alias string
}

type CreateInput struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Enabled     int    `json:"enabled"`
	ChatEnabled int    `json:"chat_enabled"`
}
type UpdateInput struct {
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Enabled     int    `json:"enabled"`
	ChatEnabled int    `json:"chat_enabled"`
}

type ListOptions struct {
	Type    string
	Keyword string
	Page    int
	Limit   int
}

type ListResult struct {
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
	Total int                 `json:"total"`
	List  []models.AgentSkill `json:"list"`
}

func (store Store) ormer() orm.Ormer {
	if strings.TrimSpace(store.Alias) != "" {
		return orm.NewOrmUsingDB(store.Alias)
	}
	return orm.NewOrm()
}

func validateToggle(name string, value int) error {
	if value != 0 && value != 1 {
		return fmt.Errorf("%s must be 0 or 1", name)
	}
	return nil
}
func (store Store) List(ctx context.Context) ([]models.AgentSkill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items := make([]models.AgentSkill, 0)
	_, err := store.ormer().QueryTable(new(models.AgentSkill)).Filter("Deleted", 0).OrderBy("id").All(&items)
	return items, err
}

func (store Store) ListPage(ctx context.Context, options ListOptions) (ListResult, error) {
	items, err := store.List(ctx)
	if err != nil {
		return ListResult{}, err
	}
	kind := strings.ToLower(strings.TrimSpace(options.Type))
	keyword := strings.ToLower(strings.TrimSpace(options.Keyword))
	filtered := make([]models.AgentSkill, 0, len(items))
	for _, item := range items {
		if kind != "" && strings.ToLower(strings.TrimSpace(item.Type)) != kind {
			continue
		}
		if keyword != "" {
			haystack := strings.ToLower(strings.Join([]string{item.Name, item.DisplayName, item.Description}, "\n"))
			if !strings.Contains(haystack, keyword) {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	page, limit := options.Page, options.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	start := (page - 1) * limit
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return ListResult{Page: page, Limit: limit, Total: len(filtered), List: filtered[start:end]}, nil
}

func (store Store) GetByName(ctx context.Context, name string) (*models.AgentSkill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	item := &models.AgentSkill{Name: strings.TrimSpace(name)}
	if item.Name == "" {
		return nil, fmt.Errorf("skill name is required")
	}
	if err := store.ormer().QueryTable(new(models.AgentSkill)).Filter("Name", item.Name).Filter("Deleted", 0).One(item); err != nil {
		return nil, err
	}
	return item, nil
}
func (store Store) Create(ctx context.Context, input CreateInput) (*models.AgentSkill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	input.Name = strings.TrimSpace(input.Name)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.Name == "" {
		return nil, fmt.Errorf("skill name is required")
	}
	if err := validateToggle("enabled", input.Enabled); err != nil {
		return nil, err
	}
	if err := validateToggle("chat_enabled", input.ChatEnabled); err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	o := store.ormer()
	existing := &models.AgentSkill{Name: input.Name}
	if err := o.Read(existing, "Name"); err == nil && existing.Deleted == 1 {
		existing.DisplayName, existing.Description, existing.Enabled = input.DisplayName, strings.TrimSpace(input.Description), input.Enabled
		existing.ChatEnabled, existing.Deleted, existing.UpdatedAt = input.ChatEnabled, 0, now
		_, err = o.Update(existing, "DisplayName", "Description", "Enabled", "ChatEnabled", "Deleted", "UpdatedAt")
		return existing, err
	}
	item := &models.AgentSkill{
		Name: input.Name, DisplayName: input.DisplayName, Description: strings.TrimSpace(input.Description),
		Type: "native", Enabled: input.Enabled, ChatEnabled: input.ChatEnabled, CreatedAt: now, UpdatedAt: now,
	}
	id, err := o.Insert(item)
	if err != nil {
		return nil, err
	}
	item.ID = id
	return item, nil
}
func (store Store) Update(ctx context.Context, id int64, input UpdateInput) (*models.AgentSkill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateToggle("enabled", input.Enabled); err != nil {
		return nil, err
	}
	if err := validateToggle("chat_enabled", input.ChatEnabled); err != nil {
		return nil, err
	}
	item := &models.AgentSkill{ID: id}
	o := store.ormer()
	if err := o.Read(item); err != nil {
		return nil, err
	}
	item.DisplayName = strings.TrimSpace(input.DisplayName)
	item.Description = strings.TrimSpace(input.Description)
	item.Enabled = input.Enabled
	item.ChatEnabled = input.ChatEnabled
	item.UpdatedAt = time.Now().UnixMilli()
	_, err := o.Update(item, "DisplayName", "Description", "Enabled", "ChatEnabled", "UpdatedAt")
	return item, err
}

func (store Store) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	o := store.ormer()
	item := &models.AgentSkill{ID: id}
	if err := o.Read(item); err != nil {
		return err
	}
	item.Enabled, item.Deleted, item.UpdatedAt = 0, 1, time.Now().UnixMilli()
	_, err := o.Update(item, "Enabled", "Deleted", "UpdatedAt")
	return err
}
func (store Store) EnsureDefaults(ctx context.Context, defaults []CreateInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	o := store.ormer()
	for _, input := range defaults {
		var existing models.AgentSkill
		err := o.QueryTable(new(models.AgentSkill)).Filter("Name", strings.TrimSpace(input.Name)).One(&existing)
		if err == nil {
			if existing.Type != "" && existing.Type != "native" {
				return fmt.Errorf("default native skill %s conflicts with %s skill", input.Name, existing.Type)
			}
			fields := make([]string, 0, 2)
			if existing.Type == "" {
				existing.Type = "native"
				fields = append(fields, "Type")
			}
			if existing.ChatEnabled != 0 && existing.ChatEnabled != 1 {
				existing.ChatEnabled = input.ChatEnabled
				fields = append(fields, "ChatEnabled")
			}
			if len(fields) > 0 {
				_, err = o.Update(&existing, fields...)
			}
			if err != nil {
				return err
			}
			continue
		}
		if err != orm.ErrNoRows {
			return err
		}
		if _, err := store.Create(ctx, input); err != nil {
			return fmt.Errorf("create default skill %s: %w", input.Name, err)
		}
	}
	return nil
}
