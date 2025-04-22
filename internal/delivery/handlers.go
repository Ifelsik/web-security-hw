package delivery

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Ifelsik/web-security-hw/internal/proxy"
	"github.com/Ifelsik/web-security-hw/internal/usecase"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type ProxyControlHandlers struct {
	log             *logrus.Entry
	usecase         usecase.UseCase
	proxyServerAddr string
}

func NewHistoryHandlers(usecase usecase.UseCase, log *logrus.Logger, proxyServerAddr string) *ProxyControlHandlers {
	return &ProxyControlHandlers{
		usecase:         usecase,
		log:             logrus.NewEntry(log),
		proxyServerAddr: proxyServerAddr,
	}
}

func (h *ProxyControlHandlers) GetRequestsHistory(w http.ResponseWriter, r *http.Request) {
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

func (h *ProxyControlHandlers) GetRequestByID(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "text/plain")
	h.log.Debug("host", request.Host)
	err = request.Write(w)
	if err != nil {
		h.log.Errorf("Failed to write request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.log.Debug("Request response sent")
}

func (h *ProxyControlHandlers) RepeatRequest(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Repeating request")

	vars := mux.Vars(r)
	strID, ok := vars["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h.log.Debug("Request ID: ", strID)

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

	pc := proxy.NewProxyClient(h.log, h.proxyServerAddr)
	err = pc.Connect()
	if err != nil {
		h.log.Errorf("Failed to connect to proxy: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := pc.SendHTTP(request)
	if err != nil {
		h.log.Errorf("Failed to send request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	err = resp.Write(w)
	if err != nil {
		h.log.Errorf("Failed to write request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.log.Debug("Request response sent")
}

func (h *ProxyControlHandlers) Scan(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Scanning for vulnerabilities")

	vars := mux.Vars(r)
	strID, ok := vars["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h.log.Debug("Request ID: ", strID)

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

	pc := proxy.NewProxyClient(h.log, h.proxyServerAddr)
	err = pc.Connect()
	if err != nil {
		h.log.Errorf("Failed to connect to proxy: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bustedPath := make([]string, 0)

	generator := h.usecase.DirBusterIterable(request)
	h.log.Debug("Testing path")
	for req := generator(); req != nil; req = generator() {
		resp, err := pc.SendHTTP(request)
		if err != nil {
			h.log.Errorf("Failed to send request: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if resp.StatusCode != http.StatusNotFound {
			bustedPath = append(bustedPath, req.URL.Path)
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	_, err = w.Write([]byte(strings.Join(bustedPath, "\n")))
	if err != nil {
		h.log.Errorf("Failed to write request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.log.Debug("Request response sent")
}
