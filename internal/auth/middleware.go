package auth

import (
	"slices"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func Middleware(secret string) func(roles []string) echo.MiddlewareFunc {
	return func(roles []string) echo.MiddlewareFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				header := c.Request().Header.Get("Authorization")
				headerToken, ok := strings.CutPrefix(header, "Bearer ")
				if !ok {
					return echo.NewHTTPError(401, "Invalid token")
				}

				claims := &CustomClaims{}
				token, err := jwt.ParseWithClaims(
					headerToken,
					claims,
					func(t *jwt.Token) (any, error) { return []byte(secret), nil },
					jwt.WithValidMethods([]string{"HS256"}),
				)

				if err != nil || !token.Valid {
					return echo.NewHTTPError(401, "Invalid token")
				}

				userID, err := strconv.ParseUint(claims.UserID, 10, 64)
				if err != nil {
					return echo.NewHTTPError(401, "Invalid token")
				}

				if !slices.Contains(roles, claims.Role) {
					return echo.NewHTTPError(403, "No enough permissions")
				}

				c.Set("userID", userID)

				return next(c)
			}
		}
	}
}
