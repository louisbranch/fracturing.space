package seed

import "time"

// fakeMCPClient satisfies mcpClient with injectable function fields.
type fakeMCPClient struct {
	writeMessage      func(message any) error
	readResponseForID func(requestID any, timeout time.Duration) (any, []byte, error)
	closed            bool
}

func (f *fakeMCPClient) WriteMessage(message any) error {
	if f.writeMessage != nil {
		return f.writeMessage(message)
	}
	return nil
}

func (f *fakeMCPClient) ReadResponseForID(requestID any, timeout time.Duration) (any, []byte, error) {
	if f.readResponseForID != nil {
		return f.readResponseForID(requestID, timeout)
	}
	return map[string]any{"id": requestID}, nil, nil
}

func (f *fakeMCPClient) Close() {
	f.closed = true
}
