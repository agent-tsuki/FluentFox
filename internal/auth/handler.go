package auth

import (
	"encoding/json"
	"net/http"

	"github.com/fluentfox/api/pkg/response"
	"go.uber.org/zap"
)

type AuthHandler struct {
	service *AuthService
	logger     *zap.Logger
}

func NewAuthHandler(service *AuthService, log *zap.Logger) *AuthHandler {
	return &AuthHandler{service: service, logger: log}
}

// Register godoc                                                                                                                                                                           
// @Summary      Register a new user                                                                                                                                                        
// @Description  Creates a new user account and sends a verification email                                                                                                                  
// @Tags         auth                                                                                                                                                                       
// @Accept       json                                                                                                                                                                       
// @Produce      json                                                                                                                                                                       
// @Param        request  body      RegisterRequest  true  "Registration payload"                                                                                                           
// @Success      201      {object}  map[string]string                                                                                                                                       
// @Failure      400      {object}  response.ErrorResponse                                                                                                                                  
// @Failure      409      {object}  response.ErrorResponse                                                                                                                                  
// @Failure      422      {object}  response.ErrorResponse                                                                                                                                  
// @Failure      500      {object}  response.ErrorResponse                                                                                                                                  
// @Router       /auth/register [post] 
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Registering new user ")
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.service.registerUser(r.Context(), req); err != nil {
		response.HandleError(w, err, h.logger)
		return
	}

	response.JSON(w, http.StatusCreated, map[string]string{
		"message": "registration successful, check your email to verify",
	})
}