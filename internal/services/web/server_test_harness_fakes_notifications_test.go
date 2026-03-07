package web

import (
	"context"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	"google.golang.org/grpc"
)

type fakeWebNotificationClient struct {
	listResp   *notificationsv1.ListNotificationsResponse
	listErr    error
	getResp    *notificationsv1.GetNotificationResponse
	getErr     error
	markResp   *notificationsv1.MarkNotificationReadResponse
	markErr    error
	unreadResp *notificationsv1.GetUnreadNotificationStatusResponse
	unreadErr  error
}

func (f fakeWebNotificationClient) ListNotifications(context.Context, *notificationsv1.ListNotificationsRequest, ...grpc.CallOption) (*notificationsv1.ListNotificationsResponse, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &notificationsv1.ListNotificationsResponse{}, nil
}

func (f fakeWebNotificationClient) GetNotification(_ context.Context, req *notificationsv1.GetNotificationRequest, _ ...grpc.CallOption) (*notificationsv1.GetNotificationResponse, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getResp != nil {
		return f.getResp, nil
	}
	return &notificationsv1.GetNotificationResponse{Notification: &notificationsv1.Notification{Id: req.GetNotificationId()}}, nil
}

func (f fakeWebNotificationClient) GetUnreadNotificationStatus(context.Context, *notificationsv1.GetUnreadNotificationStatusRequest, ...grpc.CallOption) (*notificationsv1.GetUnreadNotificationStatusResponse, error) {
	if f.unreadErr != nil {
		return nil, f.unreadErr
	}
	if f.unreadResp != nil {
		return f.unreadResp, nil
	}
	return &notificationsv1.GetUnreadNotificationStatusResponse{}, nil
}

func (f fakeWebNotificationClient) MarkNotificationRead(_ context.Context, req *notificationsv1.MarkNotificationReadRequest, _ ...grpc.CallOption) (*notificationsv1.MarkNotificationReadResponse, error) {
	if f.markErr != nil {
		return nil, f.markErr
	}
	if f.markResp != nil {
		return f.markResp, nil
	}
	return &notificationsv1.MarkNotificationReadResponse{Notification: &notificationsv1.Notification{Id: req.GetNotificationId()}}, nil
}
