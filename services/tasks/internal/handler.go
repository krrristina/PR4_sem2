package internal

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/krrristina/PR3_sem2/proto"
	"github.com/krrristina/PR3_sem2/shared/middleware"
)

type Handler struct {
	AuthClient pb.AuthServiceClient
	Log        *zap.Logger
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}

func (h *Handler) verifyToken(ctx context.Context, token, reqID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Прокидываем request-id в gRPC метаданные
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-request-id", reqID))

	h.Log.Info("calling grpc verify",
		zap.String("request_id", reqID),
		zap.String("component", "auth_client"),
	)

	resp, err := h.AuthClient.Verify(ctx, &pb.VerifyRequest{Token: token})
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.Unauthenticated:
			h.Log.Warn("grpc verify: unauthenticated",
				zap.String("request_id", reqID),
				zap.String("component", "auth_client"),
			)
			return "", fmt.Errorf("unauthorized")
		case codes.DeadlineExceeded:
			h.Log.Error("grpc verify: deadline exceeded",
				zap.String("request_id", reqID),
				zap.String("component", "auth_client"),
			)
			return "", fmt.Errorf("auth unavailable")
		default:
			h.Log.Error("grpc verify: unexpected error",
				zap.String("request_id", reqID),
				zap.String("component", "auth_client"),
				zap.Error(err),
			)
			return "", fmt.Errorf("auth unavailable")
		}
	}

	if !resp.Valid {
		return "", fmt.Errorf("unauthorized")
	}
	return resp.Subject, nil
}

func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	token := extractToken(r)
	if token == "" {
		h.Log.Warn("missing token",
			zap.String("request_id", reqID),
			zap.String("component", "handler"),
		)
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	subject, err := h.verifyToken(r.Context(), token, reqID)
	if err != nil {
		if err.Error() == "unauthorized" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		} else {
			http.Error(w, "auth service unavailable", http.StatusServiceUnavailable)
		}
		return
	}

	h.Log.Info("request authorized",
		zap.String("request_id", reqID),
		zap.String("subject", subject),
	)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"tasks": ["task1", "task2"], "user": "%s"}`, subject)
}
