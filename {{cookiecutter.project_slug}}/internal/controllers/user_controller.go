package controllers

import (
	"errors"

	"{{ cookiecutter.go_module }}/configs"
	"{{ cookiecutter.go_module }}/internal/common/auth"
	"{{ cookiecutter.go_module }}/internal/common/responses"
	"{{ cookiecutter.go_module }}/internal/core/apierrors"
	"{{ cookiecutter.go_module }}/internal/core/services"
	"{{ cookiecutter.go_module }}/internal/core/vo"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserController struct {
	userService *services.UserService
	cfg         *configs.Config
}

func NewUserController(userService *services.UserService, cfg *configs.Config) *UserController {
	return &UserController{
		userService: userService,
		cfg:         cfg,
	}
}

func (u *UserController) CreateUser(c *gin.Context) {
	req := vo.CreateUserRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.AbortWithBindError(c, err)
		return
	}

	user, err := u.userService.CreateUser(&req)
	if err != nil {
		abortOnError(c, err)
		return
	}
	responses.Created(c, vo.NewUserVO(user))
}

func (u *UserController) Login(c *gin.Context) {
	req := vo.LoginRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.AbortWithBindError(c, err)
		return
	}

	user, err := u.userService.Login(req.Email, req.Password)
	if err != nil {
		abortOnError(c, err)
		return
	}

	token, err := auth.GenerateJwtToken(u.cfg.JwtSecret, user.ID, auth.JwtOptions{
		Issuer:     u.cfg.JwtIssuer,
		Audience:   u.cfg.JwtAudience,
		Expiration: u.cfg.JwtExpiration,
		Leeway:     u.cfg.JwtLeeway,
	})
	if err != nil {
		abortOnError(c, err)
		return
	}

	responses.Ok(c, vo.TokenVO{
		Token:     token,
		ExpiresIn: int(u.cfg.JwtExpiration.Seconds()),
	})
}

// Me returns the user behind the token of the current request.
func (u *UserController) Me(c *gin.Context) {
	raw, _ := c.Get("userID")
	userID, ok := raw.(string)
	if !ok {
		responses.AbortWithAPIError(c, apierrors.ErrAccessTokenInvalid)
		return
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		responses.AbortWithAPIError(c, apierrors.ErrAccessTokenInvalid)
		return
	}

	user, err := u.userService.GetByID(id)
	if err != nil {
		abortOnError(c, err)
		return
	}
	responses.Ok(c, vo.NewUserVO(user))
}

func abortOnError(c *gin.Context, err error) {
	var apiErr *apierrors.APIError
	if errors.As(err, &apiErr) {
		responses.AbortWithAPIError(c, apiErr)
		return
	}
	responses.AbortWithError(c, err)
}
