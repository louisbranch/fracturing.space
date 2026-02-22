package declarative

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	defaultPageSize = int32(50)
	seedReason      = "seed_declarative_manifest"
)

type authClient interface {
	CreateUser(ctx context.Context, in *authv1.CreateUserRequest, opts ...grpc.CallOption) (*authv1.CreateUserResponse, error)
	GetUser(ctx context.Context, in *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.GetUserResponse, error)
	ListUsers(ctx context.Context, in *authv1.ListUsersRequest, opts ...grpc.CallOption) (*authv1.ListUsersResponse, error)
	ListUserEmails(ctx context.Context, in *authv1.ListUserEmailsRequest, opts ...grpc.CallOption) (*authv1.ListUserEmailsResponse, error)
}

type connectionsClient interface {
	SetUserProfile(ctx context.Context, in *connectionsv1.SetUserProfileRequest, opts ...grpc.CallOption) (*connectionsv1.SetUserProfileResponse, error)
	AddContact(ctx context.Context, in *connectionsv1.AddContactRequest, opts ...grpc.CallOption) (*connectionsv1.AddContactResponse, error)
}

type campaignClient interface {
	CreateCampaign(ctx context.Context, in *gamev1.CreateCampaignRequest, opts ...grpc.CallOption) (*gamev1.CreateCampaignResponse, error)
	GetCampaign(ctx context.Context, in *gamev1.GetCampaignRequest, opts ...grpc.CallOption) (*gamev1.GetCampaignResponse, error)
	ListCampaigns(ctx context.Context, in *gamev1.ListCampaignsRequest, opts ...grpc.CallOption) (*gamev1.ListCampaignsResponse, error)
}

type participantClient interface {
	CreateParticipant(ctx context.Context, in *gamev1.CreateParticipantRequest, opts ...grpc.CallOption) (*gamev1.CreateParticipantResponse, error)
	GetParticipant(ctx context.Context, in *gamev1.GetParticipantRequest, opts ...grpc.CallOption) (*gamev1.GetParticipantResponse, error)
	ListParticipants(ctx context.Context, in *gamev1.ListParticipantsRequest, opts ...grpc.CallOption) (*gamev1.ListParticipantsResponse, error)
}

type characterClient interface {
	CreateCharacter(ctx context.Context, in *gamev1.CreateCharacterRequest, opts ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error)
	SetDefaultControl(ctx context.Context, in *gamev1.SetDefaultControlRequest, opts ...grpc.CallOption) (*gamev1.SetDefaultControlResponse, error)
	GetCharacterSheet(ctx context.Context, in *gamev1.GetCharacterSheetRequest, opts ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error)
	ListCharacters(ctx context.Context, in *gamev1.ListCharactersRequest, opts ...grpc.CallOption) (*gamev1.ListCharactersResponse, error)
}

type sessionClient interface {
	StartSession(ctx context.Context, in *gamev1.StartSessionRequest, opts ...grpc.CallOption) (*gamev1.StartSessionResponse, error)
	EndSession(ctx context.Context, in *gamev1.EndSessionRequest, opts ...grpc.CallOption) (*gamev1.EndSessionResponse, error)
	GetSession(ctx context.Context, in *gamev1.GetSessionRequest, opts ...grpc.CallOption) (*gamev1.GetSessionResponse, error)
	ListSessions(ctx context.Context, in *gamev1.ListSessionsRequest, opts ...grpc.CallOption) (*gamev1.ListSessionsResponse, error)
}

type forkClient interface {
	ForkCampaign(ctx context.Context, in *gamev1.ForkCampaignRequest, opts ...grpc.CallOption) (*gamev1.ForkCampaignResponse, error)
}

type listingClient interface {
	CreateCampaignListing(ctx context.Context, in *listingv1.CreateCampaignListingRequest, opts ...grpc.CallOption) (*listingv1.CreateCampaignListingResponse, error)
	GetCampaignListing(ctx context.Context, in *listingv1.GetCampaignListingRequest, opts ...grpc.CallOption) (*listingv1.GetCampaignListingResponse, error)
}

