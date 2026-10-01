package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// errorBody is the JSON body of every error response.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var httpStatus = map[codes.Code]int{
	codes.InvalidArgument:    http.StatusBadRequest,
	codes.Unauthenticated:    http.StatusUnauthorized,
	codes.PermissionDenied:   http.StatusForbidden,
	codes.NotFound:           http.StatusNotFound,
	codes.AlreadyExists:      http.StatusConflict,
	codes.FailedPrecondition: http.StatusConflict,
	codes.ResourceExhausted:  http.StatusTooManyRequests,
	codes.Unavailable:        http.StatusServiceUnavailable,
	codes.DeadlineExceeded:   http.StatusGatewayTimeout,
}

// grpcError writes the HTTP equivalent of a gRPC error. Messages of internal
// or unknown errors are not forwarded to the client.
func (h *Handler) grpcError(c *gin.Context, err error) {
	st := status.Convert(err)
	code, ok := httpStatus[st.Code()]
	msg := st.Message()
	if !ok {
		code = http.StatusInternalServerError
		msg = "internal error"
		h.log.ErrorContext(c.Request.Context(), "backend call failed",
			"path", c.FullPath(), "code", st.Code().String(), "error", st.Message(),
			"request_id", c.GetString("request_id"))
	}
	c.AbortWithStatusJSON(code, errorBody{Error: errorDetail{Code: st.Code().String(), Message: msg}})
}

func badRequest(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, errorBody{Error: errorDetail{Code: "INVALID_ARGUMENT", Message: msg}})
}

func notFound(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusNotFound, errorBody{Error: errorDetail{Code: "NOT_FOUND", Message: msg}})
}

// bindJSON decodes and validates the body, writing a 400 on failure.
func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) && len(verrs) > 0 {
			fe := verrs[0]
			badRequest(c, "field '"+fe.Field()+"' failed validation: "+fe.Tag())
			return false
		}
		badRequest(c, "malformed JSON body")
		return false
	}
	return true
}
