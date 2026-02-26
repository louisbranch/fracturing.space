package campaign

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInviteRecipientUsernameRequired = errors.New("recipient username is required")
	ErrInviteConnectionsUnavailable    = errors.New("connections service is not configured")
	ErrInviteRecipientUsernameFormat   = errors.New("recipient username must start with @")
)

// ResolveCampaignParticipantByUserID resolves the participant record for a user in a campaign.
func ResolveCampaignParticipantByUserID(ctx context.Context, campaignID string, userID string, listParticipants func(context.Context, *statev1.ListParticipantsRequest) (*statev1.ListParticipantsResponse, error)) (*statev1.Participant, error) {
	if listParticipants == nil {
		return nil, errors.New("participant client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" || userID == "" {
		return nil, nil
	}

	pageToken := ""
	for {
		resp, err := listParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		if err != nil {
			return nil, err
		}
		for _, participant := range resp.GetParticipants() {
			if participant == nil {
				continue
			}
			if strings.TrimSpace(participant.GetUserId()) == userID {
				return participant, nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}

	return nil, nil
}

// ListAllContacts loads paginated user contacts for a user.
func ListAllContacts(ctx context.Context, ownerUserID string, listContacts func(context.Context, *connectionsv1.ListContactsRequest) (*connectionsv1.ListContactsResponse, error)) ([]*connectionsv1.Contact, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return nil, nil
	}
	if listContacts == nil {
		return nil, errors.New("connections client is not configured")
	}

	pageToken := ""
	seenTokens := make(map[string]struct{})
	contacts := make([]*connectionsv1.Contact, 0)
	for {
		resp, err := listContacts(ctx, &connectionsv1.ListContactsRequest{
			OwnerUserId: ownerUserID,
			PageSize:    50,
			PageToken:   pageToken,
		})
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, resp.GetContacts()...)
		nextToken := strings.TrimSpace(resp.GetNextPageToken())
		if nextToken == "" {
			break
		}
		if _, ok := seenTokens[nextToken]; ok {
			return nil, errors.New("list contacts: repeated page token " + nextToken)
		}
		seenTokens[nextToken] = struct{}{}
		pageToken = nextToken
	}
	return contacts, nil
}

// ListAllCampaignParticipants loads all campaign participants, using cache when available.
func ListAllCampaignParticipants(
	ctx context.Context,
	campaignID string,
	cachedCampaignParticipants func(context.Context, string) ([]*statev1.Participant, bool),
	listParticipants func(context.Context, *statev1.ListParticipantsRequest) (*statev1.ListParticipantsResponse, error),
	setCampaignParticipantsCache func(context.Context, string, []*statev1.Participant),
) ([]*statev1.Participant, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, nil
	}
	if cachedCampaignParticipants != nil {
		if cached, ok := cachedCampaignParticipants(ctx, campaignID); ok {
			return cached, nil
		}
	}

	if listParticipants == nil {
		return nil, errors.New("participant client is not configured")
	}

	pageToken := ""
	seenTokens := make(map[string]struct{})
	participants := make([]*statev1.Participant, 0)
	for {
		resp, err := listParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		if err != nil {
			return nil, err
		}
		participants = append(participants, resp.GetParticipants()...)
		nextToken := strings.TrimSpace(resp.GetNextPageToken())
		if nextToken == "" {
			break
		}
		if _, ok := seenTokens[nextToken]; ok {
			return nil, errors.New("list participants: repeated page token " + nextToken)
		}
		seenTokens[nextToken] = struct{}{}
		pageToken = nextToken
	}

	if setCampaignParticipantsCache != nil {
		setCampaignParticipantsCache(ctx, campaignID, participants)
	}

	return participants, nil
}

// ListInviteContactOptions builds invite options from available contacts.
func ListInviteContactOptions(
	ctx context.Context,
	campaignID string,
	ownerUserID string,
	invites []*statev1.Invite,
	listContacts func(context.Context, *connectionsv1.ListContactsRequest) (*connectionsv1.ListContactsResponse, error),
	cachedCampaignParticipants func(context.Context, string) ([]*statev1.Participant, bool),
	listParticipants func(context.Context, *statev1.ListParticipantsRequest) (*statev1.ListParticipantsResponse, error),
	setCampaignParticipantsCache func(context.Context, string, []*statev1.Participant),
) []templates.CampaignInviteContactOption {
	if listContacts == nil {
		return nil
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return nil
	}

	contacts, err := ListAllContacts(ctx, ownerUserID, listContacts)
	if err != nil {
		log.Printf("web: list invite contacts failed: %v", err)
		return nil
	}
	if len(contacts) == 0 {
		return nil
	}

	participants, err := ListAllCampaignParticipants(
		ctx,
		campaignID,
		cachedCampaignParticipants,
		listParticipants,
		setCampaignParticipantsCache,
	)
	if err != nil {
		log.Printf("web: list campaign participants for contact options failed: %v", err)
		return nil
	}
	return buildInviteContactOptions(contacts, participants, invites)
}

// BuildInviteContactOptions exposes invite option generation for campaign wrappers and tests.
func BuildInviteContactOptions(contacts []*connectionsv1.Contact, participants []*statev1.Participant, invites []*statev1.Invite) []templates.CampaignInviteContactOption {
	return buildInviteContactOptions(contacts, participants, invites)
}

func buildInviteContactOptions(contacts []*connectionsv1.Contact, participants []*statev1.Participant, invites []*statev1.Invite) []templates.CampaignInviteContactOption {
	participantUsers := make(map[string]struct{})
	for _, participant := range participants {
		if participant == nil {
			continue
		}
		userID := strings.TrimSpace(participant.GetUserId())
		if userID == "" {
			continue
		}
		participantUsers[userID] = struct{}{}
	}

	pendingInviteRecipients := make(map[string]struct{})
	for _, invite := range invites {
		if invite == nil || invite.GetStatus() != statev1.InviteStatus_PENDING {
			continue
		}
		recipientUserID := strings.TrimSpace(invite.GetRecipientUserId())
		if recipientUserID == "" {
			continue
		}
		pendingInviteRecipients[recipientUserID] = struct{}{}
	}

	options := make([]templates.CampaignInviteContactOption, 0, len(contacts))
	seen := make(map[string]struct{})
	for _, contact := range contacts {
		if contact == nil {
			continue
		}
		userID := strings.TrimSpace(contact.GetContactUserId())
		if userID == "" {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		if _, ok := participantUsers[userID]; ok {
			continue
		}
		if _, ok := pendingInviteRecipients[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		options = append(options, templates.CampaignInviteContactOption{
			UserID: userID,
			Label:  userID,
		})
	}
	sort.Slice(options, func(i int, j int) bool {
		return options[i].UserID < options[j].UserID
	})
	return options
}

// LookupInviteRecipientVerification resolves a recipient user identifier for invite preview.
func LookupInviteRecipientVerification(ctx context.Context, recipientUserID string, lookupUser func(context.Context, *connectionsv1.LookupUserProfileRequest) (*connectionsv1.LookupUserProfileResponse, error)) (templates.CampaignInviteVerification, error) {
	recipientUserID = strings.TrimSpace(recipientUserID)
	if recipientUserID == "" {
		return templates.CampaignInviteVerification{}, ErrInviteRecipientUsernameRequired
	}
	if !strings.HasPrefix(recipientUserID, "@") {
		return templates.CampaignInviteVerification{}, ErrInviteRecipientUsernameFormat
	}
	username := strings.TrimSpace(strings.TrimPrefix(recipientUserID, "@"))
	if username == "" {
		return templates.CampaignInviteVerification{}, ErrInviteRecipientUsernameRequired
	}
	if lookupUser == nil {
		return templates.CampaignInviteVerification{}, ErrInviteConnectionsUnavailable
	}

	resp, err := lookupUser(ctx, &connectionsv1.LookupUserProfileRequest{
		Username: username,
	})
	if err != nil {
		return templates.CampaignInviteVerification{}, err
	}
	profileRecord := resp.GetUserProfile()
	if profileRecord == nil {
		return templates.CampaignInviteVerification{}, status.Error(codes.NotFound, "username not found")
	}
	resolvedUserID := strings.TrimSpace(profileRecord.GetUserId())
	if resolvedUserID == "" {
		return templates.CampaignInviteVerification{}, status.Error(codes.NotFound, "username not found")
	}

	verification := templates.CampaignInviteVerification{
		HasResult: true,
		Username:  strings.TrimSpace(profileRecord.GetUsername()),
		UserID:    resolvedUserID,
	}
	if verification.Username == "" {
		verification.Username = username
	}
	verification.Name = strings.TrimSpace(profileRecord.GetName())
	verification.AvatarSetID = strings.TrimSpace(profileRecord.GetAvatarSetId())
	verification.AvatarAssetID = strings.TrimSpace(profileRecord.GetAvatarAssetId())
	verification.Bio = strings.TrimSpace(profileRecord.GetBio())
	return verification, nil
}

// ResolveInviteRecipientUserID resolves a user ID for an invite lookup value.
func ResolveInviteRecipientUserID(ctx context.Context, recipientUserID string, lookupUser func(context.Context, *connectionsv1.LookupUserProfileRequest) (*connectionsv1.LookupUserProfileResponse, error)) (string, error) {
	recipientUserID = strings.TrimSpace(recipientUserID)
	if recipientUserID == "" {
		return "", nil
	}
	if !strings.HasPrefix(recipientUserID, "@") {
		return recipientUserID, nil
	}
	username := strings.TrimSpace(strings.TrimPrefix(recipientUserID, "@"))
	if username == "" {
		return "", ErrInviteRecipientUsernameRequired
	}
	if lookupUser == nil {
		return "", ErrInviteConnectionsUnavailable
	}
	resp, err := lookupUser(ctx, &connectionsv1.LookupUserProfileRequest{
		Username: username,
	})
	if err != nil {
		return "", err
	}
	record := resp.GetUserProfile()
	if record == nil {
		return "", status.Error(codes.NotFound, "username not found")
	}
	resolvedUserID := strings.TrimSpace(record.GetUserId())
	if resolvedUserID == "" {
		return "", status.Error(codes.NotFound, "username not found")
	}
	return resolvedUserID, nil
}

// RenderInviteRecipientLookupError renders specific lookup errors to HTTP responses.
func RenderInviteRecipientLookupError(w http.ResponseWriter, r *http.Request, err error, renderError func(http.ResponseWriter, *http.Request, int, string, string)) {
	switch {
	case errors.Is(err, ErrInviteRecipientUsernameRequired):
		renderError(w, r, http.StatusBadRequest, "Invite action unavailable", "recipient username is required")
	case errors.Is(err, ErrInviteRecipientUsernameFormat):
		renderError(w, r, http.StatusBadRequest, "Invite action unavailable", "recipient username must start with @")
	case errors.Is(err, ErrInviteConnectionsUnavailable):
		renderError(w, r, http.StatusServiceUnavailable, "Invite action unavailable", "connections service is not configured")
	case status.Code(err) == codes.InvalidArgument:
		renderError(w, r, http.StatusBadRequest, "Invite action unavailable", "recipient username is invalid")
	case status.Code(err) == codes.NotFound:
		renderError(w, r, http.StatusBadRequest, "Invite action unavailable", "recipient username was not found")
	default:
		renderError(w, r, grpcErrorFromCode(err, http.StatusBadGateway), "Invite action unavailable", "failed to resolve invite recipient")
	}
}

func grpcErrorFromCode(err error, fallback int) int {
	if err == nil {
		return fallback
	}
	if statusCode := status.Code(err); statusCode != codes.OK {
		return grpcErrorStatusFromCode(statusCode)
	}
	return fallback
}

func grpcErrorStatusFromCode(code codes.Code) int {
	switch code {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusBadRequest
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.Internal, codes.Unknown, codes.DataLoss, codes.Unavailable, codes.DeadlineExceeded:
		return http.StatusBadGateway
	default:
		return http.StatusBadGateway
	}
}
