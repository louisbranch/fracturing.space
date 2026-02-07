package errors_test

import (
	"errors"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/errors"
	"github.com/louisbranch/fracturing.space/internal/errors/i18n"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNew(t *testing.T) {
	err := apperrors.New(apperrors.CodeCampaignNameEmpty, "campaign name is empty")

	if err.Code != apperrors.CodeCampaignNameEmpty {
		t.Errorf("Code = %v, want %v", err.Code, apperrors.CodeCampaignNameEmpty)
	}
	if err.Message != "campaign name is empty" {
		t.Errorf("Message = %v, want %v", err.Message, "campaign name is empty")
	}
	if err.Error() != "campaign name is empty" {
		t.Errorf("Error() = %v, want %v", err.Error(), "campaign name is empty")
	}
}

func TestWithMetadata(t *testing.T) {
	metadata := map[string]string{"FromStatus": "DRAFT", "ToStatus": "COMPLETED"}
	err := apperrors.WithMetadata(
		apperrors.CodeCampaignInvalidStatusTransition,
		"invalid status transition: DRAFT -> COMPLETED",
		metadata,
	)

	if err.Code != apperrors.CodeCampaignInvalidStatusTransition {
		t.Errorf("Code = %v, want %v", err.Code, apperrors.CodeCampaignInvalidStatusTransition)
	}
	if len(err.Metadata) != 2 {
		t.Errorf("Metadata len = %v, want %v", len(err.Metadata), 2)
	}
	if err.Metadata["FromStatus"] != "DRAFT" {
		t.Errorf("Metadata[FromStatus] = %v, want %v", err.Metadata["FromStatus"], "DRAFT")
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := apperrors.Wrap(apperrors.CodeNotFound, "campaign not found", cause)

	if err.Code != apperrors.CodeNotFound {
		t.Errorf("Code = %v, want %v", err.Code, apperrors.CodeNotFound)
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
	if err.Unwrap() != cause {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), cause)
	}
}

func TestErrorIs(t *testing.T) {
	err1 := apperrors.New(apperrors.CodeCampaignNameEmpty, "campaign name is empty")
	err2 := apperrors.New(apperrors.CodeCampaignNameEmpty, "different message")
	err3 := apperrors.New(apperrors.CodeNotFound, "not found")

	// Same code should match
	if !errors.Is(err1, err2) {
		t.Errorf("errors.Is(err1, err2) = false, want true")
	}

	// Different codes should not match
	if errors.Is(err1, err3) {
		t.Errorf("errors.Is(err1, err3) = true, want false")
	}
}

func TestErrorAs(t *testing.T) {
	original := apperrors.WithMetadata(
		apperrors.CodeCampaignInvalidStatusTransition,
		"transition failed",
		map[string]string{"FromStatus": "DRAFT"},
	)

	// Wrap in a standard error
	wrapped := apperrors.Wrap(apperrors.CodeUnknown, "outer error", original)

	var target *apperrors.Error
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As() = false, want true")
	}
	// errors.As finds the first match in the chain
	if target.Code != apperrors.CodeUnknown {
		t.Errorf("target.Code = %v, want %v", target.Code, apperrors.CodeUnknown)
	}
}

