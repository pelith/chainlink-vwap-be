package httpwrap

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// HandlerFunc is the function signature for API handlers that return Response and ErrorResponse.
type HandlerFunc func(*http.Request) (*Response, *ErrorResponse)

// Response is the success payload returned by handlers. The Handler writes it as JSON.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       any
}

// ErrorResponse is the error payload returned by handlers. The Handler writes status + JSON body.
type ErrorResponse struct {
	StatusCode int
	ErrorMsg   string
	Err        error
}

// Handler wraps a HandlerFunc into an http.HandlerFunc.
// It writes Content-Type, status code, and JSON body in one place (per api-guide Response Construction).
func Handler(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, errResp := fn(r)
		if errResp != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(errResp.StatusCode)

			body := map[string]string{"error": errResp.ErrorMsg}
			if encErr := json.NewEncoder(w).Encode(body); encErr != nil {
				http.Error(w, fmt.Sprintf("encode json: %v", encErr), http.StatusInternalServerError)
			}

			return
		}

		if resp.Header != nil {
			for k, v := range resp.Header {
				w.Header()[k] = v
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(resp.StatusCode)

		if resp.Body != nil {
			if encErr := json.NewEncoder(w).Encode(resp.Body); encErr != nil {
				http.Error(w, fmt.Sprintf("encode json: %v", encErr), http.StatusInternalServerError)
			}
		}
	}
}

// NewInvalidParamErrorResponse returns an ErrorResponse for invalid URL/query parameters (400).
func NewInvalidParamErrorResponse(param string) *ErrorResponse {
	return &ErrorResponse{
		StatusCode: http.StatusBadRequest,
		ErrorMsg:   "invalid param: " + param,
	}
}

// ErrorRenderer writes an error response to w. Used by middleware.
type ErrorRenderer interface {
	Render(w http.ResponseWriter, r *http.Request)
}

type errorRenderer struct {
	statusCode int
	msg        string
}

func (e *errorRenderer) Render(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(e.statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": e.msg})
}

// NewUnauthorizedError returns an ErrorRenderer for 401 Unauthorized.
func NewUnauthorizedError(err error) ErrorRenderer { //nolint:ireturn // public API returns interface
	msg := "unauthorized"
	if err != nil {
		msg = err.Error()
	}

	return &errorRenderer{statusCode: http.StatusUnauthorized, msg: msg}
}

// NewInternalServerError returns an ErrorRenderer for 500 Internal Server Error.
func NewInternalServerError(err error) ErrorRenderer { //nolint:ireturn // public API returns interface
	msg := "internal error"
	if err != nil {
		msg = err.Error()
	}

	return &errorRenderer{statusCode: http.StatusInternalServerError, msg: msg}
}
