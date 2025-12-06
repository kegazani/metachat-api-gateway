package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	chatPb "github.com/kegazani/metachat-proto/chat"
	diaryPb "github.com/kegazani/metachat-proto/diary"
	matchRequestPb "github.com/kegazani/metachat-proto/match_request"
	matchingPb "github.com/kegazani/metachat-proto/matching"
	userPb "github.com/kegazani/metachat-proto/user"
)

const CorrelationIDHeader = "X-Correlation-ID"
const CorrelationIDMetadataKey = "correlation_id"

type correlationIDKey struct{}

var CorrelationIDKey = correlationIDKey{}

// GatewayHandler handles HTTP requests and forwards them to gRPC services
type GatewayHandler struct {
	userClient         userPb.UserServiceClient
	diaryClient        diaryPb.DiaryServiceClient
	matchingClient     matchingPb.MatchingServiceClient
	matchRequestClient matchRequestPb.MatchRequestServiceClient
	chatClient         chatPb.ChatServiceClient
	logger             *logrus.Logger
}

// NewGatewayHandler creates a new gateway handler
func NewGatewayHandler(userClient userPb.UserServiceClient, diaryClient diaryPb.DiaryServiceClient, matchingClient matchingPb.MatchingServiceClient, matchRequestClient matchRequestPb.MatchRequestServiceClient, chatClient chatPb.ChatServiceClient, logger *logrus.Logger) *GatewayHandler {
	return &GatewayHandler{
		userClient:         userClient,
		diaryClient:        diaryClient,
		matchingClient:     matchingClient,
		matchRequestClient: matchRequestClient,
		chatClient:         chatClient,
		logger:             logger,
	}
}

// RegisterRoutes registers HTTP routes for the gateway
func (h *GatewayHandler) RegisterRoutes(router *mux.Router) {
	// Auth routes
	router.HandleFunc("/auth/login", h.Login).Methods("POST")
	router.HandleFunc("/auth/register", h.Register).Methods("POST")

	// User routes
	router.HandleFunc("/users/{id}", h.GetUser).Methods("GET")
	router.HandleFunc("/users/{id}", h.UpdateUserProfile).Methods("PUT")
	router.HandleFunc("/users/{id}/archetype", h.AssignArchetype).Methods("POST")
	router.HandleFunc("/users/{id}/archetype", h.UpdateArchetype).Methods("PUT")
	router.HandleFunc("/users/{id}/modalities", h.UpdateModalities).Methods("PUT")
	router.HandleFunc("/users", h.ListUsers).Methods("GET")
	router.HandleFunc("/users/{id}/profile-progress", h.GetUserProfileProgress).Methods("GET")
	router.HandleFunc("/users/{id}/statistics", h.GetUserStatistics).Methods("GET")

	// Diary routes
	router.HandleFunc("/diary/entries", h.CreateDiaryEntry).Methods("POST")
	router.HandleFunc("/diary/entries/{id}", h.GetDiaryEntry).Methods("GET")
	router.HandleFunc("/diary/entries/{id}", h.UpdateDiaryEntry).Methods("PUT")
	router.HandleFunc("/diary/entries/{id}", h.DeleteDiaryEntry).Methods("DELETE")
	router.HandleFunc("/diary/sessions", h.StartDiarySession).Methods("POST")
	router.HandleFunc("/diary/sessions/{id}", h.GetDiarySession).Methods("GET")
	router.HandleFunc("/diary/sessions/{id}/end", h.EndDiarySession).Methods("PUT")
	router.HandleFunc("/diary/entries", h.ListDiaryEntries).Methods("GET")
	router.HandleFunc("/diary/sessions", h.ListDiarySessions).Methods("GET")
	router.HandleFunc("/diary/entries/user/{userId}", h.GetDiaryEntriesByUser).Methods("GET")
	router.HandleFunc("/diary/sessions/user/{userId}", h.GetDiarySessionsByUser).Methods("GET")
	router.HandleFunc("/diary/analytics", h.GetDiaryAnalytics).Methods("GET")

	// Matching routes
	router.HandleFunc("/users/{id1}/common-topics/{id2}", h.GetCommonTopics).Methods("GET")

	// Match Request routes
	router.HandleFunc("/match-requests", h.CreateMatchRequest).Methods("POST")
	router.HandleFunc("/match-requests/user/{user_id}", h.GetUserMatchRequests).Methods("GET")
	router.HandleFunc("/match-requests/{request_id}/accept", h.AcceptMatchRequest).Methods("PUT")
	router.HandleFunc("/match-requests/{request_id}/reject", h.RejectMatchRequest).Methods("PUT")
	router.HandleFunc("/match-requests/{request_id}", h.GetMatchRequest).Methods("GET")
	router.HandleFunc("/match-requests/{request_id}", h.CancelMatchRequest).Methods("DELETE")

	// Chat routes
	router.HandleFunc("/chats", h.CreateChat).Methods("POST")
	router.HandleFunc("/chats/{chat_id}", h.GetChat).Methods("GET")
	router.HandleFunc("/chats/user/{user_id}", h.GetUserChats).Methods("GET")
	router.HandleFunc("/chats/{chat_id}/messages", h.SendMessage).Methods("POST")
	router.HandleFunc("/chats/{chat_id}/messages", h.GetChatMessages).Methods("GET")
	router.HandleFunc("/chats/{chat_id}/messages/read", h.MarkMessagesAsRead).Methods("PUT")

	router.Use(h.correlationIDMiddleware)
	router.Use(h.loggingMiddleware)
	router.Use(h.corsMiddleware)

	// Add catch-all handler for unmatched routes
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.logger.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
			"remote": r.RemoteAddr,
		}).Warn("Route not found")
		http.Error(w, "Route not found", http.StatusNotFound)
	})
}

