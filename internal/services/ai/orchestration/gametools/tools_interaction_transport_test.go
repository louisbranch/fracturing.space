package gametools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
)

type interactionTransportClientStub struct {
	statev1.InteractionServiceClient
	getInteractionStateFunc      func(context.Context, *statev1.GetInteractionStateRequest, ...grpc.CallOption) (*statev1.GetInteractionStateResponse, error)
	activateSceneFunc            func(context.Context, *statev1.ActivateSceneRequest, ...grpc.CallOption) (*statev1.ActivateSceneResponse, error)
	openScenePlayerPhaseFunc     func(context.Context, *statev1.OpenScenePlayerPhaseRequest, ...grpc.CallOption) (*statev1.OpenScenePlayerPhaseResponse, error)
	recordSceneGMInteractionFunc func(context.Context, *statev1.RecordSceneGMInteractionRequest, ...grpc.CallOption) (*statev1.RecordSceneGMInteractionResponse, error)
	resolveScenePlayerReviewFunc func(context.Context, *statev1.ResolveScenePlayerReviewRequest, ...grpc.CallOption) (*statev1.ResolveScenePlayerReviewResponse, error)
	openSessionOOCFunc           func(context.Context, *statev1.OpenSessionOOCRequest, ...grpc.CallOption) (*statev1.OpenSessionOOCResponse, error)
	resolveSessionOOCFunc        func(context.Context, *statev1.ResolveSessionOOCRequest, ...grpc.CallOption) (*statev1.ResolveSessionOOCResponse, error)
	postSessionOOCFunc           func(context.Context, *statev1.PostSessionOOCRequest, ...grpc.CallOption) (*statev1.PostSessionOOCResponse, error)
	markOOCReadyToResumeFunc     func(context.Context, *statev1.MarkOOCReadyToResumeRequest, ...grpc.CallOption) (*statev1.MarkOOCReadyToResumeResponse, error)
	clearOOCReadyToResumeFunc    func(context.Context, *statev1.ClearOOCReadyToResumeRequest, ...grpc.CallOption) (*statev1.ClearOOCReadyToResumeResponse, error)
}

type campaignAITransportClientStub struct {
	statev1.CampaignAIOrchestrationServiceClient
	concludeSessionFunc func(context.Context, *statev1.ConcludeSessionRequest, ...grpc.CallOption) (*statev1.ConcludeSessionResponse, error)
}

func (s *interactionTransportClientStub) GetInteractionState(ctx context.Context, req *statev1.GetInteractionStateRequest, opts ...grpc.CallOption) (*statev1.GetInteractionStateResponse, error) {
	if s.getInteractionStateFunc != nil {
		return s.getInteractionStateFunc(ctx, req, opts...)
	}
	return &statev1.GetInteractionStateResponse{}, nil
}

func (s *interactionTransportClientStub) ActivateScene(ctx context.Context, req *statev1.ActivateSceneRequest, opts ...grpc.CallOption) (*statev1.ActivateSceneResponse, error) {
	if s.activateSceneFunc != nil {
		return s.activateSceneFunc(ctx, req, opts...)
	}
	return &statev1.ActivateSceneResponse{}, nil
}

func (s *interactionTransportClientStub) OpenScenePlayerPhase(ctx context.Context, req *statev1.OpenScenePlayerPhaseRequest, opts ...grpc.CallOption) (*statev1.OpenScenePlayerPhaseResponse, error) {
	if s.openScenePlayerPhaseFunc != nil {
		return s.openScenePlayerPhaseFunc(ctx, req, opts...)
	}
	return &statev1.OpenScenePlayerPhaseResponse{}, nil
}

func (*interactionTransportClientStub) SubmitScenePlayerAction(context.Context, *statev1.SubmitScenePlayerActionRequest, ...grpc.CallOption) (*statev1.SubmitScenePlayerActionResponse, error) {
	return nil, nil
}

func (*interactionTransportClientStub) YieldScenePlayerPhase(context.Context, *statev1.YieldScenePlayerPhaseRequest, ...grpc.CallOption) (*statev1.YieldScenePlayerPhaseResponse, error) {
	return nil, nil
}

