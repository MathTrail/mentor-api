package roadmap

import (
	"net/http"

	"github.com/MathTrail/mentor-api/internal/apierror"
	"github.com/MathTrail/mentor-api/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for roadmap endpoints.
type Handler struct {
	service Service
	logger  *zap.Logger
}

// NewHandler creates a new roadmap handler.
func NewHandler(service Service, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// GetRecommendations godoc
// @Summary Get learning recommendations
// @Description Returns personalised learning focus areas based on student progress.
// @Tags roadmap
// @Produce json
// @Param X-User-ID header string true "Student UUID (set by Oathkeeper)"
// @Success 200 {object} Recommendation
// @Failure 400 {object} apierror.Response
// @Failure 500 {object} apierror.Response
// @Router /api/v1/roadmap/recommendations [get]
func (h *Handler) GetRecommendations(ctx *gin.Context) {
	rawID := ctx.GetHeader(middleware.UserIDHeader)
	if rawID == "" {
		ctx.JSON(http.StatusBadRequest, apierror.Response{
			Code:    "MISSING_USER_ID",
			Message: "X-User-ID header is required",
		})
		return
	}

	studentID, err := uuid.Parse(rawID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, apierror.Response{
			Code:    "INVALID_USER_ID",
			Message: "X-User-ID must be a valid UUID",
		})
		return
	}

	rec, err := h.service.GetRecommendations(ctx.Request.Context(), studentID)
	if err != nil {
		h.logger.Error("failed to get recommendations",
			zap.Error(err),
			zap.Stringer("student_id", studentID),
		)
		ctx.JSON(http.StatusInternalServerError, apierror.Response{
			Code:    "INTERNAL_ERROR",
			Message: "failed to get recommendations",
		})
		return
	}

	ctx.JSON(http.StatusOK, rec)
}
