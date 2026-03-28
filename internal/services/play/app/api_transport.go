package app

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

func parseInt64(value string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(value), 10, 64)
}

func parseInt(value string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(value))
}

var (
	errInvalidBeforeSequence  = errors.New("invalid before_seq")
	errInvalidLimit           = errors.New("invalid limit")
	errInvalidAIDebugPageSize = errors.New("invalid page_size")
	errChatHistoryUnavailable = errors.New("chat history unavailable")
)

// chatHistoryPage captures the browser paging cursor after transport parsing.
type chatHistoryPage struct {
	BeforeSequenceID int64
	Limit            int
}

// aiDebugPage captures browser pagination for AI debug turn history.
type aiDebugPage struct {
	PageSize  int
	PageToken string
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

// handleAIDebugTurns returns session-scoped AI GM turn summaries for the play shell.
func (s *Server) handleAIDebugTurns(w http.ResponseWriter, r *http.Request) {
	req, ok := s.requirePlayRequest(w, r)
	if !ok {
		return
	}
	page, err := parseAIDebugPage(r)
	if err != nil {
		if errors.Is(err, errInvalidAIDebugPageSize) {
			writeJSONError(w, http.StatusBadRequest, "invalid page_size")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid ai debug request")
		return
	}
	turns, err := s.application().aiDebugTurns(r.Context(), req, page)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, turns)
}

// handleAIDebugTurn returns one turn plus its ordered debug entries.
func (s *Server) handleAIDebugTurn(w http.ResponseWriter, r *http.Request) {
	req, ok := s.requirePlayRequest(w, r)
	if !ok {
		return
	}
	turnID := strings.TrimSpace(r.PathValue("turnID"))
	if turnID == "" {
		http.NotFound(w, r)
		return
	}
	turn, err := s.application().aiDebugTurn(r.Context(), req, turnID)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, turn)
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

// parseAIDebugPage keeps browser pagination validation for AI debug history in one place.
func parseAIDebugPage(r *http.Request) (aiDebugPage, error) {
	page := aiDebugPage{PageSize: 20}
	if r == nil || r.URL == nil {
		return page, nil
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("page_token")); raw != "" {
		page.PageToken = raw
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("page_size")); raw != "" {
		value, err := parseInt(raw)
		if err != nil {
			return aiDebugPage{}, errInvalidAIDebugPageSize
		}
		if value <= 0 {
			return aiDebugPage{}, errInvalidAIDebugPageSize
		}
		if value > 50 {
			value = 50
		}
		page.PageSize = value
	}
	return page, nil
}
