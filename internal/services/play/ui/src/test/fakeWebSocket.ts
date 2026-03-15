type SocketEventMap = {
  close: CloseEvent;
  error: Event;
  message: MessageEvent;
  open: Event;
};

type SocketEventType = keyof SocketEventMap;
type SocketListener<T extends SocketEventType> = (event: SocketEventMap[T]) => void;

export class MockWebSocket {
  static instances: MockWebSocket[] = [];

  readonly sent: string[] = [];
  readonly url: string;
  private readonly listeners: {
    [K in SocketEventType]: Set<SocketListener<K>>;
  } = {
    close: new Set(),
    error: new Set(),
    message: new Set(),
    open: new Set(),
  };

  constructor(url: string | URL) {
    this.url = String(url);
    MockWebSocket.instances.push(this);
  }

  static reset(): void {
    MockWebSocket.instances = [];
  }

  addEventListener<T extends SocketEventType>(type: T, listener: SocketListener<T>): void {
    this.listeners[type].add(listener);
  }

  removeEventListener<T extends SocketEventType>(type: T, listener: SocketListener<T>): void {
    this.listeners[type].delete(listener);
  }

  close(): void {
    this.emitClose();
  }

  send(payload: string): void {
    this.sent.push(payload);
  }

  emitClose(init?: CloseEventInit): void {
    this.dispatch("close", new CloseEvent("close", init));
  }

  emitError(): void {
    this.dispatch("error", new Event("error"));
  }

  emitMessage(data: string): void {
    this.dispatch("message", new MessageEvent("message", { data }));
  }

  emitOpen(): void {
    this.dispatch("open", new Event("open"));
  }

  private dispatch<T extends SocketEventType>(type: T, event: SocketEventMap[T]): void {
    for (const listener of this.listeners[type]) {
      listener(event);
    }
  }
}
