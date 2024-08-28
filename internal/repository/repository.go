package repository

import (
	"ad-service/internal/domain"
	"ad-service/internal/infrastructure/cache"
	"ad-service/internal/infrastructure/metrics"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type AdRepository interface {
	GetAllAds(ctx context.Context, page int, pageSize int, sortBy string, sortOrder string) ([]*domain.Ad, error)
	GetAdByID(ctx context.Context, id int64) (*domain.Ad, error)
	CreateAd(ctx context.Context, ad *domain.Ad) (*domain.Ad, error)
	UpdateAd(ctx context.Context, ad *domain.Ad) (*domain.Ad, error)
	DeleteAd(ctx context.Context, id int64) error
	CountAds(ctx context.Context) (int, error)
}

type mysqlAdRepository struct {
	db      *sql.DB
	cache   cache.Cache
	metrics *metrics.RepositoryMetrics
	tracer  trace.Tracer
}

func NewMysqlAdRepository(db *sql.DB, cache cache.Cache, metrics *metrics.RepositoryMetrics) AdRepository {
	tracer := otel.Tracer("ad-service/repository")
	return &mysqlAdRepository{
		db:      db,
		cache:   cache,
		metrics: metrics,
		tracer:  tracer,
	}
}

func (r *mysqlAdRepository) GetAllAds(ctx context.Context, limit int, offset int, sortBy string, order string) ([]*domain.Ad, error) {
	ctx, span := r.tracer.Start(ctx, "Repository GetAllAds")
	defer span.End()

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		r.metrics.QueryCount.WithLabelValues("GetAllAds", status).Inc()
		r.metrics.QueryDuration.WithLabelValues("GetAllAds", status).Observe(duration)
	}()

	isDefaultPagination := limit == 10 && offset == 0 && sortBy == "created_at" && order == "ASC"
	cacheKey := "ads:default_page"

	if isDefaultPagination {
		cacheSpanCtx, cacheSpan := r.tracer.Start(ctx, "Redis Get")
		cachedAds, err := r.cache.Get(cacheSpanCtx, cacheKey)
		cacheSpan.End()

		if err == nil {
			var ads []*domain.Ad
			if err := json.Unmarshal([]byte(cachedAds), &ads); err == nil {
				return ads, nil
			}
		}
	}

	query := fmt.Sprintf(`
		SELECT id, title, description, price, created_at, updated_at, active
		FROM ads
		ORDER BY %s %s
		LIMIT ? OFFSET ?`, sortBy, order)

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		status = "error"
		span.RecordError(err)
		span.SetAttributes(
			attribute.String("query", query),
			attribute.Int("limit", limit),
			attribute.Int("offset", offset),
			attribute.String("sort_by", sortBy),
			attribute.String("order", order),
		)
		return nil, fmt.Errorf("failed to retrieve ads: %w", err)
	}
	defer rows.Close()

	var ads []*domain.Ad
	for rows.Next() {
		var ad domain.Ad
		if err := rows.Scan(&ad.ID, &ad.Title, &ad.Description, &ad.Price, &ad.CreatedAt, &ad.UpdatedAt, &ad.Active); err != nil {
			status = "error"
			span.RecordError(err)
			return nil, fmt.Errorf("failed to scan ad: %w", err)
		}
		ads = append(ads, &ad)
	}

	if err := rows.Err(); err != nil {
		status = "error"
		span.RecordError(err)
		return nil, fmt.Errorf("rows error: %w", err)
	}

	if isDefaultPagination {
		adsJSON, err := json.Marshal(ads)
		if err == nil {
			cacheSpanCtx, cacheSpan := r.tracer.Start(ctx, "Redis Set")
			r.cache.Set(cacheSpanCtx, cacheKey, string(adsJSON), 10*time.Minute)
			cacheSpan.End()
		}
	}

	return ads, nil
}

func (r *mysqlAdRepository) GetAdByID(ctx context.Context, id int64) (*domain.Ad, error) {
	ctx, span := r.tracer.Start(ctx, "Repository GetAdByID")
	defer span.End()

	span.SetAttributes(attribute.Int64("ad.id", id))

	cacheKey := fmt.Sprintf("ad:%d", id)

	cacheSpanCtx, cacheSpan := r.tracer.Start(ctx, "Redis Get")
	cachedAd, err := r.cache.Get(cacheSpanCtx, cacheKey)
	cacheSpan.End()

	if err == nil {
		var ad domain.Ad
		if err := json.Unmarshal([]byte(cachedAd), &ad); err == nil {
			return &ad, nil
		}
	}

	query := `
		SELECT id, title, description, price, created_at, updated_at, active 
		FROM ads 
		WHERE id = ?
	`

	ad := &domain.Ad{}

	err = r.db.QueryRowContext(ctx, query, id).Scan(
		&ad.ID,
		&ad.Title,
		&ad.Description,
		&ad.Price,
		&ad.CreatedAt,
		&ad.UpdatedAt,
		&ad.Active,
	)

	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	adJSON, err := json.Marshal(ad)
	if err == nil {
		cacheSpanCtx, cacheSpan := r.tracer.Start(ctx, "Redis Set")
		r.cache.Set(cacheSpanCtx, cacheKey, string(adJSON), 10*time.Minute)
		cacheSpan.End()
	}

	return ad, nil
}

