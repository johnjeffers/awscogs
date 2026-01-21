package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/johnjeffers/awscogs/backend/internal/api/handlers"
	"github.com/johnjeffers/awscogs/backend/internal/aws"
	"github.com/johnjeffers/awscogs/backend/internal/config"
)

// NewRouter creates and configures the HTTP router
func NewRouter(cfg *config.Config, discovery *aws.Discovery) *chi.Mux {
	r := chi.NewRouter()

	// Base middleware (applied to all routes)
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

	// Health check endpoints (without logging)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Handlers
	costsHandler := handlers.NewCostsHandler(cfg, discovery)
	configHandler := handlers.NewConfigHandler(cfg, discovery)

	// Routes (with logging)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Logger)

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
		r.Get("/costs/nat", costsHandler.GetNATGatewayCosts)
		r.Get("/costs/eip", costsHandler.GetElasticIPCosts)
		r.Get("/costs/secrets", costsHandler.GetSecretsCosts)
		r.Get("/costs/publicipv4", costsHandler.GetPublicIPv4Costs)
	})

	// Serve embedded frontend for all other routes
	r.Handle("/*", NewSPAHandler())

	return r
}
