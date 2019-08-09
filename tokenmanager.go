package main

import (
	"context"
	"github.com/kurin/blazer/b2"
	cache "github.com/patrickmn/go-cache"
	"sync"
	"time"
)

// TokenManager - ...
type TokenManager struct {
	cache  *cache.Cache
	mutex  *sync.Mutex
	bucket *b2.Bucket
	ctx    *context.Context
}

// NewTokenManager - ...
func NewTokenManager(ctx *context.Context, bucket *b2.Bucket) *TokenManager {
	c := cache.New(
		time.Hour,      // TTL
		30*time.Minute, // cleanup interval
	)

	return &TokenManager{
		mutex:  &sync.Mutex{},
		bucket: bucket,
		cache:  c,
		ctx:    ctx,
	}
}

// GetToken - Get a backblaze B2 token for
func (m TokenManager) GetToken(prefix string) (string, error) {
	val, found := m.cache.Get(prefix)
	if found {
		return val.(string), nil
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Re-attempt cache fetching, someone else holding the lock might have fetched
	val, found = m.cache.Get(prefix)
	if found {
		return val.(string), nil
	}

	// Token validity is set to 2 hours so we will at worst (near expiration)
	// have one hour worth of validity
	tokenValidity := 2 * time.Hour
	token, err := m.bucket.AuthToken(*m.ctx, prefix+"/", tokenValidity)
	if err != nil {
		return "", err
	}
	m.cache.Set(prefix, token, cache.DefaultExpiration)

	return token, nil
}