func TestGRPCCodeMapping(t *testing.T) {
	tests := []struct {
		code     apperrors.Code
		expected codes.Code
	}{
		{apperrors.CodeCampaignNameEmpty, codes.InvalidArgument},
		{apperrors.CodeCampaignInvalidStatusTransition, codes.FailedPrecondition},
		{apperrors.CodeNotFound, codes.NotFound},
		{apperrors.CodeActiveSessionExists, codes.FailedPrecondition},
		{apperrors.CodeUnknown, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			got := tt.code.GRPCCode()
			if got != tt.expected {
				t.Errorf("GRPCCode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestToGRPCStatus(t *testing.T) {
	err := apperrors.WithMetadata(
		apperrors.CodeCampaignInvalidStatusTransition,
		"internal: transition DRAFT -> COMPLETED not allowed",
		map[string]string{"FromStatus": "DRAFT", "ToStatus": "COMPLETED"},
	)

	grpcErr := err.ToGRPCStatus("en-US", "Cannot transition campaign from DRAFT to COMPLETED")

	st := status.Convert(grpcErr)
	if st.Code() != codes.FailedPrecondition {
		t.Errorf("gRPC Code = %v, want %v", st.Code(), codes.FailedPrecondition)
	}
	if st.Message() != "internal: transition DRAFT -> COMPLETED not allowed" {
		t.Errorf("gRPC Message = %v, want %v", st.Message(), "internal: transition DRAFT -> COMPLETED not allowed")
	}

	// Check ErrorInfo detail
	var foundErrorInfo, foundLocalizedMessage bool
	for _, detail := range st.Details() {
		switch d := detail.(type) {
		case *errdetails.ErrorInfo:
			foundErrorInfo = true
			if d.Reason != string(apperrors.CodeCampaignInvalidStatusTransition) {
				t.Errorf("ErrorInfo.Reason = %v, want %v", d.Reason, apperrors.CodeCampaignInvalidStatusTransition)
			}
			if d.Domain != apperrors.Domain {
				t.Errorf("ErrorInfo.Domain = %v, want %v", d.Domain, apperrors.Domain)
			}
			if d.Metadata["FromStatus"] != "DRAFT" {
				t.Errorf("ErrorInfo.Metadata[FromStatus] = %v, want %v", d.Metadata["FromStatus"], "DRAFT")
			}
		case *errdetails.LocalizedMessage:
			foundLocalizedMessage = true
			if d.Locale != "en-US" {
				t.Errorf("LocalizedMessage.Locale = %v, want %v", d.Locale, "en-US")
			}
			if d.Message != "Cannot transition campaign from DRAFT to COMPLETED" {
				t.Errorf("LocalizedMessage.Message = %v, want %v", d.Message, "Cannot transition campaign from DRAFT to COMPLETED")
			}
		}
	}

	if !foundErrorInfo {
		t.Error("ErrorInfo detail not found")
	}
	if !foundLocalizedMessage {
		t.Error("LocalizedMessage detail not found")
	}
}

func TestHandleError(t *testing.T) {
	t.Run("domain error", func(t *testing.T) {
		err := apperrors.New(apperrors.CodeCampaignNameEmpty, "internal: campaign name empty")
		grpcErr := apperrors.HandleError(err, "en-US")

		st := status.Convert(grpcErr)
		if st.Code() != codes.InvalidArgument {
			t.Errorf("gRPC Code = %v, want %v", st.Code(), codes.InvalidArgument)
		}
	})

	t.Run("unknown error", func(t *testing.T) {
		err := errors.New("random error")
		grpcErr := apperrors.HandleError(err, "en-US")

		st := status.Convert(grpcErr)
		if st.Code() != codes.Internal {
			t.Errorf("gRPC Code = %v, want %v", st.Code(), codes.Internal)
		}
		if st.Message() != "an unexpected error occurred" {
			t.Errorf("gRPC Message = %v, want %v", st.Message(), "an unexpected error occurred")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		grpcErr := apperrors.HandleError(nil, "en-US")
		if grpcErr != nil {
			t.Errorf("HandleError(nil) = %v, want nil", grpcErr)
		}
	})
}

func TestGetCode(t *testing.T) {
	t.Run("domain error", func(t *testing.T) {
		err := apperrors.New(apperrors.CodeNotFound, "not found")
		code := apperrors.GetCode(err)
		if code != apperrors.CodeNotFound {
			t.Errorf("GetCode() = %v, want %v", code, apperrors.CodeNotFound)
		}
	})

	t.Run("wrapped domain error", func(t *testing.T) {
		inner := apperrors.New(apperrors.CodeNotFound, "not found")
		outer := apperrors.Wrap(apperrors.CodeUnknown, "outer", inner)
		code := apperrors.GetCode(outer)
		if code != apperrors.CodeUnknown {
			t.Errorf("GetCode() = %v, want %v", code, apperrors.CodeUnknown)
		}
	})

	t.Run("unknown error", func(t *testing.T) {
		err := errors.New("random error")
		code := apperrors.GetCode(err)
		if code != apperrors.CodeUnknown {
			t.Errorf("GetCode() = %v, want %v", code, apperrors.CodeUnknown)
		}
	})
}

func TestIsCode(t *testing.T) {
	err := apperrors.New(apperrors.CodeNotFound, "not found")

	if !apperrors.IsCode(err, apperrors.CodeNotFound) {
		t.Error("IsCode() = false, want true")
	}
	if apperrors.IsCode(err, apperrors.CodeCampaignNameEmpty) {
		t.Error("IsCode() = true, want false")
	}
}

func TestI18nCatalogFormat(t *testing.T) {
	catalog := i18n.GetCatalog("en-US")

	t.Run("simple message", func(t *testing.T) {
		msg := catalog.Format(string(apperrors.CodeCampaignNameEmpty), nil)
		if msg != "Campaign name cannot be empty" {
			t.Errorf("Format() = %v, want %v", msg, "Campaign name cannot be empty")
		}
	})

	t.Run("message with template", func(t *testing.T) {
		metadata := map[string]string{"FromStatus": "DRAFT", "ToStatus": "COMPLETED"}
		msg := catalog.Format(string(apperrors.CodeCampaignInvalidStatusTransition), metadata)
		expected := "Cannot transition campaign from DRAFT to COMPLETED"
		if msg != expected {
			t.Errorf("Format() = %v, want %v", msg, expected)
		}
	})

	t.Run("unknown code fallback", func(t *testing.T) {
		msg := catalog.Format("UNKNOWN_CODE", nil)
		if msg != "UNKNOWN_CODE" {
			t.Errorf("Format() = %v, want %v", msg, "UNKNOWN_CODE")
		}
	})
}

func TestI18nCatalogFallback(t *testing.T) {
	// Unknown locale should fall back to en-US
	catalog := i18n.GetCatalog("fr-FR")
	if catalog.Locale() != "en-US" {
		t.Errorf("Locale() = %v, want %v", catalog.Locale(), "en-US")
	}
}
