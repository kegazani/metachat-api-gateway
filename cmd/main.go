package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"metachat/api-gateway/internal/handlers"

	diaryPb "github.com/kegazani/metachat-proto/diary"
	userPb "github.com/kegazani/metachat-proto/user"
	matchingPb "github.com/kegazani/metachat-proto/matching"
	matchRequestPb "github.com/kegazani/metachat-proto/match_request"
	chatPb "github.com/kegazani/metachat-proto/chat"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
	logger.SetLevel(logrus.InfoLevel)

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
	matchingServiceAddr := viper.GetString("services.matching_service_address")
	matchRequestServiceAddr := viper.GetString("services.match_request_service_address")
	chatServiceAddr := viper.GetString("services.chat_service_address")

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

	matchingConn, err := grpc.Dial(matchingServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Failed to connect to matching service: %v", err)
	}
	defer matchingConn.Close()

	matchRequestConn, err := grpc.Dial(matchRequestServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Failed to connect to match request service: %v", err)
	}
	defer matchRequestConn.Close()

	chatConn, err := grpc.Dial(chatServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Failed to connect to chat service: %v", err)
	}
	defer chatConn.Close()

	// Create gRPC clients
	userClient := userPb.NewUserServiceClient(userConn)
	diaryClient := diaryPb.NewDiaryServiceClient(diaryConn)
	matchingClient := matchingPb.NewMatchingServiceClient(matchingConn)
	matchRequestClient := matchRequestPb.NewMatchRequestServiceClient(matchRequestConn)
	chatClient := chatPb.NewChatServiceClient(chatConn)

	// Initialize handlers
	gatewayHandler := handlers.NewGatewayHandler(userClient, diaryClient, matchingClient, matchRequestClient, chatClient, logger)

	// Setup HTTP router
	router := mux.NewRouter()
	gatewayHandler.RegisterRoutes(router)

	// Create HTTP server
	port := viper.GetString("server.port")
	if port == "" {
		port = "8080" // Default API Gateway port
	}

	loggingRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.WithFields(logrus.Fields{
			"method": r.Method,
			"url":    r.URL.String(),
			"path":   r.URL.Path,
			"remote": r.RemoteAddr,
		}).Info("HTTP request received")
		router.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      loggingRouter,
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