func (*interactionTransportClientStub) WithdrawScenePlayerYield(context.Context, *statev1.WithdrawScenePlayerYieldRequest, ...grpc.CallOption) (*statev1.WithdrawScenePlayerYieldResponse, error) {
	return nil, nil
}

func (*interactionTransportClientStub) InterruptScenePlayerPhase(context.Context, *statev1.InterruptScenePlayerPhaseRequest, ...grpc.CallOption) (*statev1.InterruptScenePlayerPhaseResponse, error) {
	return nil, nil
}

func (s *interactionTransportClientStub) RecordSceneGMInteraction(ctx context.Context, req *statev1.RecordSceneGMInteractionRequest, opts ...grpc.CallOption) (*statev1.RecordSceneGMInteractionResponse, error) {
	if s.recordSceneGMInteractionFunc != nil {
		return s.recordSceneGMInteractionFunc(ctx, req, opts...)
	}
	return &statev1.RecordSceneGMInteractionResponse{}, nil
}

func (s *interactionTransportClientStub) ResolveScenePlayerReview(ctx context.Context, req *statev1.ResolveScenePlayerReviewRequest, opts ...grpc.CallOption) (*statev1.ResolveScenePlayerReviewResponse, error) {
	if s.resolveScenePlayerReviewFunc != nil {
		return s.resolveScenePlayerReviewFunc(ctx, req, opts...)
	}
	return &statev1.ResolveScenePlayerReviewResponse{}, nil
}

func (s *interactionTransportClientStub) OpenSessionOOC(ctx context.Context, req *statev1.OpenSessionOOCRequest, opts ...grpc.CallOption) (*statev1.OpenSessionOOCResponse, error) {
	if s.openSessionOOCFunc != nil {
		return s.openSessionOOCFunc(ctx, req, opts...)
	}
	return &statev1.OpenSessionOOCResponse{}, nil
}

func (s *interactionTransportClientStub) PostSessionOOC(ctx context.Context, req *statev1.PostSessionOOCRequest, opts ...grpc.CallOption) (*statev1.PostSessionOOCResponse, error) {
	if s.postSessionOOCFunc != nil {
		return s.postSessionOOCFunc(ctx, req, opts...)
	}
	return &statev1.PostSessionOOCResponse{}, nil
}

func (s *interactionTransportClientStub) MarkOOCReadyToResume(ctx context.Context, req *statev1.MarkOOCReadyToResumeRequest, opts ...grpc.CallOption) (*statev1.MarkOOCReadyToResumeResponse, error) {
	if s.markOOCReadyToResumeFunc != nil {
		return s.markOOCReadyToResumeFunc(ctx, req, opts...)
	}
	return &statev1.MarkOOCReadyToResumeResponse{}, nil
}

func (s *interactionTransportClientStub) ClearOOCReadyToResume(ctx context.Context, req *statev1.ClearOOCReadyToResumeRequest, opts ...grpc.CallOption) (*statev1.ClearOOCReadyToResumeResponse, error) {
	if s.clearOOCReadyToResumeFunc != nil {
		return s.clearOOCReadyToResumeFunc(ctx, req, opts...)
	}
	return &statev1.ClearOOCReadyToResumeResponse{}, nil
}

func (s *interactionTransportClientStub) ResolveSessionOOC(ctx context.Context, req *statev1.ResolveSessionOOCRequest, opts ...grpc.CallOption) (*statev1.ResolveSessionOOCResponse, error) {
	if s.resolveSessionOOCFunc != nil {
		return s.resolveSessionOOCFunc(ctx, req, opts...)
	}
	return &statev1.ResolveSessionOOCResponse{}, nil
}

func (*interactionTransportClientStub) SetSessionGMAuthority(context.Context, *statev1.SetSessionGMAuthorityRequest, ...grpc.CallOption) (*statev1.SetSessionGMAuthorityResponse, error) {
	return nil, nil
}

func (*interactionTransportClientStub) SetSessionCharacterController(context.Context, *statev1.SetSessionCharacterControllerRequest, ...grpc.CallOption) (*statev1.SetSessionCharacterControllerResponse, error) {
	return nil, nil
}

func (*interactionTransportClientStub) RetryAIGMTurn(context.Context, *statev1.RetryAIGMTurnRequest, ...grpc.CallOption) (*statev1.RetryAIGMTurnResponse, error) {
	return nil, nil
}

