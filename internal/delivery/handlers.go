package delivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Ifelsik/web-security-hw/internal/usecase"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type HistoryHandlers struct {
	log *logrus.Entry
	usecase usecase.UseCase
}

func NewHistoryHandlers(usecase usecase.UseCase, log *logrus.Logger) *HistoryHandlers {
	return &HistoryHandlers{
		usecase: usecase,
		log:     logrus.NewEntry(log),
	}
}

func (h *HistoryHandlers) GetRequestsHistory(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Getting requests history")

	requestsList, err := h.usecase.GetRequestsHistory(r.Context())
	if err != nil {
		h.log.Errorf("Failed to get requests history: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	encoder := json.NewEncoder(w)
	err = encoder.Encode(requestsList)
	if err != nil {
		h.log.Errorf("Failed to encode requests history: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.log.Debug("Requests history response sent")
}

func (h *HistoryHandlers) GetRequestByID(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Getting request by ID")

	vars := mux.Vars(r)
	strID, ok := vars["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(strID)
	if err != nil {
		h.log.Errorf("Failed to parse request ID: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	request, err := h.usecase.GetRequestByID(r.Context(), uint64(id))
	if err != nil {
		h.log.Errorf("Failed to get request by ID: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	encoder := json.NewEncoder(w)
	err = encoder.Encode(request)
	if err != nil {
		h.log.Errorf("Failed to encode request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
