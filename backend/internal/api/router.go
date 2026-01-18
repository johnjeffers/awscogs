package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/johnjeffers/infra-utilities/awscogs/backend/internal/api/handlers"
	"github.com/johnjeffers/infra-utilities/awscogs/backend/internal/aws"
	"github.com/johnjeffers/infra-utilities/awscogs/backend/internal/config"
)

// NewRouter creates and configures the HTTP router
func NewRouter(cfg *config.Config, discovery *aws.Discovery) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Handlers
	healthHandler := handlers.NewHealthHandler()
	costsHandler := handlers.NewCostsHandler(cfg, discovery)
	configHandler := handlers.NewConfigHandler(cfg, discovery)

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		// Health
		r.Get("/health", healthHandler.ServeHTTP)

		// Configuration
		r.Get("/config", configHandler.GetConfig)

		// Costs
		r.Get("/costs", costsHandler.GetCosts)
		r.Get("/costs/accounts", costsHandler.GetAccountCosts)
		r.Get("/costs/regions", costsHandler.GetRegionCosts)
		r.Get("/costs/ec2", costsHandler.GetEC2Costs)
		r.Get("/costs/ebs", costsHandler.GetEBSCosts)
		r.Get("/costs/ecs", costsHandler.GetECSCosts)
		r.Get("/costs/rds", costsHandler.GetRDSCosts)
		r.Get("/costs/eks", costsHandler.GetEKSCosts)
		r.Get("/costs/elb", costsHandler.GetELBCosts)
	})

	return r
}
