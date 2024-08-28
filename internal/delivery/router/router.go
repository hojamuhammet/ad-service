package router

import (
	"ad-service/internal/delivery/handler"
	"ad-service/internal/infrastructure/metrics"
	"ad-service/internal/service"
	"ad-service/pkg/logger"

	"github.com/go-chi/chi/v5"
)

func SetupAdRoutes(adRouter *chi.Mux, adService service.AdService, loggers *logger.Loggers, metrics *metrics.HandlerMetrics) {
	adHandler := handler.NewAdHandler(adService, loggers, metrics)

	adRouter.Get("/ads", adHandler.GetAllAds)
	adRouter.Get("/ads/{id}", adHandler.GetAdByID)
	adRouter.Post("/ads", adHandler.CreateAd)
	adRouter.Put("/ads/{id}", adHandler.UpdateAd)
	adRouter.Delete("/ads/{id}", adHandler.DeleteAd)
}
