package reports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

// ReportStore is the interface for report_generations CRUD operations.
// Implementations are expected to enforce tenant-scoped RLS via SET LOCAL.
type ReportStore interface {
	CreateReportGeneration(ctx context.Context, arg CreateReportGenerationParams) (ReportRecord, error)
	UpdateReportStatus(ctx context.Context, arg UpdateReportStatusParams) error
	GetReportGeneration(ctx context.Context, tenantID, id string) (ReportRecord, error)
	ListReportGenerations(ctx context.Context, arg ListReportGenerationsParams) ([]ReportRecord, error)
	CountReportGenerations(ctx context.Context, tenantID string) (int64, error)
	CountReportGenerationsToday(ctx context.Context, tenantID string) (int64, error)
	DeleteReportGeneration(ctx context.Context, tenantID, id string) error
	DeleteExpiredReports(ctx context.Context) ([]ReportRecord, error)
}

// CreateReportGenerationParams holds params for inserting a report_generations row.
type CreateReportGenerationParams struct {
	ID         string
	TenantID   string
	ReportType string
	Format     string
	Name       string
	Filters    ReportFilters
	CreatedBy  string
	ExpiresAt  time.Time
}

// UpdateReportStatusParams holds params for updating a report_generations row.
type UpdateReportStatusParams struct {
	ID             string
	TenantID       string
	Status         string
	FilePath       string
	FileSizeBytes  int64
	ChecksumSHA256 string
	RowCount       int
	ErrorMessage   string
	CompletedAt    *time.Time
}

// ListReportGenerationsParams holds params for listing report_generations.
type ListReportGenerationsParams struct {
	TenantID   string
	Status     string
	ReportType string
	Format     string
	Cursor     string
	Limit      int32
}

// assemblerFactory creates an Assembler. Factories allow lazy construction
// with request-scoped dependencies if needed.
type assemblerFactory func() Assembler

// ReportCounts holds aggregate counts for the stat cards.
type ReportCounts struct {
	Total     int64 `json:"total"`
	Today     int64 `json:"today"`
	Completed int64 `json:"completed"`
}

// Service orchestrates report generation: assemblers → renderers → MinIO storage.
type Service struct {
	store      ReportStore
	minio      *minio.Client
	bucket     string
	assemblers map[ReportType]assemblerFactory
	renderers  map[ReportFormat]Renderer
}

// NewService creates a report Service.
func NewService(store ReportStore, minioClient *minio.Client, bucket string) *Service {
	return &Service{
		store:      store,
		minio:      minioClient,
		bucket:     bucket,
		assemblers: make(map[ReportType]assemblerFactory),
		renderers:  make(map[ReportFormat]Renderer),
	}
}

// RegisterAssembler registers an assembler factory for a report type.
func (s *Service) RegisterAssembler(rt ReportType, factory assemblerFactory) {
	s.assemblers[rt] = factory
}

// RegisterRenderer registers a renderer for a format.
func (s *Service) RegisterRenderer(rf ReportFormat, renderer Renderer) {
	s.renderers[rf] = renderer
}

// validReportTypes is the set of allowed report types.
var validReportTypes = map[ReportType]bool{
	ReportEndpoints:   true,
	ReportPatches:     true,
	ReportCVEs:        true,
	ReportDeployments: true,
	ReportCompliance:  true,
	ReportExecutive:   true,
}

// validFormats is the set of allowed output formats.
var validFormats = map[ReportFormat]bool{
	FormatPDF:  true,
	FormatCSV:  true,
	FormatXLSX: true,
}

