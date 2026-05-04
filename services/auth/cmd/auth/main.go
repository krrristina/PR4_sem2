package main

import (
	"net"
	"os"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	pb "github.com/krrristina/PR3_sem2/proto"
	grpcserver "github.com/krrristina/PR3_sem2/services/auth/internal/grpc"
	"github.com/krrristina/PR3_sem2/shared/logger"
)

func main() {
	// Создаём логгер для сервиса auth
	log, err := logger.New("auth")
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	port := os.Getenv("AUTH_GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, &grpcserver.AuthGRPCServer{Log: log})

	log.Info("gRPC Auth server starting", zap.String("port", port))

	if err := s.Serve(lis); err != nil {
		log.Fatal("failed to serve", zap.Error(err))
	}
}