type runnerDeps struct {
	auth         authClient
	connections  connectionsClient
	campaigns    campaignClient
	participants participantClient
	characters   characterClient
	sessions     sessionClient
	forks        forkClient
	listings     listingClient
}

// Config holds declarative runner settings.
type Config struct {
	ManifestPath string
	StatePath    string
	Verbose      bool
}

// Runner applies one declarative manifest with idempotent state tracking.
type Runner struct {
	cfg  Config
	deps runnerDeps
	errW io.Writer
}

func newRunnerWithClients(cfg Config, deps runnerDeps) *Runner {
	return &Runner{
		cfg:  cfg,
		deps: deps,
		errW: os.Stderr,
	}
}

// Run loads and applies the configured manifest file.
func (r *Runner) Run(ctx context.Context) error {
	manifest, err := LoadManifest(r.cfg.ManifestPath)
	if err != nil {
		return err
	}
	return r.RunManifest(ctx, manifest)
}

// RunManifest applies one manifest directly.
func (r *Runner) RunManifest(ctx context.Context, manifest Manifest) error {
	if r == nil {
		return fmt.Errorf("runner is required")
	}
	if err := ValidateManifest(manifest); err != nil {
		return err
	}
	if err := r.requireDeps(); err != nil {
		return err
	}

	state, err := loadState(r.cfg.StatePath)
	if err != nil {
		return err
	}

	userIDs, err := r.applyUsers(ctx, manifest, &state)
	if err != nil {
		return err
	}
	if err := r.applyContacts(ctx, manifest, userIDs); err != nil {
		return err
	}

	campaignIDs, participantIDs, err := r.applyCampaigns(ctx, manifest, userIDs, &state)
	if err != nil {
		return err
	}
	forkCampaignIDs, err := r.applyForks(ctx, manifest, userIDs, campaignIDs, &state)
	if err != nil {
		return err
	}

	combinedCampaignIDs := make(map[string]string, len(campaignIDs)+len(forkCampaignIDs))
	for key, campaignID := range campaignIDs {
		combinedCampaignIDs[key] = campaignID
	}
	for key, campaignID := range forkCampaignIDs {
		combinedCampaignIDs[key] = campaignID
	}
	_ = participantIDs
	if err := r.applyListings(ctx, manifest, combinedCampaignIDs); err != nil {
		return err
	}

	return saveState(r.cfg.StatePath, state)
}

func (r *Runner) requireDeps() error {
	if r.deps.auth == nil {
		return fmt.Errorf("auth client is required")
	}
	if r.deps.connections == nil {
		return fmt.Errorf("connections client is required")
	}
	if r.deps.campaigns == nil {
		return fmt.Errorf("campaign client is required")
	}
	if r.deps.listings == nil {
		return fmt.Errorf("listing client is required")
	}
	return nil
}

func (r *Runner) logf(format string, args ...any) {
	if r == nil || !r.cfg.Verbose {
		return
	}
	if r.errW == nil {
		return
	}
	_, _ = fmt.Fprintf(r.errW, format+"\n", args...)
}

