// Package router provides the API Gateway routing configuration.
package router

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// Service registry maps service names to their base URLs.
var serviceRegistry = map[string]string{
	"auth":         getServiceURL("AUTH_SERVICE_URL", "http://localhost:8081"),
	"otp":          getServiceURL("OTP_SERVICE_URL", "http://localhost:8082"),
	"user":         getServiceURL("USER_SERVICE_URL", "http://localhost:8083"),
	"driver":       getServiceURL("DRIVER_SERVICE_URL", "http://localhost:8084"),
	"ride-order":   getServiceURL("RIDE_ORDER_SERVICE_URL", "http://localhost:8085"),
	"food-order":   getServiceURL("FOOD_ORDER_SERVICE_URL", "http://localhost:8086"),
	"cart":         getServiceURL("CART_SERVICE_URL", "http://localhost:8087"),
	"matching":     getServiceURL("MATCHING_SERVICE_URL", "http://localhost:8088"),
	"payment":      getServiceURL("PAYMENT_SERVICE_URL", "http://localhost:8089"),
	"wallet":       getServiceURL("WALLET_SERVICE_URL", "http://localhost:8090"),
	"settlement":   getServiceURL("SETTLEMENT_SERVICE_URL", "http://localhost:8091"),
	"pricing":      getServiceURL("PRICING_SERVICE_URL", "http://localhost:8092"),
	"promo":        getServiceURL("PROMO_SERVICE_URL", "http://localhost:8093"),
	"location":     getServiceURL("LOCATION_SERVICE_URL", "http://localhost:8094"),
	"chat":         getServiceURL("CHAT_SERVICE_URL", "http://localhost:8095"),
	"notification": getServiceURL("NOTIFICATION_SERVICE_URL", "http://localhost:8096"),
	"email":        getServiceURL("EMAIL_SERVICE_URL", "http://localhost:8097"),
	"emergency":    getServiceURL("EMERGENCY_SERVICE_URL", "http://localhost:8098"),
	"merchant":     getServiceURL("MERCHANT_SERVICE_URL", "http://localhost:8099"),
	"menu":         getServiceURL("MENU_SERVICE_URL", "http://localhost:8100"),
	"rating":       getServiceURL("RATING_SERVICE_URL", "http://localhost:8101"),
	"review":       getServiceURL("REVIEW_SERVICE_URL", "http://localhost:8102"),
	"audit-log":    getServiceURL("AUDIT_LOG_SERVICE_URL", "http://localhost:8103"),
}

// NewRouter creates a new chi router with all API routes configured.
func NewRouter() *chi.Mux {
	r := chi.NewRouter()

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.SuccessResponse(w, http.StatusOK, map[string]string{
			"status":  "healthy",
			"service": "api-gateway",
		})
	})

	// Route to microservices
	r.Route("/api/v1", func(r chi.Router) {
		r.Handle("/auth/*", proxyTo("auth"))
		r.Handle("/otp/*", proxyTo("otp"))
		r.Handle("/users/*", proxyTo("user"))
		r.Handle("/drivers/*", proxyTo("driver"))
		r.Handle("/rides/*", proxyTo("ride-order"))
		r.Handle("/foods/*", proxyTo("food-order"))
		r.Handle("/cart/*", proxyTo("cart"))
		r.Handle("/matching/*", proxyTo("matching"))
		r.Handle("/payments/*", proxyTo("payment"))
		r.Handle("/wallet/*", proxyTo("wallet"))
		r.Handle("/settlements/*", proxyTo("settlement"))
		r.Handle("/pricing/*", proxyTo("pricing"))
		r.Handle("/promos/*", proxyTo("promo"))
		r.Handle("/locations/*", proxyTo("location"))
		r.Handle("/chat/*", proxyTo("chat"))
		r.Handle("/notifications/*", proxyTo("notification"))
		r.Handle("/emails/*", proxyTo("email"))
		r.Handle("/emergency/*", proxyTo("emergency"))
		r.Handle("/merchants/*", proxyTo("merchant"))
		r.Handle("/menus/*", proxyTo("menu"))
		r.Handle("/ratings/*", proxyTo("rating"))
		r.Handle("/reviews/*", proxyTo("review"))
		r.Handle("/audit/*", proxyTo("audit-log"))
	})

	return r
}

// proxyTo creates a reverse proxy handler for the given service.
func proxyTo(serviceName string) http.Handler {
	targetURL, exists := serviceRegistry[serviceName]
	if !exists {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			utils.ErrorResponse(w, http.StatusBadGateway, "service not found: "+serviceName)
		})
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			utils.ErrorResponse(w, http.StatusInternalServerError, "invalid service URL")
		})
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	return proxy
}

// getServiceURL reads a service URL from environment or returns default.
func getServiceURL(envKey, defaultURL string) string {
	if url := os.Getenv(envKey); url != "" {
		return url
	}
	return defaultURL
}
