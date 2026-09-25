package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type UpdateAPI struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

type UpdateInstallRequest struct {
	Version string `json:"version" binding:"required"`
}

func NewUpdateAPIFromEnv() UpdateAPI {
	return UpdateAPI{
		BaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("GOTIFY_MU_UPDATER_URL")), "/"),
		Token:   strings.TrimSpace(os.Getenv("GOTIFY_MU_UPDATER_TOKEN")),
		Client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *UpdateAPI) Status(ctx *gin.Context) {
	if !a.configured() {
		ctx.JSON(http.StatusOK, gin.H{
			"ready":   false,
			"state":   "unavailable",
			"message": "Managed updater helper is not configured on this installation.",
		})
		return
	}

	response, err := a.request(ctx, http.MethodGet, "/status", nil)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"state":   "unavailable",
			"message": err.Error(),
		})
		return
	}
	defer response.Body.Close()
	a.copyResponse(ctx, response)
}

func (a *UpdateAPI) Install(ctx *gin.Context) {
	if !a.configured() {
		ctx.AbortWithError(http.StatusServiceUnavailable, errors.New("managed updater helper is not configured"))
		return
	}

	var request UpdateInstallRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	request.Version = strings.TrimSpace(strings.TrimPrefix(request.Version, "v"))
	if request.Version == "" {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("version is required"))
		return
	}

	payload, err := json.Marshal(request)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	response, err := a.request(ctx, http.MethodPost, "/install", bytes.NewReader(payload))
	if err != nil {
		ctx.AbortWithError(http.StatusBadGateway, err)
		return
	}
	defer response.Body.Close()
	a.copyResponse(ctx, response)
}

func (a *UpdateAPI) configured() bool {
	return a.BaseURL != "" && a.Token != ""
}

func (a *UpdateAPI) request(ctx *gin.Context, method, path string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx.Request.Context(), method, a.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+a.Token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := a.Client.Do(request)
	if err != nil {
		return nil, errors.New("managed updater helper is unavailable")
	}
	return response, nil
}

func (a *UpdateAPI) copyResponse(ctx *gin.Context, response *http.Response) {
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		ctx.AbortWithError(http.StatusBadGateway, err)
		return
	}
	ctx.Data(response.StatusCode, contentType, body)
}
