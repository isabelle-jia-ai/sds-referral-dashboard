package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var tokenStore = struct {
	sync.RWMutex
	tokens map[string]time.Time
}{tokens: make(map[string]time.Time)}

const tokenTTL = 24 * time.Hour

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func storeToken(token string) {
	tokenStore.Lock()
	defer tokenStore.Unlock()

	// Evict expired tokens opportunistically
	now := time.Now()
	for k, v := range tokenStore.tokens {
		if now.Sub(v) > tokenTTL {
			delete(tokenStore.tokens, k)
		}
	}
	tokenStore.tokens[token] = now
}

func validateToken(token string) bool {
	tokenStore.RLock()
	defer tokenStore.RUnlock()
	created, ok := tokenStore.tokens[token]
	return ok && time.Since(created) < tokenTTL
}

func handleAuthVerify() gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := os.Getenv("PROTECTED_TAB_PASSWORD")
		if expected == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Protected tab not configured"})
			return
		}

		var req struct {
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		if subtle.ConstantTimeCompare([]byte(req.Password), []byte(expected)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect password"})
			return
		}

		token, err := generateToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}
		storeToken(token)

		c.JSON(http.StatusOK, gin.H{"success": true, "token": token})
	}
}

func handleAuthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Auth-Token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"valid": false})
			return
		}
		c.JSON(http.StatusOK, gin.H{"valid": validateToken(token)})
	}
}