func (r *Runner) applyUsers(ctx context.Context, manifest Manifest, state *seedState) (map[string]string, error) {
	result := make(map[string]string, len(manifest.Users))
	for _, user := range manifest.Users {
		userID, err := r.resolveUserID(ctx, manifest, user, state)
		if err != nil {
			return nil, err
		}
		result[user.Key] = userID
		if err := r.applyPublicProfile(ctx, userID, user.PublicProfile); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *Runner) resolveUserID(ctx context.Context, manifest Manifest, user ManifestUser, state *seedState) (string, error) {
	entryKey := stateKeyUser(user.Key)
	if existingID := strings.TrimSpace(state.Entries[entryKey]); existingID != "" {
		exists, err := r.userExists(ctx, existingID)
		if err != nil {
			return "", err
		}
		if exists {
			return existingID, nil
		}
	}

	foundID, err := r.findUserIDByEmail(ctx, user.Email)
	if err != nil {
		return "", err
	}
	if foundID != "" {
		state.Entries[entryKey] = foundID
		return foundID, nil
	}

	resp, err := r.deps.auth.CreateUser(ctx, &authv1.CreateUserRequest{
		Email:  user.Email,
		Locale: parseLocale(user.Locale),
	})
	if err != nil {
		return "", fmt.Errorf("create user %q: %w", user.Key, err)
	}
	userID := strings.TrimSpace(resp.GetUser().GetId())
	if userID == "" {
		return "", fmt.Errorf("create user %q: missing user id", user.Key)
	}
	state.Entries[entryKey] = userID
	r.logf("seed %s: created user %s (%s)", manifest.Name, user.Key, userID)
	return userID, nil
}

func (r *Runner) userExists(ctx context.Context, userID string) (bool, error) {
	resp, err := r.deps.auth.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
	if err != nil {
		return false, fmt.Errorf("get user %q: %w", userID, err)
	}
	if resp == nil || resp.GetUser() == nil {
		return false, nil
	}
	return strings.TrimSpace(resp.GetUser().GetId()) != "", nil
}

func (r *Runner) findUserIDByEmail(ctx context.Context, email string) (string, error) {
	target := normalizeEmail(email)
	pageToken := ""
	for {
		resp, err := r.deps.auth.ListUsers(ctx, &authv1.ListUsersRequest{
			PageSize:  defaultPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			return "", fmt.Errorf("list users: %w", err)
		}

		for _, user := range resp.GetUsers() {
			userID := strings.TrimSpace(user.GetId())
			if userID == "" {
				continue
			}
			emailsResp, err := r.deps.auth.ListUserEmails(ctx, &authv1.ListUserEmailsRequest{UserId: userID})
			if err != nil {
				return "", fmt.Errorf("list user emails for %s: %w", userID, err)
			}
			for _, candidate := range emailsResp.GetEmails() {
				if normalizeEmail(candidate.GetEmail()) == target {
					return userID, nil
				}
			}
		}

		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return "", nil
}

func (r *Runner) applyPublicProfile(ctx context.Context, userID string, profile ManifestPublicProfile) error {
	username := strings.TrimSpace(profile.Username)
	name := strings.TrimSpace(profile.Name)
	if username == "" || name == "" {
		return nil
	}

	_, err := r.deps.connections.SetUserProfile(ctx, &connectionsv1.SetUserProfileRequest{
		UserId:        userID,
		Username:      username,
		Name:          name,
		AvatarSetId:   strings.TrimSpace(profile.AvatarSetID),
		AvatarAssetId: strings.TrimSpace(profile.AvatarAssetID),
		Bio:           strings.TrimSpace(profile.Bio),
	})
	if err != nil {
		return fmt.Errorf("set public profile for user %s: %w", userID, err)
	}
	return nil
}

func (r *Runner) applyContacts(ctx context.Context, manifest Manifest, userIDs map[string]string) error {
	for _, user := range manifest.Users {
		ownerUserID := strings.TrimSpace(userIDs[user.Key])
		if ownerUserID == "" {
			return fmt.Errorf("missing resolved owner user id for %q", user.Key)
		}
		for _, contactKey := range user.Contacts {
			contactUserID := strings.TrimSpace(userIDs[contactKey])
			if contactUserID == "" {
				return fmt.Errorf("missing resolved contact user id for key %q", contactKey)
			}
			_, err := r.deps.connections.AddContact(ctx, &connectionsv1.AddContactRequest{
				OwnerUserId:   ownerUserID,
				ContactUserId: contactUserID,
			})
			if err != nil {
				return fmt.Errorf("add contact %s->%s: %w", user.Key, contactKey, err)
			}
		}
	}
	return nil
}

func (r *Runner) applyCampaigns(ctx context.Context, manifest Manifest, userIDs map[string]string, state *seedState) (map[string]string, map[string]string, error) {
	campaignIDs := make(map[string]string, len(manifest.Campaigns))
	participantIDs := make(map[string]string)
	for _, campaign := range manifest.Campaigns {
		campaignID, err := r.resolveCampaignID(ctx, manifest, campaign, userIDs, state)
		if err != nil {
			return nil, nil, err
		}
		campaignIDs[campaign.Key] = campaignID

		participants, err := r.applyParticipants(ctx, campaign.Key, campaignID, campaign.Participants, userIDs, state)
		if err != nil {
			return nil, nil, err
		}
		for participantKey, participantID := range participants {
			participantIDs[stateKeyParticipant(campaign.Key, participantKey)] = participantID
		}

		if err := r.applyCharacters(ctx, campaign.Key, campaignID, campaign.Characters, participants, state); err != nil {
			return nil, nil, err
		}
		if err := r.applySessions(ctx, campaign.Key, campaignID, campaign.Sessions, state); err != nil {
			return nil, nil, err
		}
	}
	return campaignIDs, participantIDs, nil
}

func (r *Runner) resolveCampaignID(ctx context.Context, manifest Manifest, campaign ManifestCampaign, userIDs map[string]string, state *seedState) (string, error) {
	entryKey := stateKeyCampaign(campaign.Key)
	if existingID := strings.TrimSpace(state.Entries[entryKey]); existingID != "" {
		exists, err := r.campaignExists(ctx, existingID)
		if err != nil {
			return "", err
		}
		if exists {
			return existingID, nil
		}
	}

	marker := campaignSeedMarker(manifest.Name, campaign.Key)
	foundID, err := r.findCampaignIDByMarker(ctx, marker)
	if err != nil {
		return "", err
	}
	if foundID != "" {
		state.Entries[entryKey] = foundID
		return foundID, nil
	}

	ownerUserID := strings.TrimSpace(userIDs[campaign.OwnerUserKey])
	if ownerUserID == "" {
		return "", fmt.Errorf("campaign %q owner user %q is unresolved", campaign.Key, campaign.OwnerUserKey)
	}

	resp, err := r.deps.campaigns.CreateCampaign(gameWriteContext(ctx, ownerUserID), &gamev1.CreateCampaignRequest{
		Name:         campaign.Name,
		System:       parseGameSystem(defaultSystemLabel(campaign.System)),
		GmMode:       parseGmMode(defaultGmModeLabel(campaign.GmMode)),
		Intent:       parseCampaignIntent(defaultCampaignIntentLabel(campaign.Intent)),
		AccessPolicy: parseAccessPolicy(defaultAccessPolicyLabel(campaign.AccessPolicy)),
		ThemePrompt:  appendSeedMarker(campaign.ThemePrompt, marker),
	})
	if err != nil {
		return "", fmt.Errorf("create campaign %q: %w", campaign.Key, err)
	}
	campaignID := strings.TrimSpace(resp.GetCampaign().GetId())
	if campaignID == "" {
		return "", fmt.Errorf("create campaign %q: missing campaign id", campaign.Key)
	}
	state.Entries[entryKey] = campaignID
	return campaignID, nil
}

func (r *Runner) campaignExists(ctx context.Context, campaignID string) (bool, error) {
	resp, err := r.deps.campaigns.GetCampaign(gameAdminContext(ctx, ""), &gamev1.GetCampaignRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		return false, fmt.Errorf("get campaign %q: %w", campaignID, err)
	}
	if resp == nil || resp.GetCampaign() == nil {
		return false, nil
	}
	return strings.TrimSpace(resp.GetCampaign().GetId()) != "", nil
}

func (r *Runner) findCampaignIDByMarker(ctx context.Context, marker string) (string, error) {
	pageToken := ""
	for {
		resp, err := r.deps.campaigns.ListCampaigns(gameAdminContext(ctx, ""), &gamev1.ListCampaignsRequest{
			PageSize:  defaultPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			return "", fmt.Errorf("list campaigns: %w", err)
		}
		for _, campaign := range resp.GetCampaigns() {
			theme := strings.TrimSpace(campaign.GetThemePrompt())
			if strings.Contains(theme, marker) {
				return strings.TrimSpace(campaign.GetId()), nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return "", nil
}

func (r *Runner) applyParticipants(ctx context.Context, campaignKey, campaignID string, participants []ManifestParticipant, userIDs map[string]string, state *seedState) (map[string]string, error) {
	result := make(map[string]string, len(participants))
	if r.deps.participants == nil {
		if len(participants) == 0 {
			return result, nil
		}
		return nil, fmt.Errorf("participant client is required for campaign %q", campaignKey)
	}

	for _, participant := range participants {
		entryKey := stateKeyParticipant(campaignKey, participant.Key)
		if existingID := strings.TrimSpace(state.Entries[entryKey]); existingID != "" {
			exists, err := r.participantExists(ctx, campaignID, existingID)
			if err != nil {
				return nil, err
			}
			if exists {
				result[participant.Key] = existingID
				continue
			}
		}

		foundID, err := r.findParticipantIDByName(ctx, campaignID, participant)
		if err != nil {
			return nil, err
		}
		if foundID != "" {
			state.Entries[entryKey] = foundID
			result[participant.Key] = foundID
			continue
		}

		userID := ""
		if participant.UserKey != "" {
			userID = strings.TrimSpace(userIDs[participant.UserKey])
			if userID == "" {
				return nil, fmt.Errorf("campaign %q participant %q references unresolved user key %q", campaignKey, participant.Key, participant.UserKey)
			}
		}
		resp, err := r.deps.participants.CreateParticipant(gameAdminContext(ctx, ""), &gamev1.CreateParticipantRequest{
			CampaignId: campaignID,
			UserId:     userID,
			Name:       participant.Name,
			Role:       parseParticipantRole(defaultParticipantRoleLabel(participant.Role)),
			Controller: parseParticipantController(defaultParticipantControllerLabel(participant.Controller)),
		})
		if err != nil {
			return nil, fmt.Errorf("create participant %q/%q: %w", campaignKey, participant.Key, err)
		}
		participantID := strings.TrimSpace(resp.GetParticipant().GetId())
		if participantID == "" {
			return nil, fmt.Errorf("create participant %q/%q: missing id", campaignKey, participant.Key)
		}
		state.Entries[entryKey] = participantID
		result[participant.Key] = participantID
	}
	return result, nil
}

func (r *Runner) participantExists(ctx context.Context, campaignID, participantID string) (bool, error) {
	resp, err := r.deps.participants.GetParticipant(gameAdminContext(ctx, ""), &gamev1.GetParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		return false, fmt.Errorf("get participant %q/%q: %w", campaignID, participantID, err)
	}
	if resp == nil || resp.GetParticipant() == nil {
		return false, nil
	}
	return strings.TrimSpace(resp.GetParticipant().GetId()) != "", nil
}

func (r *Runner) findParticipantIDByName(ctx context.Context, campaignID string, participant ManifestParticipant) (string, error) {
	pageToken := ""
	targetName := strings.TrimSpace(participant.Name)
	targetRole := parseParticipantRole(defaultParticipantRoleLabel(participant.Role))
	for {
		resp, err := r.deps.participants.ListParticipants(gameAdminContext(ctx, ""), &gamev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   defaultPageSize,
			PageToken:  pageToken,
		})
		if err != nil {
			return "", fmt.Errorf("list participants for campaign %s: %w", campaignID, err)
		}
		for _, existing := range resp.GetParticipants() {
			if strings.TrimSpace(existing.GetName()) == targetName && existing.GetRole() == targetRole {
				return strings.TrimSpace(existing.GetId()), nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return "", nil
}

func (r *Runner) applyCharacters(ctx context.Context, campaignKey, campaignID string, characters []ManifestCharacter, participantIDs map[string]string, state *seedState) error {
	if len(characters) == 0 {
		return nil
	}
	if r.deps.characters == nil {
		return fmt.Errorf("character client is required for campaign %q", campaignKey)
	}

	for _, character := range characters {
		entryKey := stateKeyCharacter(campaignKey, character.Key)
		characterID := strings.TrimSpace(state.Entries[entryKey])
		if characterID != "" {
			exists, err := r.characterExists(ctx, campaignID, characterID)
			if err != nil {
				return err
			}
			if !exists {
				characterID = ""
			}
		}
		if characterID == "" {
			foundID, err := r.findCharacterIDByName(ctx, campaignID, character)
			if err != nil {
				return err
			}
			characterID = foundID
		}
		if characterID == "" {
			resp, err := r.deps.characters.CreateCharacter(gameAdminContext(ctx, ""), &gamev1.CreateCharacterRequest{
				CampaignId: campaignID,
				Name:       character.Name,
				Kind:       parseCharacterKind(defaultCharacterKindLabel(character.Kind)),
				Notes:      character.Notes,
			})
			if err != nil {
				return fmt.Errorf("create character %q/%q: %w", campaignKey, character.Key, err)
			}
			characterID = strings.TrimSpace(resp.GetCharacter().GetId())
			if characterID == "" {
				return fmt.Errorf("create character %q/%q: missing id", campaignKey, character.Key)
			}
		}
		state.Entries[entryKey] = characterID

		controllerKey := strings.TrimSpace(character.ControllerParticipantKey)
		if controllerKey != "" {
			participantID := strings.TrimSpace(participantIDs[controllerKey])
			if participantID == "" {
				return fmt.Errorf("character %q/%q references unknown participant key %q", campaignKey, character.Key, controllerKey)
			}
			if _, err := r.deps.characters.SetDefaultControl(gameAdminContext(ctx, ""), &gamev1.SetDefaultControlRequest{
				CampaignId:    campaignID,
				CharacterId:   characterID,
				ParticipantId: stringValue(participantID),
			}); err != nil {
				return fmt.Errorf("set character control %q/%q: %w", campaignKey, character.Key, err)
			}
		}
	}
	return nil
}

func (r *Runner) characterExists(ctx context.Context, campaignID, characterID string) (bool, error) {
	resp, err := r.deps.characters.GetCharacterSheet(gameAdminContext(ctx, ""), &gamev1.GetCharacterSheetRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return false, fmt.Errorf("get character sheet %q/%q: %w", campaignID, characterID, err)
	}
	if resp == nil || resp.GetCharacter() == nil {
		return false, nil
	}
	return strings.TrimSpace(resp.GetCharacter().GetId()) != "", nil
}

func (r *Runner) findCharacterIDByName(ctx context.Context, campaignID string, character ManifestCharacter) (string, error) {
	pageToken := ""
	targetName := strings.TrimSpace(character.Name)
	targetKind := parseCharacterKind(defaultCharacterKindLabel(character.Kind))
	for {
		resp, err := r.deps.characters.ListCharacters(gameAdminContext(ctx, ""), &gamev1.ListCharactersRequest{
			CampaignId: campaignID,
			PageSize:   defaultPageSize,
			PageToken:  pageToken,
		})
		if err != nil {
			return "", fmt.Errorf("list characters for campaign %s: %w", campaignID, err)
		}
		for _, existing := range resp.GetCharacters() {
			if strings.TrimSpace(existing.GetName()) == targetName && existing.GetKind() == targetKind {
				return strings.TrimSpace(existing.GetId()), nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return "", nil
}

func (r *Runner) applySessions(ctx context.Context, campaignKey, campaignID string, sessions []ManifestSession, state *seedState) error {
	if len(sessions) == 0 {
		return nil
	}
	if r.deps.sessions == nil {
		return fmt.Errorf("session client is required for campaign %q", campaignKey)
	}

	for _, session := range sessions {
		entryKey := stateKeySession(campaignKey, session.Key)
		sessionID := strings.TrimSpace(state.Entries[entryKey])
		sessionRecord := (*gamev1.Session)(nil)

		if sessionID != "" {
			record, err := r.getSession(ctx, campaignID, sessionID)
			if err != nil {
				return err
			}
			if record != nil {
				sessionRecord = record
			} else {
				sessionID = ""
			}
		}
		if sessionID == "" {
			found, err := r.findSessionByName(ctx, campaignID, session.Name)
			if err != nil {
				return err
			}
			if found != nil {
				sessionRecord = found
				sessionID = strings.TrimSpace(found.GetId())
			}
		}
		if sessionID == "" {
			startResp, err := r.deps.sessions.StartSession(gameAdminContext(ctx, ""), &gamev1.StartSessionRequest{
				CampaignId: campaignID,
				Name:       session.Name,
			})
			if err != nil {
				return fmt.Errorf("start session %q/%q: %w", campaignKey, session.Key, err)
			}
			sessionRecord = startResp.GetSession()
			sessionID = strings.TrimSpace(sessionRecord.GetId())
		}
		if sessionID == "" {
			return fmt.Errorf("session %q/%q has empty id", campaignKey, session.Key)
		}
		state.Entries[entryKey] = sessionID

		targetStatus := parseSessionStatus(defaultSessionStatusLabel(session.Status))
		if targetStatus == gamev1.SessionStatus_SESSION_ENDED && sessionRecord != nil && sessionRecord.GetStatus() == gamev1.SessionStatus_SESSION_ACTIVE {
			if _, err := r.deps.sessions.EndSession(gameAdminContext(ctx, ""), &gamev1.EndSessionRequest{
				CampaignId: campaignID,
				SessionId:  sessionID,
			}); err != nil {
				return fmt.Errorf("end session %q/%q: %w", campaignKey, session.Key, err)
			}
		}
	}
	return nil
}

func (r *Runner) getSession(ctx context.Context, campaignID, sessionID string) (*gamev1.Session, error) {
	resp, err := r.deps.sessions.GetSession(gameAdminContext(ctx, ""), &gamev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("get session %q/%q: %w", campaignID, sessionID, err)
	}
	if resp == nil {
		return nil, nil
	}
	return resp.GetSession(), nil
}

func (r *Runner) findSessionByName(ctx context.Context, campaignID, name string) (*gamev1.Session, error) {
	pageToken := ""
	targetName := strings.TrimSpace(name)
	for {
		resp, err := r.deps.sessions.ListSessions(gameAdminContext(ctx, ""), &gamev1.ListSessionsRequest{
			CampaignId: campaignID,
			PageSize:   defaultPageSize,
			PageToken:  pageToken,
		})
		if err != nil {
			return nil, fmt.Errorf("list sessions for campaign %s: %w", campaignID, err)
		}
		for _, existing := range resp.GetSessions() {
			if strings.TrimSpace(existing.GetName()) == targetName {
				return existing, nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return nil, nil
}

func (r *Runner) applyForks(ctx context.Context, manifest Manifest, userIDs map[string]string, campaignIDs map[string]string, state *seedState) (map[string]string, error) {
	results := make(map[string]string, len(manifest.Forks))
	if len(manifest.Forks) == 0 {
		return results, nil
	}
	if r.deps.forks == nil {
		return nil, fmt.Errorf("fork client is required for fork declarations")
	}

	for _, fork := range manifest.Forks {
		entryKey := stateKeyFork(fork.Key)
		if existingID := strings.TrimSpace(state.Entries[entryKey]); existingID != "" {
			exists, err := r.campaignExists(ctx, existingID)
			if err != nil {
				return nil, err
			}
			if exists {
				results[fork.Key] = existingID
				continue
			}
		}

		marker := forkSeedMarker(manifest.Name, fork.Key)
		foundID, err := r.findCampaignIDByNameContains(ctx, marker)
		if err != nil {
			return nil, err
		}
		if foundID != "" {
			state.Entries[entryKey] = foundID
			results[fork.Key] = foundID
			continue
		}

		sourceCampaignID := strings.TrimSpace(campaignIDs[fork.SourceCampaignKey])
		if sourceCampaignID == "" {
			return nil, fmt.Errorf("fork %q source campaign key %q is unresolved", fork.Key, fork.SourceCampaignKey)
		}
		ownerUserID := strings.TrimSpace(userIDs[fork.OwnerUserKey])
		if ownerUserID == "" {
			return nil, fmt.Errorf("fork %q owner user key %q is unresolved", fork.Key, fork.OwnerUserKey)
		}
		newName := appendSeedMarker(fork.NewCampaignName, marker)
		resp, err := r.deps.forks.ForkCampaign(gameWriteContext(ctx, ownerUserID), &gamev1.ForkCampaignRequest{
			SourceCampaignId: sourceCampaignID,
			ForkPoint: &gamev1.ForkPoint{
				EventSeq:  fork.EventSeq,
				SessionId: strings.TrimSpace(fork.SessionID),
			},
			NewCampaignName:  newName,
			CopyParticipants: fork.CopyParticipants,
		})
		if err != nil {
			return nil, fmt.Errorf("fork campaign %q: %w", fork.Key, err)
		}
		forkCampaignID := strings.TrimSpace(resp.GetCampaign().GetId())
		if forkCampaignID == "" {
			return nil, fmt.Errorf("fork campaign %q: missing campaign id", fork.Key)
		}
		state.Entries[entryKey] = forkCampaignID
		results[fork.Key] = forkCampaignID
	}

	return results, nil
}

func (r *Runner) findCampaignIDByNameContains(ctx context.Context, marker string) (string, error) {
	pageToken := ""
	for {
		resp, err := r.deps.campaigns.ListCampaigns(gameAdminContext(ctx, ""), &gamev1.ListCampaignsRequest{
			PageSize:  defaultPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			return "", fmt.Errorf("list campaigns: %w", err)
		}
		for _, campaign := range resp.GetCampaigns() {
			name := strings.TrimSpace(campaign.GetName())
			if strings.Contains(name, marker) {
				return strings.TrimSpace(campaign.GetId()), nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return "", nil
}

func (r *Runner) applyListings(ctx context.Context, manifest Manifest, campaignIDs map[string]string) error {
	for _, listing := range manifest.Listings {
		campaignID := strings.TrimSpace(campaignIDs[listing.CampaignKey])
		if campaignID == "" {
			return fmt.Errorf("listing campaign key %q is unresolved", listing.CampaignKey)
		}

		existing, err := r.deps.listings.GetCampaignListing(ctx, &listingv1.GetCampaignListingRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			return fmt.Errorf("get campaign listing for %s: %w", listing.CampaignKey, err)
		}
		if existing != nil && existing.GetListing() != nil && strings.TrimSpace(existing.GetListing().GetCampaignId()) != "" {
			continue
		}

		_, err = r.deps.listings.CreateCampaignListing(ctx, &listingv1.CreateCampaignListingRequest{
			CampaignId:                 campaignID,
			Title:                      listing.Title,
			Description:                listing.Description,
			RecommendedParticipantsMin: listing.RecommendedParticipantsMin,
			RecommendedParticipantsMax: listing.RecommendedParticipantsMax,
			DifficultyTier:             parseDifficultyTier(listing.DifficultyTier),
			ExpectedDurationLabel:      listing.ExpectedDurationLabel,
			System:                     parseGameSystem(defaultSystemLabel(listing.System)),
		})
		if err != nil {
			return fmt.Errorf("create campaign listing for %s: %w", listing.CampaignKey, err)
		}
	}
	return nil
}

func campaignSeedMarker(manifestName, campaignKey string) string {
	return "[seed:" + strings.TrimSpace(manifestName) + ":campaign:" + strings.TrimSpace(campaignKey) + "]"
}

func forkSeedMarker(manifestName, forkKey string) string {
	return "[seed:" + strings.TrimSpace(manifestName) + ":fork:" + strings.TrimSpace(forkKey) + "]"
}

func appendSeedMarker(value, marker string) string {
	base := strings.TrimSpace(value)
	marker = strings.TrimSpace(marker)
	if marker == "" {
		return base
	}
	if strings.Contains(base, marker) {
		return base
	}
	if base == "" {
		return marker
	}
	return base + " " + marker
}

func gameAdminContext(ctx context.Context, userID string) context.Context {
	pairs := []string{
		grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
		grpcmeta.AuthzOverrideReasonHeader, seedReason,
	}
	if strings.TrimSpace(userID) != "" {
		pairs = append(pairs, grpcmeta.UserIDHeader, strings.TrimSpace(userID))
	}
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(pairs...))
}

func gameWriteContext(ctx context.Context, userID string) context.Context {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return gameAdminContext(ctx, "")
	}
	return gameAdminContext(ctx, userID)
}
