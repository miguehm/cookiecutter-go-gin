package middlewares

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"log/slog"

	"{{ cookiecutter.go_module }}/internal/core/apierrors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func UcFirst(str string) string {
	for i, v := range str {
		return string(unicode.ToUpper(v)) + str[i+1:]
	}
	return ""
}

func LcFirst(str string) string {
	return strings.ToLower(str)
}

func Split(src string) string {
	// don't split invalid utf8
	if !utf8.ValidString(src) {
		return src
	}
	var entries []string
	var runes [][]rune
	lastClass := 0
	class := 0
	// split into fields based on class of unicode character
	for _, r := range src {
		switch true {
		case unicode.IsLower(r):
			class = 1
		case unicode.IsUpper(r):
			class = 2
		case unicode.IsDigit(r):
			class = 3
		default:
			class = 4
		}
		if class == lastClass {
			runes[len(runes)-1] = append(runes[len(runes)-1], r)
		} else {
			runes = append(runes, []rune{r})
		}
		lastClass = class
	}

	for i := 0; i < len(runes)-1; i++ {
		if unicode.IsUpper(runes[i][0]) && unicode.IsLower(runes[i+1][0]) {
			runes[i+1] = append([]rune{runes[i][len(runes[i])-1]}, runes[i+1]...)
			runes[i] = runes[i][:len(runes[i])-1]
		}
	}
	// construct []string from results
	for _, s := range runes {
		if len(s) > 0 {
			entries = append(entries, string(s))
		}
	}

	for index, word := range entries {
		if index == 0 {
			entries[index] = UcFirst(word)
		} else {
			entries[index] = LcFirst(word)
		}
	}
	justString := strings.Join(entries, " ")
	return justString
}

func ValidationErrorToText(e validator.FieldError) string {
	word := Split(e.Field())

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", word)
	case "max":
		return fmt.Sprintf("%s cannot be longer than %s", word, e.Param())
	case "min":
		return fmt.Sprintf("%s must be longer than %s", word, e.Param())
	case "email":
		return "Invalid email format"
	case "len":
		return fmt.Sprintf("%s must be %s characters long", word, e.Param())
	}
	return fmt.Sprintf("%s is not valid", word)
}

func ErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				var errMessage string
				switch e.Type {
				case gin.ErrorTypePublic:
					var apiError *apierrors.APIError
					if errors.As(e.Err, &apiError) {
						if !c.Writer.Written() {
							c.JSON(apiError.Status, gin.H{"code": apiError.ErrorCode, "message": apiError.Message})
						}
						if apiError.ErrorCode == "ErrSessionInvalid" {
							logger.Info(e.Error())
						} else {
							errMessage = e.Error()
						}
					} else {
						errMessage = e.Error()
					}
				case gin.ErrorTypeBind:
					status := http.StatusBadRequest
					if c.Writer.Status() != http.StatusOK {
						status = c.Writer.Status()
					}

					var errs validator.ValidationErrors
					if errors.As(e.Err, &errs) {
						list := make(map[string]string)
						for _, err := range errs {
							list[strings.ToLower(err.Field())] = ValidationErrorToText(err)
						}

						c.JSON(status, gin.H{"code": "ErrBadRequest", "message": "Invalid request", "details": list})
					} else {
						c.JSON(status, gin.H{"code": "ErrBadRequest", "message": "Invalid request"})
					}
					errMessage = e.Error()

				default:
					errMessage = "unexpected error: " + e.Error()
				}
				if errMessage != "" {
					reqID, _ := c.Get("request_id")
					if reqID == nil {
						reqID = c.Writer.Header().Get("X-Request-ID")
					}
					logger.Error("error when call api "+c.Request.URL.Path+": "+errMessage, "request_id", reqID)
				}

			}
			if !c.Writer.Written() {
				c.JSON(http.StatusInternalServerError, gin.H{"code": "ErrUnexpected", "message": "whoops! something went wrong"})
			}
		}
	}
}
