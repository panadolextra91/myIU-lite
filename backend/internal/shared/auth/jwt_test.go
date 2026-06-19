package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	sharedauth "github.com/panadolextra91/myiu-lite/backend/internal/shared/auth"
	"github.com/stretchr/testify/require"
)

func TestJWT(t *testing.T) {
	secret := []byte("test-secret")
	userID := int64(42)
	role := "student"

	t.Run("round trip", func(t *testing.T) {
		token, err := sharedauth.Mint(secret, userID, role, "access", time.Hour)
		require.NoError(t, err)

		claims, err := sharedauth.Parse(secret, token)
		require.NoError(t, err)
		require.Equal(t, "42", claims.Subject)
		require.Equal(t, "student", claims.Role)
	})

	t.Run("expired token", func(t *testing.T) {
		token, err := sharedauth.Mint(secret, userID, role, "access", -time.Hour)
		require.NoError(t, err)

		_, err = sharedauth.Parse(secret, token)
		require.Error(t, err)
		require.ErrorIs(t, err, jwt.ErrTokenExpired)
	})

	t.Run("wrong method", func(t *testing.T) {
		claims := sharedauth.Claims{
			Role: role,
			Type: "access",
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: "42",
			},
		}
		// none alg
		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		str, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		_, err = sharedauth.Parse(secret, str)
		require.Error(t, err)
	})
}
