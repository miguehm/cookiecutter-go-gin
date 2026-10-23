package controllers

import (
	"{{ cookiecutter.go_module }}/internal/common/responses"
	"{{ cookiecutter.go_module }}/internal/core/services"
	"{{ cookiecutter.go_module }}/internal/core/vo"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService *services.UserService
}

func NewUserController(userService *services.UserService) *UserController {
	return &UserController{
		userService: userService,
	}
}

func (u UserController) SignUp(c *gin.Context) {
	data := vo.UserVO{}
	responses.Ok(c, data)
}
