package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	authServiceURL string
}

func NewAuthMiddleware(authServiceURL string) *AuthMiddleware {
	return &AuthMiddleware{authServiceURL: authServiceURL}
}

func (m *AuthMiddleware) ValidateToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		// Validate token with auth service
		valid, err := m.validateWithAuthService(token)
		if err != nil || !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) validateWithAuthService(token string) (bool, error) {
	url := fmt.Sprintf("%s/auth/validate", m.authServiceURL)

	reqBody, _ := json.Marshal(map[string]string{"token": token})

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var result map[string]bool
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &result)
	if err != nil {
		return false, err
	}

	return result["valid"], nil
}

func extractToken(c *gin.Context) string {
	bearerToken := c.GetHeader("Authorization")
	parts := strings.Split(bearerToken, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1]
	}
	return ""
}
