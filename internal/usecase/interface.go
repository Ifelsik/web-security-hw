package usecase

import (
	"context"
	"net/http"

	"github.com/Ifelsik/web-security-hw/internal/models"
)

type RequestsListHistory struct {
	ID     uint   `json:"id"`
	Method string `json:"method"`
	Path   string `json:"path"`
}

type RequestHistory struct {
	ID         uint           `json:"id"`
	Method     string         `json:"method"`
	Path       string         `json:"path"`
	GetParams  map[string]any `json:"getParams"`
	PostParams map[string]any `json:"postParams"`
	Headers    map[string]any `json:"headers"`
	Cookies    map[string]any `json:"cookies"`
	Body       string         `json:"body"`
}

type UseCase interface {
	GetRequestsHistory(ctx context.Context) ([]*RequestsListHistory, error)
	GetRequestByID(ctx context.Context, id uint64) (*http.Request, error)
	SaveRequestResponse(req *http.Request, resp *http.Response) error
	BuildRequest(*models.Request) (*http.Request, error)
	DirBusterIterable(r *http.Request) func() *http.Request
}
