package domain

import (
	"context"
	"strings"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSceneCreateHandlerUsesContextDefaults(t *testing.T) {
	t.Parallel()

	client := &fakeSceneClient{
		createResp: &statev1.CreateSceneResponse{SceneId: "scene-1"},
	}
	handler := SceneCreateHandler(client, func() Context {
		return Context{CampaignID: "camp-1", SessionID: "sess-1"}
	}, nil)

	_, out, err := handler(context.Background(), &mcp.CallToolRequest{}, SceneCreateInput{
		Name:         "Opening",
		Description:  "Cold open",
		CharacterIDs: []string{"char-1", "char-2"},
	})
	if err != nil {
		t.Fatalf("SceneCreateHandler() error = %v", err)
	}
	if client.lastCreate == nil {
		t.Fatal("expected create request")
	}
	if client.lastCreate.GetCampaignId() != "camp-1" || client.lastCreate.GetSessionId() != "sess-1" {
		t.Fatalf("create request = %#v", client.lastCreate)
	}
	if out.SceneID != "scene-1" || out.CampaignID != "camp-1" || out.SessionID != "sess-1" {
		t.Fatalf("scene create output = %#v", out)
	}
}

func TestSceneListResourceHandlerReadsSessionScenes(t *testing.T) {
	t.Parallel()

	client := &fakeSceneClient{
		listResp: &statev1.ListScenesResponse{
			Scenes: []*statev1.Scene{{
				SceneId:     "scene-1",
				SessionId:   "sess-1",
				Name:        "Opening",
				Description: "Cold open",
				Active:      true,
				CharacterIds: []string{
					"char-1",
				},
				CreatedAt: timestamppb.Now(),
				UpdatedAt: timestamppb.Now(),
			}},
		},
	}
	res, err := SceneListResourceHandler(client)(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "campaign://camp-1/sessions/sess-1/scenes"},
	})
	if err != nil {
		t.Fatalf("SceneListResourceHandler() error = %v", err)
	}
	if client.lastList == nil {
		t.Fatal("expected list request")
	}
	if client.lastList.GetCampaignId() != "camp-1" || client.lastList.GetSessionId() != "sess-1" {
		t.Fatalf("list request = %#v", client.lastList)
	}
	if len(res.Contents) != 1 || !strings.Contains(res.Contents[0].Text, "\"scene_id\": \"scene-1\"") {
		t.Fatalf("scene list contents = %#v", res.Contents)
	}
}

func TestParseSceneListURIRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		"campaign://camp-1",
		"campaign://_/sessions/sess-1/scenes",
		"campaign://camp-1/sessions/_/scenes",
		"campaign://camp-1/sessions/sess-1/other",
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			t.Parallel()
			if _, _, err := parseSceneListURI(tc); err == nil {
				t.Fatalf("expected error for %q", tc)
			}
		})
	}
}
