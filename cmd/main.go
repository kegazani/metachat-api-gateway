package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"metachat/api-gateway/internal/handlers"
	"github.com/metachat/config/logging"
	diaryPb "github.com/metachat/proto/generated/diary"
	userPb "github.com/metachat/proto/generated/user"
)

func main() {
	// Initialize logger
	loggerConfig := logging.LoggerConfig{
		ServiceName: "api-gateway",
		Environment: viper.GetString("environment"),
		LogLevel:    viper.GetString("log.level"),
		LogFormat:   viper.GetString("log.format"),
		LogOutput:   viper.GetString("log.output"),
	}
	logger := logging.NewLogger(loggerConfig)

	// Load configuration
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/app/config")

	if err := viper.ReadInConfig(); err != nil {
		logger.Fatalf("Failed to read config file: %v", err)
	}

	// Initialize gRPC clients
	userServiceAddr := viper.GetString("services.user_service_address")
	diaryServiceAddr := viper.GetString("services.diary_service_address")

	userConn, err := grpc.Dial(userServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Failed to connect to user service: %v", err)
	}
	defer userConn.Close()

	diaryConn, err := grpc.Dial(diaryServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Failed to connect to diary service: %v", err)
	}
	defer diaryConn.Close()

	// Create gRPC clients
	userClient := userPb.NewUserServiceClient(userConn)
	diaryClient := diaryPb.NewDiaryServiceClient(diaryConn)

	// Initialize handlers
	gatewayHandler := handlers.NewGatewayHandler(userClient, diaryClient, logger)

	// Setup HTTP router
	router := mux.NewRouter()
	gatewayHandler.RegisterRoutes(router)

	// Create HTTP server
	port := viper.GetString("server.port")
	if port == "" {
		port = "8080" // Default API Gateway port
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("Starting API Gateway server on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down API Gateway server...")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("API Gateway server exited")
}
