package app

import (
	"errors"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

var (
	errInvalidBeforeSequence  = errors.New("invalid before_seq")
	errInvalidLimit           = errors.New("invalid limit")
	errChatHistoryUnavailable = errors.New("chat history unavailable")
)

// chatHistoryPage captures the browser paging cursor after transport parsing.
type chatHistoryPage struct {
	BeforeSequenceID int64
	Limit            int
}

// handleBootstrap serves the browser bootstrap contract after resolving the
// authenticated play request context.
func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	req, ok := s.requirePlayRequest(w, r)
	if !ok {
		return
	}
	bootstrap, err := s.application().bootstrap(r.Context(), req)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bootstrap)
}

// handleChatHistory parses the browser paging request and then delegates the
// transcript lookup to the app seam.
func (s *Server) handleChatHistory(w http.ResponseWriter, r *http.Request) {
	req, ok := s.requirePlayRequest(w, r)
	if !ok {
		return
	}
	page, err := parseChatHistoryPage(r)
	if err != nil {
		switch {
		case errors.Is(err, errInvalidBeforeSequence):
			writeJSONError(w, http.StatusBadRequest, "invalid before_seq")
		case errors.Is(err, errInvalidLimit):
			writeJSONError(w, http.StatusBadRequest, "invalid limit")
		default:
			writeJSONError(w, http.StatusBadRequest, "invalid history request")
		}
		return
	}
	history, err := s.application().history(r.Context(), req, page)
	if err != nil {
		if errors.Is(err, errChatHistoryUnavailable) {
			writeJSONError(w, http.StatusBadGateway, "failed to load chat history")
			return
		}
		writeRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, history)
}

// parseChatHistoryPage centralizes query parsing so history handlers and tests
// share one definition of valid browser paging input.
func parseChatHistoryPage(r *http.Request) (chatHistoryPage, error) {
	page := chatHistoryPage{
		BeforeSequenceID: 1 << 62,
		Limit:            transcript.DefaultHistoryLimit,
	}
	if r == nil || r.URL == nil {
		return page, nil
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("before_seq")); raw != "" {
		value, err := parseInt64(raw)
		if err != nil {
			return chatHistoryPage{}, errInvalidBeforeSequence
		}
		page.BeforeSequenceID = value
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := parseInt(raw)
		if err != nil {
			return chatHistoryPage{}, errInvalidLimit
		}
		page.Limit = value
	}
	return page, nil
}
