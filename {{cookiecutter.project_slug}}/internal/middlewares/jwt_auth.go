package middlewares

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"{{ cookiecutter.go_module }}/configs"
	"{{ cookiecutter.go_module }}/internal/common/auth"
	"{{ cookiecutter.go_module }}/internal/common/responses"
	"{{ cookiecutter.go_module }}/internal/core/apierrors"
)

// JwtAuth validates the Bearer token of the request and exposes the subject
// of the token as "userID" in the context.
func JwtAuth(cfg *configs.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader("Authorization")

		if authorizationHeader == "" {
			responses.AbortWithAPIError(ctx, apierrors.ErrAccessTokenInvalid)
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
			responses.AbortWithAPIError(ctx, apierrors.ErrAccessTokenInvalid)
			return
		}
		tokenString := fields[1]

		opts := auth.JwtOptions{
			Issuer:   cfg.JwtIssuer,
			Audience: cfg.JwtAudience,
			Leeway:   cfg.JwtLeeway,
		}
		claims, err := auth.VerifyJwtToken(cfg.JwtSecret, tokenString, opts)
		if err != nil || claims.UserID == uuid.Nil {
			responses.AbortWithAPIError(ctx, apierrors.ErrAccessTokenInvalid)
			return
		}

		ctx.Set("userID", claims.UserID.String())
		if claims.ID != "" {
			ctx.Set("jti", claims.ID)
		}

		ctx.Next()
	}
}
