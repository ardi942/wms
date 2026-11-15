package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type envelope struct {
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, envelope{Data: data})
}

func OKWithMeta(c *gin.Context, data any, meta any) {
	c.JSON(http.StatusOK, gin.H{"data": data, "meta": meta})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, envelope{Data: data})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func List(c *gin.Context, data any, meta Meta) {
	c.JSON(http.StatusOK, envelope{Data: data, Meta: &meta})
}

func WriteError(c *gin.Context, err error) {
	ae := AsAppError(err)
	c.AbortWithStatusJSON(ae.HTTPStatus, errorEnvelope{Error: errorBody{
		Code:    ae.Code,
		Message: ae.Message,
		Details: ae.Details,
	}})
}

func WriteRawError(c *gin.Context, status int, code, message string, details any) {
	c.AbortWithStatusJSON(status, errorEnvelope{Error: errorBody{Code: code, Message: message, Details: details}})
}
