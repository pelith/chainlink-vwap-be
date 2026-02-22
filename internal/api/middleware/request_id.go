package middleware

// Ported from Goji's middleware, source:
// https://github.com/zenazn/goji/tree/master/web/middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/httplog/v3"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Key to use when setting the request ID.
type ctxKeyRequestID struct{}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}

		span := trace.SpanFromContext(ctx)
		if span.IsRecording() {
			span.SetAttributes(attribute.String("http.request.header.x-request-id", requestID))

			spanContext := span.SpanContext()

			httplog.SetAttrs(ctx, slog.Attr{
				Key:   "otel.trace_id",
				Value: slog.StringValue(spanContext.TraceID().String()),
			})

			httplog.SetAttrs(ctx, slog.Attr{
				Key:   "otel.span_id",
				Value: slog.StringValue(spanContext.SpanID().String()),
			})
		}

		ctx = context.WithValue(ctx, ctxKeyRequestID{}, requestID)

		httplog.SetAttrs(ctx, slog.Attr{
			Key:   "http.request.header.x-request-id",
			Value: slog.StringValue(requestID),
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetReqID returns a request ID from the given context if one is present.
// Returns the empty string if a request ID cannot be found.
func GetReqID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if reqID, ok := ctx.Value(ctxKeyRequestID{}).(string); ok {
		return reqID
	}

	return ""
}
