package httpx

import "github.com/gin-gonic/gin"

const RequestIDKey = "wms.request_id"

func RequestID(c *gin.Context) string {
	value, _ := c.Get(RequestIDKey)
	requestID, _ := value.(string)
	return requestID
}
