package ai

import (
	"context"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SearchSystemReference searches the configured read-only system reference corpus.
func (s *Service) SearchSystemReference(ctx context.Context, in *aiv1.SearchSystemReferenceRequest) (*aiv1.SearchSystemReferenceResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "search system reference request is required")
	}
	if s.systemReferenceCorpus == nil {
		return nil, status.Error(codes.FailedPrecondition, "system reference corpus is unavailable")
	}
	results, err := s.systemReferenceCorpus.Search(ctx, in.GetSystem(), in.GetQuery(), int(in.GetMaxResults()))
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
func (s *Service) ReadSystemReferenceDocument(ctx context.Context, in *aiv1.ReadSystemReferenceDocumentRequest) (*aiv1.ReadSystemReferenceDocumentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "read system reference request is required")
	}
	if s.systemReferenceCorpus == nil {
		return nil, status.Error(codes.FailedPrecondition, "system reference corpus is unavailable")
	}
	document, err := s.systemReferenceCorpus.Read(ctx, in.GetSystem(), in.GetDocumentId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "read system reference document: %v", err)
	}
	return &aiv1.ReadSystemReferenceDocumentResponse{Document: referenceDocumentToProto(document)}, nil
}