// Auth handlers
func (h *GatewayHandler) Login(w http.ResponseWriter, r *http.Request) {
	h.logger.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
		"remote": r.RemoteAddr,
	}).Info("Login request received")

	var req userPb.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode login request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.userClient.Login(h.contextWithCorrelationID(r.Context()), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to login")
		http.Error(w, "Failed to login", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) Register(w http.ResponseWriter, r *http.Request) {
	h.logger.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
		"remote": r.RemoteAddr,
	}).Info("Register request received")

	var req userPb.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode register request")
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		h.logger.Error("Missing required fields in register request")
		http.Error(w, "Missing required fields: username, email, and password are required", http.StatusBadRequest)
		return
	}

	resp, err := h.userClient.Register(h.contextWithCorrelationID(r.Context()), &req)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"error":    err.Error(),
			"email":    req.Email,
			"username": req.Username,
		}).Error("Failed to register user")

		statusCode := http.StatusInternalServerError
		errorMsg := "Failed to register user"

		if err.Error() == "username already exists" ||
			err.Error() == "email already exists" ||
			err.Error() == "rpc error: code = AlreadyExists desc = username already exists" ||
			err.Error() == "rpc error: code = AlreadyExists desc = email already exists" {
			statusCode = http.StatusConflict
			errorMsg = "User with this username or email already exists"
		} else if err.Error() == "rpc error: code = Unavailable desc = connection error" ||
			err.Error() == "rpc error: code = DeadlineExceeded desc = context deadline exceeded" {
			statusCode = http.StatusServiceUnavailable
			errorMsg = "User service is temporarily unavailable"
		}

		http.Error(w, errorMsg, statusCode)
		return
	}

	w.WriteHeader(http.StatusCreated)
	encodeProtoToJSON(w, resp, h.logger)
}

