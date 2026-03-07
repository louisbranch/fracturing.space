package server

import (
	"context"
	"strings"
	"sync"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
)

const campaignAITurnSubscriptionRetryDelay = time.Second

type campaignAITurnSubscriptionWorker struct {
	ctx              context.Context
	invocationClient aiv1.InvocationServiceClient
	roomHub          *roomHub

	mu          sync.Mutex
	subscribers map[string]context.CancelFunc
	wg          sync.WaitGroup
}

func startCampaignAITurnSubscriptionWorker(ctx context.Context, invocationClient aiv1.InvocationServiceClient, roomHub *roomHub) (func(string, string, string), func(string), context.CancelFunc, chan struct{}) {
	if ctx == nil || invocationClient == nil || roomHub == nil {
		return nil, nil, nil, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	worker := &campaignAITurnSubscriptionWorker{
		ctx:              ctx,
		invocationClient: invocationClient,
		roomHub:          roomHub,
		subscribers:      make(map[string]context.CancelFunc),
	}
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		worker.mu.Lock()
		for _, subCancel := range worker.subscribers {
			subCancel()
		}
		worker.mu.Unlock()
		worker.wg.Wait()
		close(done)
	}()

	return worker.ensureCampaignSubscription, worker.releaseCampaignSubscription, cancel, done
}

func (w *campaignAITurnSubscriptionWorker) ensureCampaignSubscription(campaignID string, _ string, _ string) {
	if w == nil {
		return
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}

	w.mu.Lock()
	if _, exists := w.subscribers[campaignID]; exists {
		w.mu.Unlock()
		return
	}
	subCtx, subCancel := context.WithCancel(w.ctx)
	w.subscribers[campaignID] = subCancel
	w.wg.Add(1)
	w.mu.Unlock()

	go func() {
		defer w.wg.Done()
		consumeCampaignAITurnUpdates(subCtx, w.invocationClient, w.roomHub, campaignID)
	}()
}

func (w *campaignAITurnSubscriptionWorker) releaseCampaignSubscription(campaignID string) {
	if w == nil {
		return
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}

	w.mu.Lock()
	cancel, exists := w.subscribers[campaignID]
	if exists {
		delete(w.subscribers, campaignID)
	}
	w.mu.Unlock()
	if exists {
		cancel()
	}
}

func consumeCampaignAITurnUpdates(ctx context.Context, invocationClient aiv1.InvocationServiceClient, roomHub *roomHub, campaignID string) {
	if invocationClient == nil || roomHub == nil {
		return
	}

	afterSequenceID := uint64(0)
	for {
		if ctx.Err() != nil {
			return
		}

		room := roomHub.roomIfExists(campaignID)
		if room == nil || !room.aiRelayReady() {
			if !waitCampaignAITurnSubscriptionRetry(ctx, campaignAITurnSubscriptionRetryDelay) {
				return
			}
			continue
		}
		stream, err := invocationClient.SubscribeCampaignTurnEvents(ctx, &aiv1.SubscribeCampaignTurnEventsRequest{
			CampaignId:      campaignID,
			AfterSequenceId: afterSequenceID,
			SessionGrant:    room.aiSessionGrantValue(),
		})
		if err != nil {
			if !waitCampaignAITurnSubscriptionRetry(ctx, campaignAITurnSubscriptionRetryDelay) {
				return
			}
			continue
		}

		for {
			update, recvErr := stream.Recv()
			if recvErr != nil {
				break
			}
			if update == nil {
				continue
			}

			if update.GetSequenceId() > afterSequenceID {
				afterSequenceID = update.GetSequenceId()
			}
			if !update.GetParticipantVisible() {
				continue
			}
			content := strings.TrimSpace(update.GetContent())
			if content == "" {
				continue
			}

			room := roomHub.roomIfExists(campaignID)
			if room == nil {
				continue
			}
			msg, duplicate, subscribers := room.appendAIGMMessage(update.GetSessionId(), content, update.GetCorrelationMessageId())
			if duplicate {
				continue
			}
			frame := wsFrame{
				Type:    "chat.message",
				Payload: mustJSON(messageEnvelope{Message: msg}),
			}
			for _, subscriber := range subscribers {
				_ = subscriber.writeFrame(frame)
			}
		}

		if !waitCampaignAITurnSubscriptionRetry(ctx, campaignAITurnSubscriptionRetryDelay) {
			return
		}
	}
}

func waitCampaignAITurnSubscriptionRetry(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		delay = time.Second
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