func (s *campaignAITransportClientStub) ConcludeSession(ctx context.Context, req *statev1.ConcludeSessionRequest, opts ...grpc.CallOption) (*statev1.ConcludeSessionResponse, error) {
	if s.concludeSessionFunc != nil {
		return s.concludeSessionFunc(ctx, req, opts...)
	}
	return &statev1.ConcludeSessionResponse{}, nil
}

func newInteractionTransportSession(stub *interactionTransportClientStub) *DirectSession {
	return NewDirectSession(Clients{Interaction: stub}, SessionContext{
		CampaignID:    "camp-ctx",
		SessionID:     "sess-ctx",
		ParticipantID: "gm-1",
	})
}

func sampleInteractionState(sceneID string) *statev1.InteractionState {
	return &statev1.InteractionState{
		CampaignId:   "camp-ctx",
		CampaignName: "Test Campaign",
		ActiveScene: &statev1.InteractionScene{
			SceneId: sceneID,
			Name:    "Scene " + sceneID,
		},
		PlayerPhase: &statev1.ScenePlayerPhase{},
		Ooc:         &statev1.OOCState{},
	}
}

func decodeInteractionToolResult(t *testing.T, result string) interactionStateResult {
	t.Helper()
	var decoded interactionStateResult
	if err := json.Unmarshal([]byte(result), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return decoded
}

func TestInteractionStateRead(t *testing.T) {
	t.Parallel()

	stub := &interactionTransportClientStub{
		getInteractionStateFunc: func(_ context.Context, req *statev1.GetInteractionStateRequest, _ ...grpc.CallOption) (*statev1.GetInteractionStateResponse, error) {
			if req.GetCampaignId() != "camp-ctx" {
				t.Fatalf("campaign id = %q, want camp-ctx", req.GetCampaignId())
			}
			return &statev1.GetInteractionStateResponse{State: sampleInteractionState("scene-1")}, nil
		},
	}

	result, err := newInteractionTransportSession(stub).interactionStateRead(context.Background(), nil)
	if err != nil {
		t.Fatalf("interactionStateRead() error = %v", err)
	}
	decoded := decodeInteractionToolResult(t, result.Output)
	if decoded.ActiveScene.SceneID != "scene-1" {
		t.Fatalf("active scene = %q, want scene-1", decoded.ActiveScene.SceneID)
	}
}

func TestInteractionActivateScene(t *testing.T) {
	t.Parallel()

	stub := &interactionTransportClientStub{
		activateSceneFunc: func(_ context.Context, req *statev1.ActivateSceneRequest, _ ...grpc.CallOption) (*statev1.ActivateSceneResponse, error) {
			if req.GetCampaignId() != "camp-ctx" || req.GetSceneId() != "scene-2" {
				t.Fatalf("request = %#v", req)
			}
			return &statev1.ActivateSceneResponse{State: sampleInteractionState("scene-2")}, nil
		},
	}

	result, err := newInteractionTransportSession(stub).interactionActivateScene(context.Background(), []byte(`{"scene_id":"scene-2"}`))
	if err != nil {
		t.Fatalf("interactionActivateScene() error = %v", err)
	}
	if decoded := decodeInteractionToolResult(t, result.Output); decoded.ActiveScene.SceneID != "scene-2" {
		t.Fatalf("active scene = %q, want scene-2", decoded.ActiveScene.SceneID)
	}
}

func TestInteractionOpenScenePlayerPhase(t *testing.T) {
	t.Parallel()

	stub := &interactionTransportClientStub{
		getInteractionStateFunc: func(_ context.Context, _ *statev1.GetInteractionStateRequest, _ ...grpc.CallOption) (*statev1.GetInteractionStateResponse, error) {
			return &statev1.GetInteractionStateResponse{State: sampleInteractionState("scene-active")}, nil
		},
		openScenePlayerPhaseFunc: func(_ context.Context, req *statev1.OpenScenePlayerPhaseRequest, _ ...grpc.CallOption) (*statev1.OpenScenePlayerPhaseResponse, error) {
			if req.GetSceneId() != "scene-active" {
				t.Fatalf("scene id = %q, want scene-active", req.GetSceneId())
			}
			if len(req.GetCharacterIds()) != 1 || req.GetCharacterIds()[0] != "char-1" {
				t.Fatalf("character ids = %#v", req.GetCharacterIds())
			}
			if req.GetInteraction().GetTitle() != "Arrival" || len(req.GetInteraction().GetCharacterIds()) != 1 || req.GetInteraction().GetCharacterIds()[0] != "char-1" {
				t.Fatalf("interaction = %#v", req.GetInteraction())
			}
			return &statev1.OpenScenePlayerPhaseResponse{State: sampleInteractionState("scene-active")}, nil
		},
	}

	args := []byte(`{
		"interaction":{"title":"Arrival","beats":[{"type":"fiction","text":"The caravan arrives."}]},
		"character_ids":["char-1"]
	}`)
	result, err := newInteractionTransportSession(stub).interactionOpenScenePlayerPhase(context.Background(), args)
	if err != nil {
		t.Fatalf("interactionOpenScenePlayerPhase() error = %v", err)
	}
	if decoded := decodeInteractionToolResult(t, result.Output); decoded.ActiveScene.SceneID != "scene-active" {
		t.Fatalf("active scene = %q, want scene-active", decoded.ActiveScene.SceneID)
	}
}

func TestInteractionResolveScenePlayerReview(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		args  string
		check func(t *testing.T, req *statev1.ResolveScenePlayerReviewRequest)
	}{
		{
			name: "open next player phase",
			args: `{"open_next_player_phase":{"next_character_ids":["char-1"],"interaction":{"title":"Continue","beats":[{"type":"prompt","text":"What do you do?"}]}}}`,
			check: func(t *testing.T, req *statev1.ResolveScenePlayerReviewRequest) {
				t.Helper()
				resolution := req.GetOpenNextPlayerPhase()
				if resolution == nil || resolution.GetInteraction().GetTitle() != "Continue" || len(resolution.GetNextCharacterIds()) != 1 {
					t.Fatalf("resolution = %#v", req.GetResolution())
				}
			},
		},
		{
			name: "request revisions",
			args: `{"request_revisions":{"interaction":{"title":"Revise","beats":[{"type":"guidance","text":"Clarify the risk."}]},"revisions":[{"participant_id":"part-1","reason":"Need more detail","character_ids":["char-1"]}]}}`,
			check: func(t *testing.T, req *statev1.ResolveScenePlayerReviewRequest) {
				t.Helper()
				resolution := req.GetRequestRevisions()
				if resolution == nil || len(resolution.GetRevisions()) != 1 || resolution.GetRevisions()[0].GetParticipantId() != "part-1" {
					t.Fatalf("resolution = %#v", req.GetResolution())
				}
			},
		},
		{
			name: "return to gm",
			args: `{"return_to_gm":{"interaction":{"title":"GM Turn","beats":[{"type":"resolution","text":"The torch gutters out."}]}}}`,
			check: func(t *testing.T, req *statev1.ResolveScenePlayerReviewRequest) {
				t.Helper()
				resolution := req.GetReturnToGm()
				if resolution == nil || resolution.GetInteraction().GetTitle() != "GM Turn" {
					t.Fatalf("resolution = %#v", req.GetResolution())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := &interactionTransportClientStub{
				getInteractionStateFunc: func(_ context.Context, _ *statev1.GetInteractionStateRequest, _ ...grpc.CallOption) (*statev1.GetInteractionStateResponse, error) {
					return &statev1.GetInteractionStateResponse{State: sampleInteractionState("scene-review")}, nil
				},
				resolveScenePlayerReviewFunc: func(_ context.Context, req *statev1.ResolveScenePlayerReviewRequest, _ ...grpc.CallOption) (*statev1.ResolveScenePlayerReviewResponse, error) {
					if req.GetCampaignId() != "camp-ctx" || req.GetSceneId() != "scene-review" {
						t.Fatalf("request scope = %#v", req)
					}
					tt.check(t, req)
					return &statev1.ResolveScenePlayerReviewResponse{State: sampleInteractionState("scene-review")}, nil
				},
			}

			result, err := newInteractionTransportSession(stub).interactionResolveScenePlayerReview(context.Background(), []byte(tt.args))
			if err != nil {
				t.Fatalf("interactionResolveScenePlayerReview() error = %v", err)
			}
			if decoded := decodeInteractionToolResult(t, result.Output); decoded.ActiveScene.SceneID != "scene-review" {
				t.Fatalf("active scene = %q, want scene-review", decoded.ActiveScene.SceneID)
			}
		})
	}
}

func TestInteractionRecordSceneGMInteraction(t *testing.T) {
	t.Parallel()

	t.Run("blocks while gm review is pending", func(t *testing.T) {
		t.Parallel()

		stub := &interactionTransportClientStub{
			getInteractionStateFunc: func(_ context.Context, _ *statev1.GetInteractionStateRequest, _ ...grpc.CallOption) (*statev1.GetInteractionStateResponse, error) {
				return &statev1.GetInteractionStateResponse{State: &statev1.InteractionState{
					ActiveScene: &statev1.InteractionScene{SceneId: "scene-review"},
					PlayerPhase: &statev1.ScenePlayerPhase{Status: statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW},
					Ooc:         &statev1.OOCState{},
				}}, nil
			},
		}

		_, err := newInteractionTransportSession(stub).interactionRecordSceneGMInteraction(context.Background(), []byte(`{"interaction":{"title":"GM Turn","beats":[{"type":"fiction","text":"The gate splinters."}]}}`))
		if err == nil || err.Error() != "scene is waiting on gm review; use interaction_resolve_scene_player_review instead of interaction_record_scene_gm_interaction" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("records gm interaction on active scene", func(t *testing.T) {
		t.Parallel()

		stub := &interactionTransportClientStub{
			getInteractionStateFunc: func(_ context.Context, _ *statev1.GetInteractionStateRequest, _ ...grpc.CallOption) (*statev1.GetInteractionStateResponse, error) {
				return &statev1.GetInteractionStateResponse{State: sampleInteractionState("scene-gm")}, nil
			},
			recordSceneGMInteractionFunc: func(_ context.Context, req *statev1.RecordSceneGMInteractionRequest, _ ...grpc.CallOption) (*statev1.RecordSceneGMInteractionResponse, error) {
				if req.GetSceneId() != "scene-gm" || req.GetInteraction().GetTitle() != "GM Turn" {
					t.Fatalf("request = %#v", req)
				}
				return &statev1.RecordSceneGMInteractionResponse{State: sampleInteractionState("scene-gm")}, nil
			},
		}

		result, err := newInteractionTransportSession(stub).interactionRecordSceneGMInteraction(context.Background(), []byte(`{"interaction":{"title":"GM Turn","beats":[{"type":"fiction","text":"The gate splinters."}]}}`))
		if err != nil {
			t.Fatalf("interactionRecordSceneGMInteraction() error = %v", err)
		}
		if decoded := decodeInteractionToolResult(t, result.Output); decoded.ActiveScene.SceneID != "scene-gm" {
			t.Fatalf("active scene = %q, want scene-gm", decoded.ActiveScene.SceneID)
		}
	})
}

func TestInteractionSessionOOCHandlers(t *testing.T) {
	t.Parallel()

	t.Run("pause", func(t *testing.T) {
		t.Parallel()

		stub := &interactionTransportClientStub{
			openSessionOOCFunc: func(_ context.Context, req *statev1.OpenSessionOOCRequest, _ ...grpc.CallOption) (*statev1.OpenSessionOOCResponse, error) {
				if req.GetCampaignId() != "camp-ctx" || req.GetReason() != "bio break" {
					t.Fatalf("request = %#v", req)
				}
				return &statev1.OpenSessionOOCResponse{State: sampleInteractionState("scene-1")}, nil
			},
		}
		if _, err := newInteractionTransportSession(stub).interactionPauseOOC(context.Background(), []byte(`{"reason":"bio break"}`)); err != nil {
			t.Fatalf("interactionPauseOOC() error = %v", err)
		}
	})

	t.Run("resolve variants", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			args  string
			check func(t *testing.T, req *statev1.ResolveSessionOOCRequest)
		}{
			{
				name: "resume interrupted phase",
				args: `{"resume_interrupted_phase":true}`,
				check: func(t *testing.T, req *statev1.ResolveSessionOOCRequest) {
					t.Helper()
					if req.GetResumeInterruptedPhase() == nil {
						t.Fatalf("resolution = %#v", req.GetResolution())
					}
				},
			},
			{
				name: "return to gm",
				args: `{"return_to_gm":{"scene_id":"scene-return"}}`,
				check: func(t *testing.T, req *statev1.ResolveSessionOOCRequest) {
					t.Helper()
					if req.GetReturnToGm() == nil || req.GetReturnToGm().GetSceneId() != "scene-return" {
						t.Fatalf("resolution = %#v", req.GetResolution())
					}
				},
			},
			{
				name: "open player phase",
				args: `{"open_player_phase":{"scene_id":"scene-open","character_ids":["char-1"],"interaction":{"title":"Back In","beats":[{"type":"prompt","text":"You hear boots on stone."}]}}}`,
				check: func(t *testing.T, req *statev1.ResolveSessionOOCRequest) {
					t.Helper()
					if req.GetOpenPlayerPhase() == nil || req.GetOpenPlayerPhase().GetInteraction().GetTitle() != "Back In" || len(req.GetOpenPlayerPhase().GetNextCharacterIds()) != 1 {
						t.Fatalf("resolution = %#v", req.GetResolution())
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				stub := &interactionTransportClientStub{
					resolveSessionOOCFunc: func(_ context.Context, req *statev1.ResolveSessionOOCRequest, _ ...grpc.CallOption) (*statev1.ResolveSessionOOCResponse, error) {
						if req.GetCampaignId() != "camp-ctx" {
							t.Fatalf("campaign id = %q, want camp-ctx", req.GetCampaignId())
						}
						tt.check(t, req)
						return &statev1.ResolveSessionOOCResponse{State: sampleInteractionState("scene-1")}, nil
					},
				}
				if _, err := newInteractionTransportSession(stub).interactionResolveSessionOOC(context.Background(), []byte(tt.args)); err != nil {
					t.Fatalf("interactionResolveSessionOOC() error = %v", err)
				}
			})
		}
	})

	t.Run("post and ready state mutations", func(t *testing.T) {
		t.Parallel()

		stub := &interactionTransportClientStub{
			postSessionOOCFunc: func(_ context.Context, req *statev1.PostSessionOOCRequest, _ ...grpc.CallOption) (*statev1.PostSessionOOCResponse, error) {
				if req.GetBody() != "Need five minutes." {
					t.Fatalf("post request = %#v", req)
				}
				return &statev1.PostSessionOOCResponse{State: sampleInteractionState("scene-1")}, nil
			},
			markOOCReadyToResumeFunc: func(_ context.Context, req *statev1.MarkOOCReadyToResumeRequest, _ ...grpc.CallOption) (*statev1.MarkOOCReadyToResumeResponse, error) {
				if req.GetCampaignId() != "camp-ctx" {
					t.Fatalf("mark request = %#v", req)
				}
				return &statev1.MarkOOCReadyToResumeResponse{State: sampleInteractionState("scene-1")}, nil
			},
			clearOOCReadyToResumeFunc: func(_ context.Context, req *statev1.ClearOOCReadyToResumeRequest, _ ...grpc.CallOption) (*statev1.ClearOOCReadyToResumeResponse, error) {
				if req.GetCampaignId() != "camp-ctx" {
					t.Fatalf("clear request = %#v", req)
				}
				return &statev1.ClearOOCReadyToResumeResponse{State: sampleInteractionState("scene-1")}, nil
			},
		}

		session := newInteractionTransportSession(stub)
		if _, err := session.interactionPostOOC(context.Background(), []byte(`{"body":"Need five minutes."}`)); err != nil {
			t.Fatalf("interactionPostOOC() error = %v", err)
		}
		if _, err := session.interactionMarkOOCReady(context.Background(), []byte(`{}`)); err != nil {
			t.Fatalf("interactionMarkOOCReady() error = %v", err)
		}
		if _, err := session.interactionClearOOCReady(context.Background(), []byte(`{}`)); err != nil {
			t.Fatalf("interactionClearOOCReady() error = %v", err)
		}
	})
}

func TestInteractionConcludeSession(t *testing.T) {
	t.Parallel()

	t.Run("forwards campaign end request and persists epilogue artifact", func(t *testing.T) {
		t.Parallel()

		artifactManager := &artifactManagerTestStub{}
		campaignAI := &campaignAITransportClientStub{
			concludeSessionFunc: func(_ context.Context, req *statev1.ConcludeSessionRequest, _ ...grpc.CallOption) (*statev1.ConcludeSessionResponse, error) {
				if req.GetCampaignId() != "camp-ctx" || req.GetSessionId() != "sess-ctx" {
					t.Fatalf("request scope = %#v", req)
				}
				if !req.GetEndCampaign() {
					t.Fatal("end_campaign = false, want true")
				}
				if req.GetEpilogue() != "The harbor rebuilds in peace." {
					t.Fatalf("epilogue = %q", req.GetEpilogue())
				}
				return &statev1.ConcludeSessionResponse{
					SessionId:         "sess-ctx",
					EndedSceneIds:     []string{"scene-1", "scene-2"},
					CampaignCompleted: true,
				}, nil
			},
		}

		session := NewDirectSession(Clients{
			CampaignAI: campaignAI,
			Artifact:   artifactManager,
		}, SessionContext{
			CampaignID:    "camp-ctx",
			SessionID:     "sess-ctx",
			ParticipantID: "gm-1",
		})

		result, err := session.interactionConcludeSession(context.Background(), []byte(`{
			"conclusion":"The tide finally settles.",
			"summary":"## Key Events\n\nThe gate held.\n\n## NPCs Met\n\nCaptain Vale.\n\n## Decisions Made\n\nThey ended the war.\n\n## Unresolved Threads\n\nWho financed the raiders?\n\n## Next Session Hooks\n\nCelebrate at dawn.",
			"end_campaign":true,
			"epilogue":"The harbor rebuilds in peace."
		}`))
		if err != nil {
			t.Fatalf("interactionConcludeSession() error = %v", err)
		}

		payload := decodeToolOutput[interactionConcludeSessionResult](t, result.Output)
		if !payload.CampaignCompleted {
			t.Fatal("campaign_completed = false, want true")
		}
		if artifactManager.lastPath != "epilogue.md" {
			t.Fatalf("artifact path = %q, want epilogue.md", artifactManager.lastPath)
		}
		if artifactManager.lastContent != "The harbor rebuilds in peace." {
			t.Fatalf("artifact content = %q", artifactManager.lastContent)
		}
	})

	t.Run("uses service response for campaign_completed", func(t *testing.T) {
		t.Parallel()

		session := NewDirectSession(Clients{
			CampaignAI: &campaignAITransportClientStub{
				concludeSessionFunc: func(_ context.Context, _ *statev1.ConcludeSessionRequest, _ ...grpc.CallOption) (*statev1.ConcludeSessionResponse, error) {
					return &statev1.ConcludeSessionResponse{
						SessionId:         "sess-ctx",
						EndedSceneIds:     []string{"scene-1"},
						CampaignCompleted: false,
					}, nil
				},
			},
		}, SessionContext{
			CampaignID: "camp-ctx",
			SessionID:  "sess-ctx",
		})

		result, err := session.interactionConcludeSession(context.Background(), []byte(`{
			"conclusion":"The group camps for the night.",
			"summary":"## Key Events\n\nThey survived.\n\n## NPCs Met\n\nCaptain Vale.\n\n## Decisions Made\n\nThey delayed the final choice.\n\n## Unresolved Threads\n\nWhat waits inland?\n\n## Next Session Hooks\n\nBegin the march.",
			"end_campaign":false
		}`))
		if err != nil {
			t.Fatalf("interactionConcludeSession() error = %v", err)
		}

		payload := decodeToolOutput[interactionConcludeSessionResult](t, result.Output)
		if payload.CampaignCompleted {
			t.Fatal("campaign_completed = true, want false")
		}
	})
}

func TestInteractionHandlersWrapTransportErrors(t *testing.T) {
	t.Parallel()

	stub := &interactionTransportClientStub{
		activateSceneFunc: func(context.Context, *statev1.ActivateSceneRequest, ...grpc.CallOption) (*statev1.ActivateSceneResponse, error) {
			return nil, errors.New("boom")
		},
	}

	_, err := newInteractionTransportSession(stub).interactionActivateScene(context.Background(), []byte(`{"scene_id":"scene-2"}`))
	if err == nil || err.Error() != "set active scene failed: boom" {
		t.Fatalf("error = %v", err)
	}
}