// User handlers
func (h *GatewayHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	req := &userPb.GetUserRequest{
		Id: vars["id"],
	}

	resp, err := h.userClient.GetUser(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user")
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req userPb.UpdateUserProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Id = vars["id"]

	resp, err := h.userClient.UpdateUserProfile(h.contextWithCorrelationID(r.Context()), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update user profile")
		http.Error(w, "Failed to update user profile", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) AssignArchetype(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req userPb.AssignArchetypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Id = vars["id"]

	resp, err := h.userClient.AssignArchetype(h.contextWithCorrelationID(r.Context()), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to assign archetype")
		http.Error(w, "Failed to assign archetype", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) UpdateArchetype(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req userPb.UpdateArchetypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Id = vars["id"]

	resp, err := h.userClient.UpdateArchetype(h.contextWithCorrelationID(r.Context()), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update archetype")
		http.Error(w, "Failed to update archetype", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) UpdateModalities(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req userPb.UpdateModalitiesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Id = vars["id"]

	resp, err := h.userClient.UpdateModalities(h.contextWithCorrelationID(r.Context()), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update modalities")
		http.Error(w, "Failed to update modalities", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")
	filter := r.URL.Query().Get("filter")

	req := &userPb.ListUsersRequest{
		Page:   parsePage(page),
		Limit:  parseLimit(limit),
		Filter: filter,
	}

	resp, err := h.userClient.ListUsers(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list users")
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) CreateDiaryEntry(w http.ResponseWriter, r *http.Request) {
	var req diaryPb.CreateDiaryEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.diaryClient.CreateDiaryEntry(h.contextWithCorrelationID(r.Context()), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create diary entry")
		http.Error(w, "Failed to create diary entry", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) GetDiaryEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	req := &diaryPb.GetDiaryEntryRequest{
		Id: vars["id"],
	}

	resp, err := h.diaryClient.GetDiaryEntry(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get diary entry")
		http.Error(w, "Failed to get diary entry", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) UpdateDiaryEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req diaryPb.UpdateDiaryEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Id = vars["id"]

	resp, err := h.diaryClient.UpdateDiaryEntry(h.contextWithCorrelationID(r.Context()), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update diary entry")
		http.Error(w, "Failed to update diary entry", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) DeleteDiaryEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	req := &diaryPb.DeleteDiaryEntryRequest{
		Id: vars["id"],
	}

	_, err := h.diaryClient.DeleteDiaryEntry(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to delete diary entry")
		http.Error(w, "Failed to delete diary entry", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *GatewayHandler) StartDiarySession(w http.ResponseWriter, r *http.Request) {
	var req diaryPb.StartDiarySessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.diaryClient.StartDiarySession(h.contextWithCorrelationID(r.Context()), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to start diary session")
		http.Error(w, "Failed to start diary session", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) EndDiarySession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	req := &diaryPb.EndDiarySessionRequest{
		Id: vars["id"],
	}

	resp, err := h.diaryClient.EndDiarySession(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to end diary session")
		http.Error(w, "Failed to end diary session", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) GetDiarySession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	req := &diaryPb.GetDiarySessionRequest{
		Id: vars["id"],
	}

	resp, err := h.diaryClient.GetDiarySession(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get diary session")
		http.Error(w, "Failed to get diary session", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) ListDiaryEntries(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")
	filter := r.URL.Query().Get("filter")

	req := &diaryPb.ListDiaryEntriesRequest{
		Page:   parsePage(page),
		Limit:  parseLimit(limit),
		Filter: filter,
	}

	resp, err := h.diaryClient.ListDiaryEntries(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list diary entries")
		http.Error(w, "Failed to list diary entries", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) ListDiarySessions(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")
	filter := r.URL.Query().Get("filter")

	req := &diaryPb.ListDiarySessionsRequest{
		Page:   parsePage(page),
		Limit:  parseLimit(limit),
		Filter: filter,
	}

	resp, err := h.diaryClient.ListDiarySessions(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list diary sessions")
		http.Error(w, "Failed to list diary sessions", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) GetDiaryEntriesByUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	req := &diaryPb.GetDiaryEntriesByUserRequest{
		UserId: vars["userId"],
		Page:   parsePage(page),
		Limit:  parseLimit(limit),
	}

	resp, err := h.diaryClient.GetDiaryEntriesByUser(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get diary entries by user")
		http.Error(w, "Failed to get diary entries by user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) GetDiarySessionsByUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	req := &diaryPb.GetDiarySessionsByUserRequest{
		UserId: vars["userId"],
		Page:   parsePage(page),
		Limit:  parseLimit(limit),
	}

	resp, err := h.diaryClient.GetDiarySessionsByUser(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get diary sessions by user")
		http.Error(w, "Failed to get diary sessions by user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) GetDiaryAnalytics(w http.ResponseWriter, r *http.Request) {
	req := &diaryPb.GetDiaryAnalyticsRequest{}

	resp, err := h.diaryClient.GetDiaryAnalytics(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get diary analytics")
		http.Error(w, "Failed to get diary analytics", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeProtoToJSON(w, resp, h.logger)
}

var protoJSONMarshaler = protojson.MarshalOptions{
	EmitUnpopulated: true,
	UseProtoNames:   true,
}

func encodeProtoToJSON(w http.ResponseWriter, msg proto.Message, logger *logrus.Logger) {
	jsonBytes, err := protoJSONMarshaler.Marshal(msg)
	if err != nil {
		logger.WithError(err).Error("Failed to marshal protobuf message to JSON")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(jsonBytes); err != nil {
		logger.WithError(err).Error("Failed to write response")
	}
}

func parsePage(pageStr string) int32 {
	if pageStr == "" {
		return 1
	}
	return 1
}

func parseLimit(limitStr string) int32 {
	if limitStr == "" {
		return 10
	}
	return 10
}

func (h *GatewayHandler) GetUserProfileProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	req := &userPb.GetUserProfileProgressRequest{Id: userID}
	resp, err := h.userClient.GetUserProfileProgress(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user profile progress")
		http.Error(w, "Failed to get user profile progress", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) GetUserStatistics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	req := &userPb.GetUserStatisticsRequest{Id: userID}
	resp, err := h.userClient.GetUserStatistics(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user statistics")
		http.Error(w, "Failed to get user statistics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) GetCommonTopics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID1 := vars["id1"]
	userID2 := vars["id2"]

	req := &matchingPb.GetCommonTopicsRequest{UserId1: userID1, UserId2: userID2}
	resp, err := h.matchingClient.GetCommonTopics(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get common topics")
		http.Error(w, "Failed to get common topics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) CreateMatchRequest(w http.ResponseWriter, r *http.Request) {
	var req matchRequestPb.CreateMatchRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode create match request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.matchRequestClient.CreateMatchRequest(h.contextWithCorrelationID(r.Context()), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create match request")
		http.Error(w, "Failed to create match request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) GetUserMatchRequests(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]
	status := r.URL.Query().Get("status")

	req := &matchRequestPb.GetUserMatchRequestsRequest{UserId: userID, Status: status}
	resp, err := h.matchRequestClient.GetUserMatchRequests(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user match requests")
		http.Error(w, "Failed to get user match requests", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) AcceptMatchRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	requestID := vars["request_id"]
	userID := r.URL.Query().Get("user_id")

	req := &matchRequestPb.AcceptMatchRequestRequest{RequestId: requestID, UserId: userID}
	resp, err := h.matchRequestClient.AcceptMatchRequest(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to accept match request")
		http.Error(w, "Failed to accept match request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) RejectMatchRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	requestID := vars["request_id"]
	userID := r.URL.Query().Get("user_id")

	req := &matchRequestPb.RejectMatchRequestRequest{RequestId: requestID, UserId: userID}
	resp, err := h.matchRequestClient.RejectMatchRequest(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to reject match request")
		http.Error(w, "Failed to reject match request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) GetMatchRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	requestID := vars["request_id"]

	req := &matchRequestPb.GetMatchRequestRequest{RequestId: requestID}
	resp, err := h.matchRequestClient.GetMatchRequest(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get match request")
		http.Error(w, "Failed to get match request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) CancelMatchRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	requestID := vars["request_id"]
	userID := r.URL.Query().Get("user_id")

	req := &matchRequestPb.CancelMatchRequestRequest{RequestId: requestID, UserId: userID}
	resp, err := h.matchRequestClient.CancelMatchRequest(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to cancel match request")
		http.Error(w, "Failed to cancel match request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	var req chatPb.CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode create chat request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.chatClient.CreateChat(h.contextWithCorrelationID(r.Context()), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create chat")
		http.Error(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) GetChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["chat_id"]

	req := &chatPb.GetChatRequest{ChatId: chatID}
	resp, err := h.chatClient.GetChat(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get chat")
		http.Error(w, "Failed to get chat", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) GetUserChats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]

	req := &chatPb.GetUserChatsRequest{UserId: userID}
	resp, err := h.chatClient.GetUserChats(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user chats")
		http.Error(w, "Failed to get user chats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["chat_id"]

	var req chatPb.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode send message request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.ChatId = chatID

	resp, err := h.chatClient.SendMessage(h.contextWithCorrelationID(r.Context()), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to send message")
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["chat_id"]
	limitStr := r.URL.Query().Get("limit")
	beforeMessageID := r.URL.Query().Get("before_message_id")

	req := &chatPb.GetChatMessagesRequest{
		ChatId:          chatID,
		BeforeMessageId: beforeMessageID,
	}
	if limitStr != "" {
		var limit int32
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil {
			req.Limit = limit
		}
	}

	resp, err := h.chatClient.GetChatMessages(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get chat messages")
		http.Error(w, "Failed to get chat messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) MarkMessagesAsRead(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["chat_id"]
	userID := r.URL.Query().Get("user_id")

	req := &chatPb.MarkMessagesAsReadRequest{ChatId: chatID, UserId: userID}
	resp, err := h.chatClient.MarkMessagesAsRead(h.contextWithCorrelationID(r.Context()), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to mark messages as read")
		http.Error(w, "Failed to mark messages as read", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeProtoToJSON(w, resp, h.logger)
}

func (h *GatewayHandler) correlationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		w.Header().Set(CorrelationIDHeader, correlationID)
		ctx := context.WithValue(r.Context(), CorrelationIDKey, correlationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *GatewayHandler) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Context().Value(CorrelationIDKey)
		if correlationID == nil {
			correlationID = "unknown"
		}

		h.logger.WithFields(logrus.Fields{
			"correlation_id": correlationID,
			"method":         r.Method,
			"path":           r.URL.Path,
			"remote":         r.RemoteAddr,
			"host":           r.Host,
			"user_agent":     r.UserAgent(),
			"content_type":   r.Header.Get("Content-Type"),
		}).Info("Incoming request")
		next.ServeHTTP(w, r)
	})
}

func (h *GatewayHandler) contextWithCorrelationID(ctx context.Context) context.Context {
	correlationID, ok := ctx.Value(CorrelationIDKey).(string)
	if !ok || correlationID == "" {
		correlationID = uuid.New().String()
	}
	return metadata.AppendToOutgoingContext(ctx, CorrelationIDMetadataKey, correlationID)
}

// corsMiddleware adds CORS headers to all responses
func (h *GatewayHandler) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
