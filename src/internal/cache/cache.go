package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"crypto/sha256"
	"io"
	"bytes"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/vmihailenco/msgpack/v5"
)

// CacheConfig holds configuration for caching
type CacheConfig struct {
	TTL           time.Duration
	KeyPrefix     string
	IgnoreParams  []string
	ExcludePaths  []string
	Strategy      CacheStrategy
	InvalidateOn  []string // HTTP methods that invalidate cache
}

// CacheStrategy defines how caching behaves
type CacheStrategy interface {
	GetKey(r *http.Request) string
	ShouldCache(r *http.Request) bool
	Store(ctx context.Context, key string, value interface{}) error
	Fetch(ctx context.Context, key string) (interface{}, error)
	Invalidate(ctx context.Context, key string) error
}

// RedisCache implements CacheStrategy
type RedisCache struct {
	client     *redis.Client
	config     *CacheConfig
	serializer Serializer
}

// Serializer interface for flexibility in serialization
type Serializer interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

// MsgPackSerializer implements Serializer using MessagePack
type MsgPackSerializer struct{}

func (s *MsgPackSerializer) Marshal(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (s *MsgPackSerializer) Unmarshal(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(client *redis.Client, config *CacheConfig) *RedisCache {
	if config.TTL == 0 {
		config.TTL = 15 * time.Minute
	}
	return &RedisCache{
		client:     client,
		config:     config,
		serializer: &MsgPackSerializer{},
	}
}

// CacheResponseWriter captures the response for caching
type CacheResponseWriter struct {
	http.ResponseWriter
	body   *bytes.Buffer
	status int
}

func NewCacheResponseWriter(w http.ResponseWriter) *CacheResponseWriter {
	return &CacheResponseWriter{
		ResponseWriter: w,
		body:          &bytes.Buffer{},
	}
}

func (w *CacheResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *CacheResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// CachedResponse represents the cached response data
type CachedResponse struct {
	Status  int
	Headers http.Header
	Body    []byte
}

// GetKey generates a cache key from the request
func (rc *RedisCache) GetKey(r *http.Request) string {
	// Create a copy of the request URL query
	query := r.URL.Query()
	
	// Remove ignored parameters
	for _, param := range rc.config.IgnoreParams {
		query.Del(param)
	}
	
	// Create a unique key based on method, path, and remaining query parameters
	key := fmt.Sprintf("%s:%s:%s:%s",
		rc.config.KeyPrefix,
		r.Method,
		r.URL.Path,
		query.Encode(),
	)
	
	// Hash the key if it's too long
	if len(key) > 200 {
		hasher := sha256.New()
		hasher.Write([]byte(key))
		key = fmt.Sprintf("%s:hash:%x", rc.config.KeyPrefix, hasher.Sum(nil))
	}
	
	return key
}

// ShouldCache determines if the request should be cached
func (rc *RedisCache) ShouldCache(r *http.Request) bool {
	// Don't cache non-GET requests
	if r.Method != http.MethodGet {
		return false
	}
	
	// Check if path is excluded
	for _, path := range rc.config.ExcludePaths {
		if strings.HasPrefix(r.URL.Path, path) {
			return false
		}
	}
	
	// Don't cache requests with cache-control: no-cache
	if r.Header.Get("Cache-Control") == "no-cache" {
		return false
	}
	
	return true
}

// Store caches the response
func (rc *RedisCache) Store(ctx context.Context, key string, value interface{}) error {
	data, err := rc.serializer.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}
	
	return rc.client.Set(ctx, key, data, rc.config.TTL).Err()
}

// Fetch retrieves cached response
func (rc *RedisCache) Fetch(ctx context.Context, key string) (interface{}, error) {
	data, err := rc.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	
	var response CachedResponse
	err = rc.serializer.Unmarshal(data, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache data: %w", err)
	}
	
	return &response, nil
}

// Invalidate removes cached items
func (rc *RedisCache) Invalidate(ctx context.Context, key string) error {
	return rc.client.Del(ctx, key).Err()
}

// Cache pattern implementations
type CachePattern interface {
	Apply(next http.HandlerFunc) http.HandlerFunc
}

// Cache-Aside Pattern
type CacheAside struct {
	cache *RedisCache
}

func (ca *CacheAside) Apply(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ca.cache.ShouldCache(r) {
			next(w, r)
			return
		}
		
		key := ca.cache.GetKey(r)
		
		// Try to get from cache
		cached, err := ca.cache.Fetch(r.Context(), key)
		if err == nil && cached != nil {
			response := cached.(*CachedResponse)
			for k, v := range response.Headers {
				w.Header()[k] = v
			}
			w.WriteHeader(response.Status)
			w.Write(response.Body)
			return
		}
		
		// Cache miss - capture and store response
		cw := NewCacheResponseWriter(w)
		next(cw, r)
		
		response := &CachedResponse{
			Status:  cw.status,
			Headers: w.Header(),
			Body:    cw.body.Bytes(),
		}
		
		if err := ca.cache.Store(r.Context(), key, response); err != nil {
			// Log cache storage error but don't fail the request
			fmt.Printf("Cache storage error: %v\n", err)
		}
	}
}

// Write-Through Pattern
type WriteThrough struct {
	cache *RedisCache
}

func (wt *WriteThrough) Apply(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			// On write operations, update cache immediately
			key := wt.cache.GetKey(r)
			cw := NewCacheResponseWriter(w)
			next(cw, r)
			
			if cw.status == http.StatusOK {
				response := &CachedResponse{
					Status:  cw.status,
					Headers: w.Header(),
					Body:    cw.body.Bytes(),
				}
				wt.cache.Store(r.Context(), key, response)
			}
			return
		}
		
		// For read operations, use cache-aside pattern
		(&CacheAside{cache: wt.cache}).Apply(next)(w, r)
	}
}

// Cache middleware factory
func WithCache(client *redis.Client, config *CacheConfig) Middleware {
	cache := NewRedisCache(client, config)
	
	return func(next http.HandlerFunc) http.HandlerFunc {
		var pattern CachePattern
		
		switch config.Strategy.(type) {
		case *WriteThrough:
			pattern = &WriteThrough{cache: cache}
		default:
			pattern = &CacheAside{cache: cache}
		}
		
		return pattern.Apply(next)
	}
}
