package opportunity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

type Service struct {
	Store Store
	Now   func() time.Time
}

func DefaultService() Service { return Service{} }

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Service) Create(ctx context.Context, input CreateInput) (models.AgentOpportunity, bool, error) {
	if err := ctx.Err(); err != nil {
		return models.AgentOpportunity{}, false, err
	}
	input.Symbol = strings.ToUpper(strings.TrimSpace(input.Symbol))
	input.SourceType = strings.ToLower(strings.TrimSpace(input.SourceType))
	input.SourceID = strings.TrimSpace(input.SourceID)
	input.Direction = strings.ToLower(strings.TrimSpace(input.Direction))
	input.AnalysisStatus = strings.ToLower(strings.TrimSpace(input.AnalysisStatus))
	if err := validateCreateInput(input); err != nil {
		return models.AgentOpportunity{}, false, err
	}
	if existing, err := s.Store.FindBySource(ctx, input.SourceType, input.SourceID); err == nil {
		return existing, false, nil
	} else if err != orm.ErrNoRows {
		return models.AgentOpportunity{}, false, err
	}
	now := s.now().UnixMilli()
	if input.ExpiresAt <= 0 {
		input.ExpiresAt = s.now().Add(time.Hour).UnixMilli()
	}
	if input.ExpiresAt <= now {
		return models.AgentOpportunity{}, false, fmt.Errorf("opportunity expires_at must be in the future")
	}
	id, err := newOpportunityID()
	if err != nil {
		return models.AgentOpportunity{}, false, err
	}
	row := models.AgentOpportunity{
		OpportunityID: id, Symbol: input.Symbol, Direction: input.Direction,
		SourceType: input.SourceType, SourceID: input.SourceID, AnalysisTaskID: strings.TrimSpace(input.AnalysisTaskID),
		Summary: strings.TrimSpace(input.Summary), Confidence: input.Confidence, MarketCondition: input.MarketCondition,
		AnalysisStatus: input.AnalysisStatus, AnalysisError: strings.TrimSpace(input.AnalysisError),
		Status: StatusNew, ExpiresAt: input.ExpiresAt, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.Store.Insert(ctx, &row); err != nil {
		// Composite source uniqueness is the final race-safe dedup guard.
		if existing, readErr := s.Store.FindBySource(ctx, input.SourceType, input.SourceID); readErr == nil {
			return existing, false, nil
		}
		return models.AgentOpportunity{}, false, err
	}
	return row, true, nil
}

func (s Service) Get(ctx context.Context, opportunityID string) (models.AgentOpportunity, error) {
	return s.Store.Get(ctx, opportunityID)
}

func (s Service) List(ctx context.Context, opt ListOptions) (ListResult, error) {
	return s.Store.List(ctx, opt)
}

func (s Service) InCooldown(ctx context.Context, symbol, sourceType string, since time.Time) (bool, models.AgentOpportunity, error) {
	row, err := s.Store.FindRecent(ctx, symbol, strings.ToLower(strings.TrimSpace(sourceType)), since.UTC().UnixMilli())
	if err == orm.ErrNoRows {
		return false, models.AgentOpportunity{}, nil
	}
	if err != nil {
		return false, models.AgentOpportunity{}, err
	}
	return true, row, nil
}

func (s Service) UpdateAnalysis(ctx context.Context, opportunityID string, update AnalysisUpdate) (models.AgentOpportunity, error) {
	row, err := s.Store.Get(ctx, opportunityID)
	if err != nil {
		return models.AgentOpportunity{}, err
	}
	if row.Status == StatusExpired {
		return models.AgentOpportunity{}, fmt.Errorf("expired opportunity cannot be updated")
	}
	update.Direction = strings.ToLower(strings.TrimSpace(update.Direction))
	update.Status = strings.ToLower(strings.TrimSpace(update.Status))
	if !validDirection(update.Direction) {
		return models.AgentOpportunity{}, fmt.Errorf("invalid opportunity direction %q", update.Direction)
	}
	if !validAnalysisStatus(update.Status) || update.Status == AnalysisPending {
		return models.AgentOpportunity{}, fmt.Errorf("invalid completed analysis status %q", update.Status)
	}
	if update.Confidence < 0 || update.Confidence > 1 {
		return models.AgentOpportunity{}, fmt.Errorf("confidence must be between 0 and 1")
	}
	row.Direction = update.Direction
	row.AnalysisTaskID = strings.TrimSpace(update.TaskID)
	row.Summary = strings.TrimSpace(update.Summary)
	row.Confidence = update.Confidence
	row.MarketCondition = update.MarketCondition
	row.AnalysisStatus = update.Status
	row.AnalysisError = strings.TrimSpace(update.Error)
	row.UpdatedAt = s.now().UnixMilli()
	if err := s.Store.Save(ctx, &row); err != nil {
		return models.AgentOpportunity{}, err
	}
	return row, nil
}

func (s Service) MarkReviewed(ctx context.Context, opportunityID string) (models.AgentOpportunity, error) {
	row, err := s.Store.Get(ctx, opportunityID)
	if err != nil {
		return models.AgentOpportunity{}, err
	}
	if row.Status == StatusExpired {
		return models.AgentOpportunity{}, fmt.Errorf("expired opportunity cannot be reviewed")
	}
	if row.Status == StatusReviewed {
		return row, nil
	}
	now := s.now().UnixMilli()
	row.Status, row.ReviewedAt, row.UpdatedAt = StatusReviewed, now, now
	if err := s.Store.Save(ctx, &row); err != nil {
		return models.AgentOpportunity{}, err
	}
	return row, nil
}

func (s Service) Expire(ctx context.Context) (int64, error) {
	return s.Store.Expire(ctx, s.now().UnixMilli())
}

func CanCreateProposal(row models.AgentOpportunity, now time.Time) bool {
	if row.Status == StatusExpired || (row.ExpiresAt > 0 && row.ExpiresAt <= now.UTC().UnixMilli()) {
		return false
	}
	if row.AnalysisStatus != AnalysisSucceeded {
		return false
	}
	return row.Direction == DirectionLong || row.Direction == DirectionShort
}

func validateCreateInput(input CreateInput) error {
	if input.Symbol == "" || !strings.HasSuffix(input.Symbol, "USDT") {
		return fmt.Errorf("opportunity symbol must be a USDT futures contract")
	}
	if input.SourceType == "" || input.SourceID == "" {
		return fmt.Errorf("opportunity source_type and source_id are required")
	}
	if !validDirection(input.Direction) {
		return fmt.Errorf("invalid opportunity direction %q", input.Direction)
	}
	if !validAnalysisStatus(input.AnalysisStatus) {
		return fmt.Errorf("invalid opportunity analysis status %q", input.AnalysisStatus)
	}
	if input.Confidence < 0 || input.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	return nil
}

func validDirection(value string) bool {
	return value == DirectionLong || value == DirectionShort || value == DirectionNeutral
}

func validAnalysisStatus(value string) bool {
	switch value {
	case AnalysisPending, AnalysisSucceeded, AnalysisPartial, AnalysisDataMissing, AnalysisFailed:
		return true
	default:
		return false
	}
}

func newOpportunityID() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "opp_" + hex.EncodeToString(buffer), nil
}
