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
}
type UpdateInput struct {
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Enabled     int    `json:"enabled"`
}

func (store Store) ormer() orm.Ormer {
	if strings.TrimSpace(store.Alias) != "" {
		return orm.NewOrmUsingDB(store.Alias)
	}
	return orm.NewOrm()
}

func validateEnabled(enabled int) error {
	if enabled != 0 && enabled != 1 {
		return fmt.Errorf("enabled must be 0 or 1")
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
	if err := validateEnabled(input.Enabled); err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	o := store.ormer()
	existing := &models.AgentSkill{Name: input.Name}
	if err := o.Read(existing, "Name"); err == nil && existing.Deleted == 1 {
		existing.DisplayName, existing.Description, existing.Enabled = input.DisplayName, strings.TrimSpace(input.Description), input.Enabled
		existing.Deleted, existing.UpdatedAt = 0, now
		_, err = o.Update(existing, "DisplayName", "Description", "Enabled", "Deleted", "UpdatedAt")
		return existing, err
	}
	item := &models.AgentSkill{
		Name: input.Name, DisplayName: input.DisplayName, Description: strings.TrimSpace(input.Description),
		Enabled: input.Enabled, CreatedAt: now, UpdatedAt: now,
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
	if err := validateEnabled(input.Enabled); err != nil {
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
	item.UpdatedAt = time.Now().UnixMilli()
	_, err := o.Update(item, "DisplayName", "Description", "Enabled", "UpdatedAt")
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
	count, err := o.QueryTable(new(models.AgentSkill)).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	for _, input := range defaults {
		if _, err := store.Create(ctx, input); err != nil {
			return fmt.Errorf("create default skill %s: %w", input.Name, err)
		}
	}
	return nil
}
