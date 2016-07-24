import { Observable } from 'rxjs/Observable';
import { Subject }    from 'rxjs/Subject';


const DEFAULT_CONNECT_TIMEOUT_INTERVAL = 5 * 1000;
const DEFAULT_RECONNECT_INTERVAL       = 1 * 1000;
const DEFAULT_RECONNECT_DECAY          = 2;
const DEFAULT_MAX_RECONNECT_INTERVAL   = 5 * 60 * 1000;


export interface State {
  readyState: number;
  metadata:   any;
}


export class ReconnectingWebSocket {

  private url:       string;
  private protocols: string[];

  private ws: WebSocket;

  private readyState: number = WebSocket.CLOSED;

  connectTimeoutInterval: number = DEFAULT_CONNECT_TIMEOUT_INTERVAL;
  reconnectInterval:      number = DEFAULT_RECONNECT_INTERVAL;
  reconnectDecay:         number = DEFAULT_RECONNECT_DECAY;
  maxReconnectInterval:   number = DEFAULT_MAX_RECONNECT_INTERVAL;

  private currentReconnectInterval:    number;
  private currentReconnectDecay:       number;
  private currentMaxReconnectInterval: number;

  private reconnectTimeout: any;

  private _state:    Subject<State>;
  private _messages: Subject<MessageEvent>;
  private _errors:   Subject<Event>;

  constructor(url: string, protocols: string[]) {
    this.url = url;
    this.protocols = protocols;

    this._state = new Subject<State>();
    this._messages = new Subject<MessageEvent>();
    this._errors = new Subject<Event>();
  }

  connect() : void {
    switch (this.readyState) {
      case WebSocket.CONNECTING:
        return;

      case WebSocket.OPEN:
        return;

      case WebSocket.CLOSING:
        break;

      case WebSocket.CLOSED:
        break;
    }

    this._connect(false);
  }

  private reconnect() : void {
    this._connect(true);
  }

  private resetReconnect() : void {
    this.currentReconnectInterval = this.reconnectInterval;
    this.currentReconnectDecay = this.reconnectDecay;
    this.currentMaxReconnectInterval = this.maxReconnectInterval;
  }

  private _connect(reconnecting: boolean) : void {
    if (!reconnecting) {
      this.resetReconnect();
    }

    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    const ws = new WebSocket(this.url, this.protocols);

    this.setState(WebSocket.CONNECTING);

    const connectTimeout = setTimeout(() => {
      ws.close();
    }, this.connectTimeoutInterval);

    ws.onopen = (ev: Event) => {
      clearTimeout(connectTimeout);
      this.setState(WebSocket.OPEN);
      this.resetReconnect();
    };

    ws.onmessage = (ev: MessageEvent) => {
      this._messages.next(ev);
    };

    ws.onerror = (ev: Event) => {
      this._errors.next(ev);
    };

    ws.onclose = (ev: CloseEvent) => {
      clearTimeout(connectTimeout);

      const readyState = this.readyState;

      this.ws = null;
      this.setState(WebSocket.CLOSED);

      if (readyState === WebSocket.CLOSING) {
        return;
      }

      this.reconnectTimeout = setTimeout(() => {
        this.reconnect();
      }, this.currentReconnectInterval);

      this.currentReconnectInterval = Math.min(
        this.currentReconnectInterval * this.currentReconnectDecay,
        this.currentMaxReconnectInterval
      );
    };

    this.ws = ws;
  }

  get state(): Observable<State> {
    return this._state;
  }

  get messages(): Observable<MessageEvent> {
    return this._messages;
  }

  get errors(): Observable<Event> {
    return this._errors;
  }

  send(data: any) : void {
    if (this.readyState === WebSocket.OPEN) {
      this.ws.send(data);
    } else {
      throw new Error('disconnected');
    }
  }

  close() : boolean {
    if (this.ws) {
      this.setState(WebSocket.CLOSING);
      this.ws.close();
      return true;
    }
    return false;
  }

  private setState(state: number) : void {
    const readyState = this.readyState;
    this.readyState = state;

    let metadata: any = {};
    if (state === WebSocket.CLOSED && readyState !== WebSocket.CLOSING) {
      metadata.reconnectInterval = this.currentReconnectInterval;
    }

    this._state.next({
      readyState: state,
      metadata: metadata
    });
  }
}
