package feedback

import (
	"net/http"

	"github.com/MathTrail/mentor-api/internal/apierror"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for feedback endpoints
type Handler struct {
	feedbacks Service
	logger    *zap.Logger
}

// NewHandler creates a new feedback handler
func NewHandler(svc Service, logger *zap.Logger) *Handler {
	return &Handler{
		feedbacks: svc,
		logger:    logger,
	}
}

// SubmitFeedback godoc
// @Summary Submit student feedback
// @Description Process student feedback about task difficulty.
// @Tags feedback
// @Accept json
// @Produce json
// @Param feedback body FeedbackRequest true "Feedback request"
// @Success 200 {object} StrategyUpdate
// @Failure 400 {object} apierror.Response
// @Failure 500 {object} apierror.Response
// @Router /api/v1/feedback [post]
func (h *Handler) SubmitFeedback(ctx *gin.Context) {
	var req FeedbackRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid feedback request", zap.Error(err))
		ctx.JSON(http.StatusBadRequest, apierror.Response{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	update, err := h.feedbacks.ProcessFeedback(ctx.Request.Context(), &req)
	if err != nil {
		h.logger.Error("failed to process feedback",
			zap.Error(err),
			zap.Stringer("student_id", req.StudentID),
			zap.String("task_id", req.TaskID),
		)
		ctx.JSON(http.StatusInternalServerError, apierror.Response{Code: "INTERNAL_ERROR", Message: "failed to process feedback"})
		return
	}

	ctx.JSON(http.StatusOK, update)
}
