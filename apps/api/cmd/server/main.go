package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/lamkaka/invisible-ms/internal/activity"
	"github.com/lamkaka/invisible-ms/internal/company"
	"github.com/lamkaka/invisible-ms/internal/dashboard"
	"github.com/lamkaka/invisible-ms/internal/shared"
	"github.com/lamkaka/invisible-ms/internal/staff"
)

func main() {
	ctx := context.Background()

	// Load config
	cfg, err := shared.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create Spanner client
	spannerClient, err := shared.NewSpannerClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create Spanner client: %v", err)
	}
	defer spannerClient.Close()

	// Initialize repositories
	companyRepo := company.NewSpannerCompanyRepository(spannerClient)
	companyActionTypeRepo := company.NewSpannerCompanyActionTypeRepository(spannerClient)
	staffRepo := staff.NewSpannerStaffRepository(spannerClient)
	activityRepo := activity.NewSpannerActivityRepository(spannerClient)
	dashboardRepo := dashboard.NewSpannerDashboardRepository(spannerClient)

	// Initialize services
	companyService := company.NewCompanyService(companyRepo, companyActionTypeRepo)
	staffService := staff.NewStaffService(staffRepo, companyService)
	activityWebhookService := activity.NewWebhookService(activityRepo, staffService, companyService)
	activitySessionService := activity.NewSessionService(activityRepo, companyService)
	dashboardService := dashboard.NewDashboardService(dashboardRepo)

	// Initialize controllers
	companyController := company.NewCompanyController(companyService)
	staffController := staff.NewStaffController(staffService)
	activityController := activity.NewActivityController(activityWebhookService, activitySessionService, cfg.WebhookSecret)
	dashboardAPIController := dashboard.NewDashboardAPIController(dashboardService)

	dashboardWebController, err := dashboard.NewDashboardWebController(dashboardService, cfg.TemplatesPath)
	if err != nil {
		log.Fatalf("Failed to create dashboard web controller: %v", err)
	}

	// Setup router
	router := mux.NewRouter()
	router.Use(shared.LoggingMiddleware)
	router.Use(shared.CORSMiddleware(cfg.CORSAllowedOrigins))

	// Register routes
	companyController.RegisterRoutes(router)
	staffController.RegisterRoutes(router)
	activityController.RegisterRoutes(router)
	dashboardAPIController.RegisterRoutes(router)
	dashboardWebController.RegisterRoutes(router)

	// Serve static files
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticPath))))

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("Server starting on port %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
