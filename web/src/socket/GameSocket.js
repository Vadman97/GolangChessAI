import SocketConstants from './constants';

const RECONNECT_DELAY_MS = 2000;
const MAX_RECONNECT_ATTEMPTS = 5;

class GameSocket {
  constructor(messageHandler) {
    this.messageHandler = messageHandler;
    this.messageQueue = [];
    this.closed = false;
    this.reconnectAttempts = 0;
    this._connect();
  }

  _connect() {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    this.socket = new WebSocket(`${wsProtocol}//${window.location.host}/ws`);
    this.socket.onopen = this.onOpen;
    this.socket.onmessage = this.messageHandler || this.onReceiveMessage;
    this.socket.onclose = this.onClose;
  }

  send(type, data) {
    if (this.socket.readyState !== WebSocket.OPEN) {
      console.warn('Socket is currently closed or unable to connect');
      console.warn('Message has been queued to send');

      this.messageQueue.push({ type, data });
      return;
    }

    if (!Object.values(SocketConstants).includes(type)) {
      throw new Error('Socket Message Type invalid');
    }

    // Need to stringify the data again because that's the "best" way for the backend to parse it
    const payload = {
      type,
      data: JSON.stringify(data),
    };

    this.socket.send(JSON.stringify(payload));
  }

  close() {
    this.closed = true;
    this.socket.close();
  }

  // WebSocket EventHandlers
  onOpen = () => {
    this.reconnectAttempts = 0;
    while (this.messageQueue.length > 0) {
      const message = this.messageQueue.shift();
      this.send(message.type, message.data);
    }
  }

  onReceiveMessage = (msg) => {
    // Default Message Handler
    const data = JSON.parse(msg.data);
    console.log('Socket Recieved:', data);
  }

  onClose = () => {
    if (this.closed) return;

    console.warn('Socket Closed!');
    if (this.reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
      this.reconnectAttempts++;
      console.warn(`Reconnecting... attempt ${this.reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS}`);
      setTimeout(() => this._connect(), RECONNECT_DELAY_MS);
    } else {
      console.error('Max reconnect attempts reached. Please reload the page.');
      this.closed = true;
    }
  }
}

export default GameSocket;
