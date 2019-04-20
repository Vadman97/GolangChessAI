import $ from 'jquery'
import SocketConstants from './socket/constants';
import GameSocket from './socket/GameSocket'

// window['jQuery'] is required for the chessboard to work (sadly)
// ordering is also important
window['jQuery'] = $;
require('oakmac-chessboard');

const boardConfig = {
  draggable: true,
  position: 'start',
  pieceTheme: 'img/chesspieces/wikipedia-svg/{piece}.svg'
};
const board = ChessBoard('board', boardConfig);

let gameSocket;

// BUTTON FOR POST REQUEST
// CrEATE SOCKET
// RECEIVE GAME STATUS
// SET BOARD
// SEND PLAYER MOVE

$(document).ready(() => {

  // gameSocket.send(SocketConstants.PlayerMove, {
  //   start: [0, 0],
  //   end: [0, 1],
  //   isCapture: false,
  //   piece: null,
  // });
});

$("#start-btn").click(() => {
  $.post(`${window.location.host}/api/game?command=start`)
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