// Generate creates a report synchronously: assemble data → render → upload to MinIO.
func (s *Service) Generate(ctx context.Context, tenantID, userID string, req GenerateRequest) (*GenerateResponse, error) {
	if !validReportTypes[req.ReportType] {
		return nil, fmt.Errorf("generate report: unsupported report_type %q", req.ReportType)
	}
	if !validFormats[req.Format] {
		return nil, fmt.Errorf("generate report: unsupported format %q", req.Format)
	}

	factory, ok := s.assemblers[req.ReportType]
	if !ok {
		return nil, fmt.Errorf("generate report: no assembler registered for report_type %q", req.ReportType)
	}
	renderer, ok := s.renderers[req.Format]
	if !ok {
		return nil, fmt.Errorf("generate report: no renderer registered for format %q", req.Format)
	}

	id := uuid.New().String()
	now := time.Now().In(IST)
	name := fmt.Sprintf("%s_%s_%s", req.ReportType, req.Format, now.Format("20060102_150405"))

	record, err := s.store.CreateReportGeneration(ctx, CreateReportGenerationParams{
		ID:         id,
		TenantID:   tenantID,
		ReportType: string(req.ReportType),
		Format:     string(req.Format),
		Name:       name,
		Filters:    req.Filters,
		CreatedBy:  userID,
		ExpiresAt:  now.Add(7 * 24 * time.Hour), // 7-day retention
	})
	if err != nil {
		return nil, fmt.Errorf("generate report: create record: %w", err)
	}

	// Update status to generating.
	if err := s.store.UpdateReportStatus(ctx, UpdateReportStatusParams{
		ID:       id,
		TenantID: tenantID,
		Status:   "generating",
	}); err != nil {
		slog.ErrorContext(ctx, "generate report: update status to generating", "id", id, "error", err)
	}

	// Assemble report data.
	assembler := factory()
	data, err := assembler.Assemble(ctx, AssembleOptions{
		TenantID:    tenantID,
		GeneratedBy: userID,
		Filters:     req.Filters,
	})
	if err != nil {
		s.failReport(ctx, tenantID, id, fmt.Sprintf("assemble failed: %v", err))
		return nil, fmt.Errorf("generate report: assemble: %w", err)
	}

	// Render to bytes.
	output, err := renderer.Render(data)
	if err != nil {
		s.failReport(ctx, tenantID, id, fmt.Sprintf("render failed: %v", err))
		return nil, fmt.Errorf("generate report: render: %w", err)
	}

	// Upload to MinIO.
	ext := renderer.FileExtension()
	objectPath := fmt.Sprintf("reports/%s/%s/%s.%s", tenantID, req.ReportType, id, ext)
	contentType := renderer.ContentType()

	_, err = s.minio.PutObject(ctx, s.bucket, objectPath, bytes.NewReader(output), int64(len(output)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		s.failReport(ctx, tenantID, id, fmt.Sprintf("upload to storage failed: %v", err))
		return nil, fmt.Errorf("generate report: upload to minio: %w", err)
	}

	// Compute checksum.
	checksum := fmt.Sprintf("%x", sha256.Sum256(output))
	completedAt := time.Now().In(IST)

	// Update status to completed.
	if err := s.store.UpdateReportStatus(ctx, UpdateReportStatusParams{
		ID:             id,
		TenantID:       tenantID,
		Status:         "completed",
		FilePath:       objectPath,
		FileSizeBytes:  int64(len(output)),
		ChecksumSHA256: checksum,
		RowCount:       data.Detail.TotalRows,
		CompletedAt:    &completedAt,
	}); err != nil {
		return nil, fmt.Errorf("generate report: update status to completed: %w", err)
	}

	slog.InfoContext(ctx, "report generated",
		"id", id,
		"tenant_id", tenantID,
		"report_type", req.ReportType,
		"format", req.Format,
		"file_size", len(output),
		"row_count", data.Detail.TotalRows,
	)

	return &GenerateResponse{
		ID:         id,
		ReportType: string(req.ReportType),
		Format:     string(req.Format),
		Status:     "completed",
		CreatedAt:  record.CreatedAt,
	}, nil
}

// Download retrieves a completed report's bytes from MinIO.
func (s *Service) Download(ctx context.Context, tenantID, id string) ([]byte, string, string, error) {
	record, err := s.store.GetReportGeneration(ctx, tenantID, id)
	if err != nil {
		return nil, "", "", fmt.Errorf("download report: get record: %w", err)
	}
	if record.Status != "completed" {
		return nil, "", "", fmt.Errorf("download report: report status is %q, expected completed", record.Status)
	}
	if record.FilePath == "" {
		return nil, "", "", fmt.Errorf("download report: no file path for report %s", id)
	}

	obj, err := s.minio.GetObject(ctx, s.bucket, record.FilePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", "", fmt.Errorf("download report: get object from storage: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, "", "", fmt.Errorf("download report: read object: %w", err)
	}

	// Determine content type and filename from format.
	contentType := "application/octet-stream"
	filename := record.Name
	switch ReportFormat(record.Format) {
	case FormatPDF:
		contentType = "application/pdf"
		filename += ".pdf"
	case FormatCSV:
		contentType = "text/csv"
		filename += ".csv"
	case FormatXLSX:
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		filename += ".xlsx"
	}

	return data, contentType, filename, nil
}

// Get retrieves a single report record.
func (s *Service) Get(ctx context.Context, tenantID, id string) (*ReportRecord, error) {
	record, err := s.store.GetReportGeneration(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get report: %w", err)
	}
	return &record, nil
}

// List returns a paginated list of report records.
func (s *Service) List(ctx context.Context, params ListReportGenerationsParams) ([]ReportRecord, int64, error) {
	records, err := s.store.ListReportGenerations(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list reports: %w", err)
	}
	total, err := s.store.CountReportGenerations(ctx, params.TenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("list reports: count: %w", err)
	}
	return records, total, nil
}

// GetCounts returns aggregate report counts for stat cards.
func (s *Service) GetCounts(ctx context.Context, tenantID string) (*ReportCounts, error) {
	total, err := s.store.CountReportGenerations(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get report counts: total: %w", err)
	}
	today, err := s.store.CountReportGenerationsToday(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get report counts: today: %w", err)
	}
	return &ReportCounts{
		Total: total,
		Today: today,
	}, nil
}

// Delete removes a report record and its MinIO object.
func (s *Service) Delete(ctx context.Context, tenantID, id string) error {
	record, err := s.store.GetReportGeneration(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete report: get record: %w", err)
	}

	// Remove from MinIO if file exists.
	if record.FilePath != "" {
		if err := s.minio.RemoveObject(ctx, s.bucket, record.FilePath, minio.RemoveObjectOptions{}); err != nil {
			slog.ErrorContext(ctx, "delete report: remove object from storage", "id", id, "path", record.FilePath, "error", err)
			// Continue with DB deletion even if MinIO removal fails.
		}
	}

	if err := s.store.DeleteReportGeneration(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete report: delete record: %w", err)
	}

	slog.InfoContext(ctx, "report deleted", "id", id, "tenant_id", tenantID)
	return nil
}

// CleanExpired deletes all expired reports and their MinIO objects.
func (s *Service) CleanExpired(ctx context.Context) error {
	expired, err := s.store.DeleteExpiredReports(ctx)
	if err != nil {
		return fmt.Errorf("clean expired reports: %w", err)
	}

	for _, r := range expired {
		if r.FilePath != "" {
			if err := s.minio.RemoveObject(ctx, s.bucket, r.FilePath, minio.RemoveObjectOptions{}); err != nil {
				slog.ErrorContext(ctx, "clean expired reports: remove object from storage", "id", r.ID, "path", r.FilePath, "error", err)
			}
		}
	}

	if len(expired) > 0 {
		slog.InfoContext(ctx, "cleaned expired reports", "count", len(expired))
	}
	return nil
}

// failReport updates a report's status to failed with an error message.
func (s *Service) failReport(ctx context.Context, tenantID, id, errMsg string) {
	if err := s.store.UpdateReportStatus(ctx, UpdateReportStatusParams{
		ID:           id,
		TenantID:     tenantID,
		Status:       "failed",
		ErrorMessage: errMsg,
	}); err != nil {
		slog.ErrorContext(ctx, "generate report: failed to update status to failed", "id", id, "error", err)
	}
}
