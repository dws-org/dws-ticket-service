package middlewares

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/oskargbc/dws-ticket-service/configs"
	log "github.com/sirupsen/logrus"
)

type KeycloakCerts struct {
	Keys []struct {
		Kid string   `json:"kid"`
		Kty string   `json:"kty"`
		Alg string   `json:"alg"`
		Use string   `json:"use"`
		N   string   `json:"n"`
		E   string   `json:"e"`
		X5c []string `json:"x5c"`
	} `json:"keys"`
}

var publicKeys = make(map[string]*rsa.PublicKey)

func KeycloakAuthMiddleware(cfg *configs.Config) gin.HandlerFunc {
	// Load public keys from Keycloak on startup
	if err := loadKeycloakPublicKeys(cfg); err != nil {
		log.WithError(err).Warn("Failed to load Keycloak public keys")
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			kid, ok := token.Header["kid"].(string)
			if !ok {
				return nil, fmt.Errorf("kid not found in token header")
			}

			publicKey, exists := publicKeys[kid]
			if !exists {
				// Try to reload keys if not found
				if err := loadKeycloakPublicKeys(cfg); err != nil {
					return nil, fmt.Errorf("failed to reload keys: %w", err)
				}
				publicKey, exists = publicKeys[kid]
				if !exists {
					return nil, fmt.Errorf("public key not found for kid: %s", kid)
				}
			}

			return publicKey, nil
		})

		if err != nil || !token.Valid {
			log.WithError(err).Warn("Invalid JWT token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Extract user ID from subject claim
		userID, ok := claims["sub"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			c.Abort()
			return
		}

		// Store user ID in context
		c.Set("user_id", userID)
		c.Next()
	}
}

func loadKeycloakPublicKeys(cfg *configs.Config) error {
	certsURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", cfg.Keycloak.URL, cfg.Keycloak.Realm)
	
	resp, err := http.Get(certsURL)
	if err != nil {
		return fmt.Errorf("failed to fetch certs: %w", err)
	}
	defer resp.Body.Close()

	var certs KeycloakCerts
	if err := json.NewDecoder(resp.Body).Decode(&certs); err != nil {
		return fmt.Errorf("failed to decode certs: %w", err)
	}

	for _, key := range certs.Keys {
		if key.Kty == "RSA" {
			publicKey, err := parseRSAPublicKey(key.N, key.E)
			if err != nil {
				log.WithError(err).WithField("kid", key.Kid).Warn("Failed to parse public key")
				continue
			}
			publicKeys[key.Kid] = publicKey
		}
	}

	log.WithField("count", len(publicKeys)).Info("Loaded Keycloak public keys")
	return nil
}

func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode n: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
