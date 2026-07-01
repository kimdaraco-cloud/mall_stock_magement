// @ai-modified 2026-07-02 add report service
package service

import (
	"context"

	"mallstock/internal/models"
	"mallstock/internal/repository"
)

// ReportService exposes dashboard and report reads.
type ReportService struct {
	Reports   *repository.ReportRepo
	Movements *repository.MovementRepo
}

func (s *ReportService) DashboardStats(ctx context.Context) (*models.DashboardStats, error) {
	return s.Reports.DashboardStats(ctx)
}

func (s *ReportService) LowStock(ctx context.Context, storeID int64) ([]models.LowStockItem, error) {
	return s.Reports.LowStock(ctx, storeID)
}

func (s *ReportService) ValuationByStore(ctx context.Context) ([]models.ValuationRow, error) {
	return s.Reports.ValuationByStore(ctx)
}

func (s *ReportService) ValuationByCategory(ctx context.Context) ([]models.ValuationRow, error) {
	return s.Reports.ValuationByCategory(ctx)
}

// MovementReport returns filtered movements for the report + CSV export.
func (s *ReportService) MovementReport(ctx context.Context, f repository.MovementFilter) ([]models.StockMovement, int, error) {
	return s.Movements.List(ctx, f)
}
