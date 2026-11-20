package response

import "github.com/gin-gonic/gin"

// Body adalah format respons JSON standar seluruh endpoint.
type Body struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success membungkus respons sukses (HTTP 2xx).
func Success(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, Body{Status: "success", Message: message, Data: data})
}

// Fail membungkus kesalahan klien (HTTP 4xx).
func Fail(c *gin.Context, code int, message string) {
	c.JSON(code, Body{Status: "fail", Message: message})
}

// Error membungkus kesalahan server (HTTP 5xx).
func Error(c *gin.Context, code int, message string) {
	c.JSON(code, Body{Status: "error", Message: message})
}

// AbortFail sama dengan Fail tetapi menghentikan rantai middleware.
func AbortFail(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, Body{Status: "fail", Message: message})
}

// AbortError sama dengan Error tetapi menghentikan rantai middleware.
func AbortError(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, Body{Status: "error", Message: message})
}
