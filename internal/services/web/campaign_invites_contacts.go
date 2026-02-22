package web

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) listInviteContactOptions(ctx context.Context, campaignID string, ownerUserID string, invites []*statev1.Invite) []webtemplates.CampaignInviteContactOption {
	if h == nil || h.connectionsClient == nil || h.participantClient == nil {
		return nil
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return nil
	}

	contacts, err := h.listAllContacts(ctx, ownerUserID)
	if err != nil {
		log.Printf("web: list invite contacts failed: %v", err)
		return nil
	}
	if len(contacts) == 0 {
		return nil
	}

	participants, err := h.listAllCampaignParticipants(ctx, campaignID)
	if err != nil {
		log.Printf("web: list campaign participants for contact options failed: %v", err)
		return nil
	}
	return buildInviteContactOptions(contacts, participants, invites)
}

func (h *handler) listAllContacts(ctx context.Context, ownerUserID string) ([]*connectionsv1.Contact, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return nil, nil
	}
	pageToken := ""
	seenTokens := make(map[string]struct{})
	contacts := make([]*connectionsv1.Contact, 0)
	for {
		resp, err := h.connectionsClient.ListContacts(ctx, &connectionsv1.ListContactsRequest{
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
			return nil, fmt.Errorf("list contacts: repeated page token %q", nextToken)
		}
		seenTokens[nextToken] = struct{}{}
		pageToken = nextToken
	}
	return contacts, nil
}

func (h *handler) listAllCampaignParticipants(ctx context.Context, campaignID string) ([]*statev1.Participant, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, nil
	}
	if cached, ok := h.cachedCampaignParticipants(ctx, campaignID); ok {
		return cached, nil
	}
	pageToken := ""
	seenTokens := make(map[string]struct{})
	participants := make([]*statev1.Participant, 0)
	for {
		resp, err := h.participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
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
			return nil, fmt.Errorf("list participants: repeated page token %q", nextToken)
		}
		seenTokens[nextToken] = struct{}{}
		pageToken = nextToken
	}
	h.setCampaignParticipantsCache(ctx, campaignID, participants)
	return participants, nil
}

func buildInviteContactOptions(contacts []*connectionsv1.Contact, participants []*statev1.Participant, invites []*statev1.Invite) []webtemplates.CampaignInviteContactOption {
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

	options := make([]webtemplates.CampaignInviteContactOption, 0, len(contacts))
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
		options = append(options, webtemplates.CampaignInviteContactOption{
			UserID: userID,
			Label:  userID,
		})
	}
	sort.Slice(options, func(i int, j int) bool {
		return options[i].UserID < options[j].UserID
	})
	return options
}
