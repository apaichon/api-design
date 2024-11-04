package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"golang.org/x/time/rate"
	"github.com/rs/cors" 
)

// Response wrapper
type Response struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Context keys
type contextKey string

const (
	UserContextKey  contextKey = "user"
	TraceContextKey contextKey = "trace"
)

// Middleware type definition
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Chain multiple middleware
func Chain(middlewares ...Middleware) Middleware {
	return func(final http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			last := final
			for i := len(middlewares) - 1; i >= 0; i-- {
				last = middlewares[i](last)
			}
			last(w, r)
		}
	}
}

// CORS middleware
func CORS() Middleware {
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Allow all origins, adjust as needed
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: true,
	})

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			corsHandler.ServeHTTP(w, r, next) // Use the CORS handler
		}
	}
}

// Validation middleware
func WithValidation(validate *validator.Validate) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, span := otel.Tracer("validation").Start(r.Context(), "validate-request")
			defer span.End()

			var payload interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				respondWithError(w, http.StatusBadRequest, "Invalid request payload")
				return
			}

			if err := validate.Struct(payload); err != nil {
				validationErrors := err.(validator.ValidationErrors)
				respondWithError(w, http.StatusBadRequest, validationErrors.Error())
				return
			}

			r = r.WithContext(ctx)
			next(w, r)
		}
	}
}

// Authentication middleware
func WithAuthentication(secretKey string) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, span := otel.Tracer("auth").Start(r.Context(), "authenticate")
			defer span.End()

			tokenString := r.Header.Get("Authorization")
			if tokenString == "" {
				respondWithError(w, http.StatusUnauthorized, "Missing authorization token")
				return
			}

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secretKey), nil
			})

			if err != nil {
				respondWithError(w, http.StatusUnauthorized, "Invalid token")
				return
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				ctx = context.WithValue(ctx, UserContextKey, claims)
				r = r.WithContext(ctx)
				next(w, r)
			} else {
				respondWithError(w, http.StatusUnauthorized, "Invalid token claims")
			}
		}
	}
}

// Authorization middleware
func WithAuthorization(requiredRole string) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, span := otel.Tracer("auth").Start(r.Context(), "authorize")
			defer span.End()

			claims, ok := r.Context().Value(UserContextKey).(jwt.MapClaims)
			if !ok {
				respondWithError(w, http.StatusForbidden, "No user context found")
				return
			}

			userRole, ok := claims["role"].(string)
			if !ok || userRole != requiredRole {
				respondWithError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			r = r.WithContext(ctx)
			next(w, r)
		}
	}
}

// Rate limiting middleware
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rate.Every(time.Second), 10) // 10 requests per second
		rl.limiters[key] = limiter
	}

	return limiter
}

func WithRateLimit(rl *RateLimiter) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, span := otel.Tracer("ratelimit").Start(r.Context(), "check-rate-limit")
			defer span.End()

			key := r.RemoteAddr // Or use user ID from context
			limiter := rl.getLimiter(key)

			if !limiter.Allow() {
				respondWithError(w, http.StatusTooManyRequests, "Rate limit exceeded")
				return
			}

			r = r.WithContext(ctx)
			next(w, r)
		}
	}
}

// Circuit Breaker middleware
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	failures         int
	lastFailure      time.Time
	mu               sync.RWMutex
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: threshold,
		resetTimeout:     timeout,
	}
}

func (cb *CircuitBreaker) isOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.failures >= cb.failureThreshold {
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.failures = 0
			return false
		}
		return true
	}
	return false
}

func WithCircuitBreaker(cb *CircuitBreaker) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, span := otel.Tracer("circuitbreaker").Start(r.Context(), "check-circuit")
			defer span.End()

			if cb.isOpen() {
				respondWithError(w, http.StatusServiceUnavailable, "Service temporarily unavailable")
				return
			}

			r = r.WithContext(ctx)
			next(w, r)
		}
	}
}

// Prometheus metrics
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

// Monitoring middleware
func WithMonitoring() Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, span := otel.Tracer("monitoring").Start(r.Context(), "monitor-request")
			defer span.End()

			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}

			r = r.WithContext(ctx)
			next(sw, r)

			duration := time.Since(start).Seconds()

			httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", sw.status)).Inc()
			httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		}
	}
}

// Logging middleware
func WithLogging() Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, span := otel.Tracer("logging").Start(r.Context(), "log-request")
			defer span.End()

			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}

			r = r.WithContext(ctx)
			next(sw, r)

			duration := time.Since(start)

			log.Printf(
				"Method: %s | Path: %s | Status: %d | Duration: %v | IP: %s",
				r.Method,
				r.URL.Path,
				sw.status,
				duration,
				r.RemoteAddr,
			)
		}
	}
}

// Helper types and functions
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, Response{
		Status:  code,
		Message: "error",
		Error:   message,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// Example struct with validation
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=20"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"required,gte=18"`
}
