import $ from 'jquery'
import SocketConstants from './socket/constants';
import GameSocket from './socket/GameSocket'

// window['jQuery'] is required for the chessboard to work (sadly)
// ordering is also important
window['jQuery'] = $;
require('oakmac-chessboard');

const config = {
  draggable: true,
  position: 'start',
};
const board = ChessBoard('board', config);

$(document).ready(() => {
  const gameSocket = new GameSocket(messageHandler);
  gameSocket.send(SocketConstants.PlayerMove, {
    start: [0, 0],
    end: [0, 1],
    isCapture: false,
    piece: null,
  });
});

function messageHandler(event) {
  const message = JSON.parse(event.data);
  switch (message.type) {
    case SocketConstants.GameFull:
      // TODO: Update UI
      alert('Game is currently in progress!');
    default:
      return;
  }
}