func (r *mysqlAdRepository) CreateAd(ctx context.Context, ad *domain.Ad) (*domain.Ad, error) {
	ctx, span := r.tracer.Start(ctx, "Repository CreateAd")
	defer span.End()

	span.SetAttributes(
		attribute.String("ad.title", ad.Title),
		attribute.Float64("ad.price", ad.Price),
	)

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		r.metrics.QueryCount.WithLabelValues("CreateAd", status).Inc()
		r.metrics.QueryDuration.WithLabelValues("CreateAd", status).Observe(duration)
	}()

	result, err := r.db.ExecContext(ctx,
		"INSERT INTO ads (title, description, price, active) VALUES (?, ?, ?, ?)",
		ad.Title, ad.Description, ad.Price, ad.Active)
	if err != nil {
		status = "error"
		span.RecordError(err)
		return nil, fmt.Errorf("failed to insert ad: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		status = "error"
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	var insertedAd domain.Ad
	err = r.db.QueryRowContext(ctx, "SELECT id, title, description, price, active, created_at, updated_at FROM ads WHERE id = ?", id).Scan(
		&insertedAd.ID,
		&insertedAd.Title,
		&insertedAd.Description,
		&insertedAd.Price,
		&insertedAd.Active,
		&insertedAd.CreatedAt,
		&insertedAd.UpdatedAt,
	)
	if err != nil {
		status = "error"
		span.RecordError(err)
		return nil, fmt.Errorf("failed to fetch inserted ad: %w", err)
	}

	return &insertedAd, nil
}

func (r *mysqlAdRepository) UpdateAd(ctx context.Context, ad *domain.Ad) (*domain.Ad, error) {
	ctx, span := r.tracer.Start(ctx, "Repository UpdateAd")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("ad.id", ad.ID),
		attribute.String("ad.title", ad.Title),
		attribute.Float64("ad.price", ad.Price),
	)

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		r.metrics.QueryCount.WithLabelValues("UpdateAd", status).Inc()
		r.metrics.QueryDuration.WithLabelValues("UpdateAd", status).Observe(duration)
	}()

	query := `
		UPDATE ads
		SET title = ?, description = ?, price = ?, active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, ad.Title, ad.Description, ad.Price, ad.Active, ad.ID)
	if err != nil {
		status = "error"
		span.RecordError(err)
		return nil, fmt.Errorf("failed to update ad: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		status = "error"
		span.RecordError(err)
		return nil, fmt.Errorf("failed to retrieve rows affected: %w", err)
	}

	if rowsAffected == 0 {
		status = "not_found"
		return nil, sql.ErrNoRows
	}

	cacheKey := fmt.Sprintf("ad:%d", ad.ID)

	cacheSpanCtx, cacheSpan := r.tracer.Start(ctx, "Redis Delete")
	r.cache.Delete(cacheSpanCtx, cacheKey)
	cacheSpan.End()

	var updatedAd domain.Ad
	err = r.db.QueryRowContext(ctx, "SELECT id, title, description, price, active, created_at, updated_at FROM ads WHERE id = ?", ad.ID).Scan(
		&updatedAd.ID,
		&updatedAd.Title,
		&updatedAd.Description,
		&updatedAd.Price,
		&updatedAd.Active,
		&updatedAd.CreatedAt,
		&updatedAd.UpdatedAt,
	)
	if err != nil {
		status = "error"
		span.RecordError(err)
		return nil, fmt.Errorf("failed to fetch updated ad: %w", err)
	}

	updatedAdJSON, err := json.Marshal(&updatedAd)
	if err == nil {
		cacheSpanCtx, cacheSpan = r.tracer.Start(ctx, "Redis Set")
		r.cache.Set(cacheSpanCtx, cacheKey, string(updatedAdJSON), 10*time.Minute)
		cacheSpan.End()
	}

	return &updatedAd, nil
}

func (r *mysqlAdRepository) DeleteAd(ctx context.Context, id int64) error {
	ctx, span := r.tracer.Start(ctx, "Repository DeleteAd")
	defer span.End()

	span.SetAttributes(attribute.Int64("ad.id", id))

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		r.metrics.QueryCount.WithLabelValues("DeleteAd", status).Inc()
		r.metrics.QueryDuration.WithLabelValues("DeleteAd", status).Observe(duration)
	}()

	query := `
		DELETE FROM ads WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		status = "error"
		span.RecordError(err)
		return fmt.Errorf("failed to delete ad: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		status = "error"
		span.RecordError(err)
		return fmt.Errorf("failed to retrieve rows affected: %w", err)
	}

	if rowsAffected == 0 {
		status = "not_found"
		return sql.ErrNoRows
	}

	cacheKey := fmt.Sprintf("ad:%d", id)

	cacheSpanCtx, cacheSpan := r.tracer.Start(ctx, "Redis Delete")
	r.cache.Delete(cacheSpanCtx, cacheKey)
	cacheSpan.End()

	return nil
}

func (r *mysqlAdRepository) CountAds(ctx context.Context) (int, error) {
	ctx, span := r.tracer.Start(ctx, "Repository CountAds")
	defer span.End()

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		r.metrics.QueryCount.WithLabelValues("CountAds", status).Inc()
		r.metrics.QueryDuration.WithLabelValues("CountAds", status).Observe(duration)
	}()

	var count int
	query := "SELECT COUNT(*) FROM ads"
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		status = "error"
		span.RecordError(err)
		return 0, fmt.Errorf("failed to count ads: %w", err)
	}
	return count, nil
}
