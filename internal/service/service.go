package service

import (
	"ad-service/internal/domain"
	"ad-service/internal/infrastructure/metrics"
	"ad-service/internal/repository"
	"context"
	"database/sql"
	"errors"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrInvalidID  = errors.New("invalid ad ID")
	ErrAdNotFound = errors.New("ad not found")
)

type PaginationResult struct {
	Ads         []*domain.Ad `json:"ads"`
	CurrentPage int          `json:"current_page"`
	NextPage    int          `json:"next_page,omitempty"`
	PrevPage    int          `json:"prev_page,omitempty"`
	TotalPages  int          `json:"total_pages"`
}

type AdService interface {
	GetAllAds(ctx context.Context, limit int, offset int, sortBy string, order string) (*PaginationResult, error)
	GetAdByID(ctx context.Context, id int64) (*domain.Ad, error)
	CreateAd(ctx context.Context, ad *domain.Ad) (*domain.Ad, error)
	UpdateAd(ctx context.Context, ad *domain.Ad) (*domain.Ad, error)
	DeleteAd(ctx context.Context, id int64) error
}

type adService struct {
	repository repository.AdRepository
	metrics    *metrics.ServiceMetrics
	tracer     trace.Tracer
}

func NewAdService(repository repository.AdRepository, metrics *metrics.ServiceMetrics) AdService {
	tracer := otel.Tracer("ad-service/service")
	return &adService{
		repository: repository,
		metrics:    metrics,
		tracer:     tracer,
	}
}

func (s *adService) GetAllAds(ctx context.Context, limit int, offset int, sortBy string, order string) (*PaginationResult, error) {
	ctx, span := s.tracer.Start(ctx, "GetAllAds")
	defer span.End()

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		s.metrics.MethodCount.WithLabelValues("GetAllAds", status).Inc()
		s.metrics.MethodDuration.WithLabelValues("GetAllAds", status).Observe(duration)
	}()

	ads, err := s.repository.GetAllAds(ctx, limit, offset, sortBy, order)
	if err != nil {
		status = "error"
		span.RecordError(err)
		return nil, err
	}

	totalCount, err := s.repository.CountAds(ctx)
	if err != nil {
		status = "error"
		span.RecordError(err)
		return nil, err
	}

	totalPages := (totalCount + limit - 1) / limit
	currentPage := (offset / limit) + 1

	var nextPage, prevPage int
	if currentPage < totalPages {
		nextPage = currentPage + 1
	}
	if currentPage > 1 {
		prevPage = currentPage - 1
	}

	span.SetAttributes(
		attribute.Int("ads.limit", limit),
		attribute.Int("ads.offset", offset),
		attribute.String("ads.sort_by", sortBy),
		attribute.String("ads.order", order),
		attribute.Int("ads.total_count", totalCount),
	)

	return &PaginationResult{
		Ads:         ads,
		CurrentPage: currentPage,
		NextPage:    nextPage,
		PrevPage:    prevPage,
		TotalPages:  totalPages,
	}, nil
}

func (s *adService) GetAdByID(ctx context.Context, id int64) (*domain.Ad, error) {
	if id <= 0 {
		err := ErrInvalidID
		return nil, err
	}

	ctx, span := s.tracer.Start(ctx, "GetAdByID")
	defer span.End()

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		s.metrics.MethodCount.WithLabelValues("GetAdByID", status).Inc()
		s.metrics.MethodDuration.WithLabelValues("GetAdByID", status).Observe(duration)
	}()

	ad, err := s.repository.GetAdByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			status = "not_found"
			return nil, ErrAdNotFound
		}
		status = "error"
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(attribute.Int64("ad.id", id))
	return ad, nil
}

func (s *adService) CreateAd(ctx context.Context, ad *domain.Ad) (*domain.Ad, error) {
	ctx, span := s.tracer.Start(ctx, "CreateAd")
	defer span.End()

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		s.metrics.MethodCount.WithLabelValues("CreateAd", status).Inc()
		s.metrics.MethodDuration.WithLabelValues("CreateAd", status).Observe(duration)
	}()

	createdAd, err := s.repository.CreateAd(ctx, ad)
	if err != nil {
		status = "error"
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int64("ad.id", createdAd.ID),
		attribute.String("ad.title", createdAd.Title),
		attribute.Float64("ad.price", createdAd.Price),
	)
	return createdAd, nil
}

func (s *adService) UpdateAd(ctx context.Context, ad *domain.Ad) (*domain.Ad, error) {
	if ad.ID <= 0 {
		err := ErrInvalidID
		return nil, err
	}

	ctx, span := s.tracer.Start(ctx, "UpdateAd")
	defer span.End()

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		s.metrics.MethodCount.WithLabelValues("UpdateAd", status).Inc()
		s.metrics.MethodDuration.WithLabelValues("UpdateAd", status).Observe(duration)
	}()

	updatedAd, err := s.repository.UpdateAd(ctx, ad)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			status = "not_found"
			return nil, ErrAdNotFound
		}
		status = "error"
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int64("ad.id", updatedAd.ID),
		attribute.String("ad.title", updatedAd.Title),
		attribute.Float64("ad.price", updatedAd.Price),
	)
	return updatedAd, nil
}

func (s *adService) DeleteAd(ctx context.Context, id int64) error {
	if id <= 0 {
		err := ErrInvalidID
		return err
	}

	ctx, span := s.tracer.Start(ctx, "DeleteAd")
	defer span.End()

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		s.metrics.MethodCount.WithLabelValues("DeleteAd", status).Inc()
		s.metrics.MethodDuration.WithLabelValues("DeleteAd", status).Observe(duration)
	}()

	err := s.repository.DeleteAd(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			status = "not_found"
			return ErrAdNotFound
		}
		status = "error"
		span.RecordError(err)
		return err
	}

	span.SetAttributes(attribute.Int64("ad.id", id))
	return nil
}
