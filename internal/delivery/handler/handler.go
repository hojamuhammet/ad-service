package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"ad-service/internal/domain"
	"ad-service/internal/service"
	"ad-service/pkg/logger"
	"ad-service/pkg/utils"

	"ad-service/internal/infrastructure/metrics"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type AdHandler struct {
	service service.AdService
	logger  *logger.Loggers
	metrics *metrics.HandlerMetrics
	tracer  trace.Tracer
}

func NewAdHandler(service service.AdService, logger *logger.Loggers, metrics *metrics.HandlerMetrics) *AdHandler {
	tracer := otel.Tracer("ad-service/handler")
	return &AdHandler{
		service: service,
		logger:  logger,
		metrics: metrics,
		tracer:  tracer,
	}
}

func (h *AdHandler) GetAdByID(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "GetAdByID")
	defer span.End()

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		h.metrics.RequestCount.WithLabelValues("GET", "/ads/{id}", status).Inc()
		h.metrics.RequestDuration.WithLabelValues("GET", "/ads/{id}", status).Observe(duration)
	}()

	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		status = "error"
		utils.RespondWithErrorJSON(w, http.StatusBadRequest, "missing id parameter")
		return
	}

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		status = "error"
		utils.RespondWithErrorJSON(w, http.StatusBadRequest, "invalid id parameter")
		return
	}

	span.SetAttributes(attribute.Int64("ad.id", id))

	ad, err := h.service.GetAdByID(ctx, id)
	if err != nil {
		if errors.Is(err, service.ErrInvalidID) {
			status = "error"
			utils.RespondWithErrorJSON(w, http.StatusBadRequest, "invalid id parameter")
		} else if errors.Is(err, service.ErrAdNotFound) {
			status = "not_found"
			utils.RespondWithErrorJSON(w, http.StatusNotFound, "ad not found")
		} else {
			status = "error"
			h.logger.ErrorLogger.Error("failed to get ad by ID", utils.Err(err))
			utils.RespondWithErrorJSON(w, http.StatusInternalServerError, "internal server error")
		}
		span.RecordError(err)
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, ad)
}

func (h *AdHandler) GetAllAds(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "GetAllAds")
	defer span.End()

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		h.metrics.RequestCount.WithLabelValues("GET", "/ads", status).Inc()
		h.metrics.RequestDuration.WithLabelValues("GET", "/ads", status).Observe(duration)
	}()

	query := r.URL.Query()

	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil || limit <= 0 {
		limit = 10 // Default limit
	}

	page, err := strconv.Atoi(query.Get("page"))
	if err != nil || page <= 0 {
		page = 1 // Default page number
	}

	offset := (page - 1) * limit

	sortBy := query.Get("sortBy")
	if sortBy == "" {
		sortBy = "created_at" // Default sort column
	}

	order := query.Get("order")
	if order == "" {
		order = "ASC" // Default sort order
	}

	span.SetAttributes(
		attribute.Int("ads.limit", limit),
		attribute.Int("ads.offset", offset),
		attribute.String("ads.sort_by", sortBy),
		attribute.String("ads.order", order),
	)

	result, err := h.service.GetAllAds(ctx, limit, offset, sortBy, order)
	if err != nil {
		status = "error"
		h.logger.ErrorLogger.Error("failed to retrieve ads", utils.Err(err))
		span.RecordError(err)
		utils.RespondWithErrorJSON(w, http.StatusInternalServerError, "could not retrieve ads")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, result)
}

func (h *AdHandler) CreateAd(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "CreateAd")
	defer span.End()

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		h.metrics.RequestCount.WithLabelValues("POST", "/ads", status).Inc()
		h.metrics.RequestDuration.WithLabelValues("POST", "/ads", status).Observe(duration)
	}()

	var adReq domain.Ad
	if err := json.NewDecoder(r.Body).Decode(&adReq); err != nil {
		status = "error"
		h.logger.ErrorLogger.Error("Invalid request payload", utils.Err(err))
		span.RecordError(err)
		utils.RespondWithErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	span.SetAttributes(
		attribute.String("ad.title", adReq.Title),
		attribute.Float64("ad.price", adReq.Price),
	)

	createdAd, err := h.service.CreateAd(ctx, &adReq)
	if err != nil {
		status = "error"
		h.logger.ErrorLogger.Error("Could not create ad", utils.Err(err))
		span.RecordError(err)
		utils.RespondWithErrorJSON(w, http.StatusInternalServerError, "Could not create ad")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, createdAd)
}

func (h *AdHandler) UpdateAd(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "UpdateAd")
	defer span.End()

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		h.metrics.RequestCount.WithLabelValues("PUT", "/ads/{id}", status).Inc()
		h.metrics.RequestDuration.WithLabelValues("PUT", "/ads/{id}", status).Observe(duration)
	}()

	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		status = "error"
		utils.RespondWithErrorJSON(w, http.StatusBadRequest, "missing id parameter")
		return
	}

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		status = "error"
		utils.RespondWithErrorJSON(w, http.StatusBadRequest, "invalid id parameter")
		return
	}

	var adRequest domain.Ad
	if err := json.NewDecoder(r.Body).Decode(&adRequest); err != nil {
		status = "error"
		h.logger.ErrorLogger.Error("failed to decode request body", utils.Err(err))
		span.RecordError(err)
		utils.RespondWithErrorJSON(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	adRequest.ID = id

	span.SetAttributes(
		attribute.Int64("ad.id", adRequest.ID),
		attribute.String("ad.title", adRequest.Title),
		attribute.Float64("ad.price", adRequest.Price),
	)

	updatedAd, err := h.service.UpdateAd(ctx, &adRequest)
	if err != nil {
		if errors.Is(err, service.ErrInvalidID) {
			status = "error"
			utils.RespondWithErrorJSON(w, http.StatusBadRequest, "invalid id parameter")
		} else if errors.Is(err, service.ErrAdNotFound) {
			status = "not_found"
			utils.RespondWithErrorJSON(w, http.StatusNotFound, "ad not found")
		} else {
			status = "error"
			h.logger.ErrorLogger.Error("failed to update ad", utils.Err(err))
			span.RecordError(err)
			utils.RespondWithErrorJSON(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, updatedAd)
}

func (h *AdHandler) DeleteAd(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "DeleteAd")
	defer span.End()

	startTime := time.Now()
	status := "success"

	defer func() {
		duration := time.Since(startTime).Seconds()
		h.metrics.RequestCount.WithLabelValues("DELETE", "/ads/{id}", status).Inc()
		h.metrics.RequestDuration.WithLabelValues("DELETE", "/ads/{id}", status).Observe(duration)
	}()

	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		status = "error"
		utils.RespondWithErrorJSON(w, http.StatusBadRequest, "missing id parameter")
		return
	}

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		status = "error"
		utils.RespondWithErrorJSON(w, http.StatusBadRequest, "invalid id parameter")
		return
	}

	err = h.service.DeleteAd(ctx, id)
	if err != nil {
		if errors.Is(err, service.ErrInvalidID) {
			status = "error"
			utils.RespondWithErrorJSON(w, http.StatusBadRequest, "invalid id parameter")
		} else if errors.Is(err, service.ErrAdNotFound) {
			status = "not_found"
			utils.RespondWithErrorJSON(w, http.StatusNotFound, "ad not found")
		} else {
			status = "error"
			h.logger.ErrorLogger.Error("failed to delete ad", utils.Err(err))
			span.RecordError(err)
			utils.RespondWithErrorJSON(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "ad deleted successfully"})
}
