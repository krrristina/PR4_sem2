package main

import (
	"net/http"
	"os"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	pb "github.com/krrristina/PR4_sem2/proto"
	"github.com/krrristina/PR4_sem2/services/tasks/internal"
	"github.com/krrristina/PR4_sem2/shared/logger"
	"github.com/krrristina/PR4_sem2/shared/middleware"
)

func main() {
	log, err := logger.New("tasks")
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	authAddr := os.Getenv("AUTH_GRPC_ADDR")
	if authAddr == "" {
		authAddr = "localhost:50051"
	}

	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("could not connect to auth", zap.Error(err))
	}
	defer conn.Close()

	h := &internal.Handler{
		AuthClient: pb.NewAuthServiceClient(conn),
		Log:        log,
	}

	mux := http.NewServeMux()

	// Роут с метриками
	mux.HandleFunc("/tasks", h.GetTasks)

	// Endpoint для Prometheus — отдаёт метрики
	mux.Handle("/metrics", promhttp.Handler())

	handler := middleware.RequestID(
		middleware.AccessLog(log)(
			middleware.Metrics("/tasks")(mux),
		),
	)

	tasksPort := os.Getenv("TASKS_PORT")
	if tasksPort == "" {
		tasksPort = "8082"
	}

	log.Info("HTTP tasks server starting", zap.String("port", tasksPort))
	if err := http.ListenAndServe(":"+tasksPort, handler); err != nil {
		log.Fatal("server failed", zap.Error(err))
	}
}
