import SocketConstants from './constants';

class GameSocket {
  constructor(messageHandler) {
    this.socket = new WebSocket(`ws://${window.location.host}/ws`);
    this.socket.onopen = this.onOpen;
    this.socket.onmessage = messageHandler || this.onReceiveMessage;
    this.socket.onclose = this.onClose;

    this.messageQueue = [];
    this.closed = false;
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

  // WebSocket EventHandlers
  onOpen = () => {
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
    console.warn('Socket Closed!');
    this.closed = true;
  }
}

export default GameSocket;
