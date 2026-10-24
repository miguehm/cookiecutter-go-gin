package middlewares

import (
	"strings"

	"github.com/gin-gonic/gin"
	"{{ cookiecutter.go_module }}/configs"
	"{{ cookiecutter.go_module }}/internal/common/auth"
	"{{ cookiecutter.go_module }}/internal/common/responses"
	"{{ cookiecutter.go_module }}/internal/core/apierrors"
)

func JwtAuth() gin.HandlerFunc {
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
			Issuer:   configs.Env.JwtIssuer,
			Audience: configs.Env.JwtAudience,
		}
		claims, err := auth.VerifyJwtToken(configs.Env.JwtSecret, tokenString, opts)
		if err != nil {
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
