package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	diaryPb "github.com/metachat/proto/generated/diary"
	userPb "github.com/metachat/proto/generated/user"
)

// GatewayHandler handles HTTP requests and forwards them to gRPC services
type GatewayHandler struct {
	userClient  userPb.UserServiceClient
	diaryClient diaryPb.DiaryServiceClient
	logger      *logrus.Logger
}

// NewGatewayHandler creates a new gateway handler
func NewGatewayHandler(userClient userPb.UserServiceClient, diaryClient diaryPb.DiaryServiceClient, logger *logrus.Logger) *GatewayHandler {
	return &GatewayHandler{
		userClient:  userClient,
		diaryClient: diaryClient,
		logger:      logger,
	}
}

// RegisterRoutes registers HTTP routes for the gateway
func (h *GatewayHandler) RegisterRoutes(router *mux.Router) {
	// User routes
	router.HandleFunc("/users", h.CreateUser).Methods("POST")
	router.HandleFunc("/users/{id}", h.GetUser).Methods("GET")
	router.HandleFunc("/users/{id}", h.UpdateUserProfile).Methods("PUT")
	router.HandleFunc("/users/{id}/archetype", h.AssignArchetype).Methods("POST")
	router.HandleFunc("/users/{id}/archetype", h.UpdateArchetype).Methods("PUT")
	router.HandleFunc("/users/{id}/modalities", h.UpdateModalities).Methods("PUT")
	router.HandleFunc("/users", h.ListUsers).Methods("GET")

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

	// Add CORS middleware
	router.Use(h.corsMiddleware)
}

// User handlers
func (h *GatewayHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req userPb.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.userClient.CreateUser(r.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create user")
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *GatewayHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	req := &userPb.GetUserRequest{
		Id: vars["id"],
	}

	resp, err := h.userClient.GetUser(r.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user")
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
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

	resp, err := h.userClient.UpdateUserProfile(r.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update user profile")
		http.Error(w, "Failed to update user profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
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

	resp, err := h.userClient.AssignArchetype(r.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to assign archetype")
		http.Error(w, "Failed to assign archetype", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
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

	resp, err := h.userClient.UpdateArchetype(r.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update archetype")
		http.Error(w, "Failed to update archetype", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
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

	resp, err := h.userClient.UpdateModalities(r.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update modalities")
		http.Error(w, "Failed to update modalities", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
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

	resp, err := h.userClient.ListUsers(r.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list users")
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Diary handlers
func (h *GatewayHandler) CreateDiaryEntry(w http.ResponseWriter, r *http.Request) {
	var req diaryPb.CreateDiaryEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.diaryClient.CreateDiaryEntry(r.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create diary entry")
		http.Error(w, "Failed to create diary entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *GatewayHandler) GetDiaryEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	req := &diaryPb.GetDiaryEntryRequest{
		Id: vars["id"],
	}

	resp, err := h.diaryClient.GetDiaryEntry(r.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get diary entry")
		http.Error(w, "Failed to get diary entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
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

	resp, err := h.diaryClient.UpdateDiaryEntry(r.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update diary entry")
		http.Error(w, "Failed to update diary entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *GatewayHandler) DeleteDiaryEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	req := &diaryPb.DeleteDiaryEntryRequest{
		Id: vars["id"],
	}

	_, err := h.diaryClient.DeleteDiaryEntry(r.Context(), req)
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

	resp, err := h.diaryClient.StartDiarySession(r.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to start diary session")
		http.Error(w, "Failed to start diary session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *GatewayHandler) EndDiarySession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	req := &diaryPb.EndDiarySessionRequest{
		Id: vars["id"],
	}

	resp, err := h.diaryClient.EndDiarySession(r.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to end diary session")
		http.Error(w, "Failed to end diary session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *GatewayHandler) GetDiarySession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	req := &diaryPb.GetDiarySessionRequest{
		Id: vars["id"],
	}

	resp, err := h.diaryClient.GetDiarySession(r.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get diary session")
		http.Error(w, "Failed to get diary session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *GatewayHandler) ListDiaryEntries(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")
	filter := r.URL.Query().Get("filter")

	req := &diaryPb.ListDiaryEntriesRequest{
		Page:   parsePage(page),
		Limit:  parseLimit(limit),
		Filter: filter,
	}

	resp, err := h.diaryClient.ListDiaryEntries(r.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list diary entries")
		http.Error(w, "Failed to list diary entries", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *GatewayHandler) ListDiarySessions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")
	filter := r.URL.Query().Get("filter")

	req := &diaryPb.ListDiarySessionsRequest{
		Page:   parsePage(page),
		Limit:  parseLimit(limit),
		Filter: filter,
	}

	resp, err := h.diaryClient.ListDiarySessions(r.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list diary sessions")
		http.Error(w, "Failed to list diary sessions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *GatewayHandler) GetDiaryEntriesByUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// Parse query parameters
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	req := &diaryPb.GetDiaryEntriesByUserRequest{
		UserId: vars["userId"],
		Page:   parsePage(page),
		Limit:  parseLimit(limit),
	}

	resp, err := h.diaryClient.GetDiaryEntriesByUser(r.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get diary entries by user")
		http.Error(w, "Failed to get diary entries by user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *GatewayHandler) GetDiarySessionsByUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// Parse query parameters
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	req := &diaryPb.GetDiarySessionsByUserRequest{
		UserId: vars["userId"],
		Page:   parsePage(page),
		Limit:  parseLimit(limit),
	}

	resp, err := h.diaryClient.GetDiarySessionsByUser(r.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get diary sessions by user")
		http.Error(w, "Failed to get diary sessions by user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *GatewayHandler) GetDiaryAnalytics(w http.ResponseWriter, r *http.Request) {
	req := &diaryPb.GetDiaryAnalyticsRequest{}

	resp, err := h.diaryClient.GetDiaryAnalytics(r.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get diary analytics")
		http.Error(w, "Failed to get diary analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Helper functions
func parsePage(pageStr string) int32 {
	if pageStr == "" {
		return 1
	}
	// Parse page string to int32
	// Implementation depends on your requirements
	return 1 // Default
}

func parseLimit(limitStr string) int32 {
	if limitStr == "" {
		return 10 // Default limit
	}
	// Parse limit string to int32
	// Implementation depends on your requirements
	return 10 // Default
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
