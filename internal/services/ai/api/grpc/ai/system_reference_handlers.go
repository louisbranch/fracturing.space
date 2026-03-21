package ai

import (
	"context"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/referencecorpus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SystemReferenceHandlers serves the read-only system reference RPCs.
type SystemReferenceHandlers struct {
	aiv1.UnimplementedSystemReferenceServiceServer

	systemReferenceCorpus *referencecorpus.Corpus
}

// NewSystemReferenceHandlers builds a system-reference RPC server with explicit deps.
func NewSystemReferenceHandlers(corpus *referencecorpus.Corpus) *SystemReferenceHandlers {
	return &SystemReferenceHandlers{systemReferenceCorpus: corpus}
}

// SearchSystemReference searches the configured read-only system reference corpus.
func (h *SystemReferenceHandlers) SearchSystemReference(ctx context.Context, in *aiv1.SearchSystemReferenceRequest) (*aiv1.SearchSystemReferenceResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "search system reference request is required")
	}
	if h.systemReferenceCorpus == nil {
		return nil, status.Error(codes.FailedPrecondition, "system reference corpus is unavailable")
	}
	results, err := h.systemReferenceCorpus.Search(ctx, in.GetSystem(), in.GetQuery(), int(in.GetMaxResults()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search system reference: %v", err)
	}
	resp := &aiv1.SearchSystemReferenceResponse{Results: make([]*aiv1.SystemReferenceDocumentSummary, 0, len(results))}
	for _, result := range results {
		resp.Results = append(resp.Results, referenceDocumentSummaryToProto(result))
	}
	return resp, nil
}

// ReadSystemReferenceDocument returns one full reference document.
func (h *SystemReferenceHandlers) ReadSystemReferenceDocument(ctx context.Context, in *aiv1.ReadSystemReferenceDocumentRequest) (*aiv1.ReadSystemReferenceDocumentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "read system reference request is required")
	}
	if h.systemReferenceCorpus == nil {
		return nil, status.Error(codes.FailedPrecondition, "system reference corpus is unavailable")
	}
	document, err := h.systemReferenceCorpus.Read(ctx, in.GetSystem(), in.GetDocumentId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "read system reference document: %v", err)
	}
	return &aiv1.ReadSystemReferenceDocumentResponse{Document: referenceDocumentToProto(document)}, nil
}
