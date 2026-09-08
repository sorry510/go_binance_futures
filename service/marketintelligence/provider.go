package marketintelligence

import (
	"context"
	"strings"
)

// SyncProviders fetches external discrete events through a bounded provider
// interface. A provider failure is recorded and returned as data quality state;
// it never prevents the remaining providers from being synchronized.
func (service Service) SyncProviders(ctx context.Context, since int64, providers ...Provider) []ProviderSyncResult {
	results := make([]ProviderSyncResult, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		source := strings.TrimSpace(provider.Name())
		result := ProviderSyncResult{Source: source}
		if err := ctx.Err(); err != nil {
			result.Error = err.Error()
			results = append(results, result)
			break
		}
		events, err := provider.Fetch(ctx, since)
		if err != nil {
			result.Error = err.Error()
			_ = service.RecordSourceFailure(context.Background(), source, service.now().UnixMilli(), err)
			results = append(results, result)
			continue
		}
		result.Fetched = len(events)
		providerFailed := false
		for _, input := range events {
			if strings.TrimSpace(input.Source) == "" {
				input.Source = source
			}
			_, created, ingestErr := service.IngestEvent(ctx, input)
			if ingestErr != nil {
				providerFailed = true
				result.Error = ingestErr.Error()
				_ = service.RecordSourceFailure(context.Background(), source, service.now().UnixMilli(), ingestErr)
				continue
			}
			if created {
				result.Inserted++
			} else {
				result.Merged++
			}
		}
		if !providerFailed {
			_ = service.RecordSourceSuccess(context.Background(), source, service.now().UnixMilli())
		}
		results = append(results, result)
	}
	return results
}
