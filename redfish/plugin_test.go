package redfish

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestBasicAuthValidator tests the BasicAuthValidator middleware.
func TestBasicAuthValidator(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	expectedUsername := "testuser"
	expectedPassword := "testpass"

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Valid credentials",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass")),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid credentials",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("wronguser:wrongpass")),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid auth format - Bearer instead of Basic",
			authHeader:     "Bearer token123",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid base64 encoding",
			authHeader:     "Basic invalid!!!base64",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing colon separator in credentials",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("usernameonly")),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := gin.New()
			router.GET("/test", BasicAuthValidator(expectedUsername, expectedPassword), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestAuthMiddleware tests the AuthMiddleware function.
func TestAuthMiddleware(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	publicEndpoints := map[string]bool{
		"/redfish/v1/":          true,
		"/redfish/v1/$metadata": true,
	}

	mockBasicAuth := func(c *gin.Context) {
		c.Header("X-Auth-Checked", "true")
		c.Next()
	}

	tests := []struct {
		name              string
		path              string
		expectAuthChecked bool
	}{
		{
			name:              "Public endpoint should not require auth",
			path:              "/redfish/v1/",
			expectAuthChecked: false,
		},
		{
			name:              "Protected endpoint should require auth",
			path:              "/redfish/v1/Systems",
			expectAuthChecked: true,
		},
		{
			name:              "Non-redfish endpoint should not require auth",
			path:              "/api/v1/devices",
			expectAuthChecked: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := gin.New()
			router.Use(AuthMiddleware(mockBasicAuth, publicEndpoints))
			router.GET(tt.path, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			router.NoRoute(func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			authChecked := w.Header().Get("X-Auth-Checked")
			if tt.expectAuthChecked {
				assert.Equal(t, "true", authChecked)
			} else {
				assert.Empty(t, authChecked)
			}
		})
	}
}
