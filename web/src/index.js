import $ from 'jquery'
import fetcher from './fetcher';
import SocketConstants from './socket/constants';
import GameSocket from './socket/GameSocket'
import { boardMatrixToObj } from './chess-helpers';

// window['jQuery'] is required for the chessboard to work (sadly)
// ordering is also important
window['jQuery'] = $;
require('oakmac-chessboard');

const boardConfig = {
  draggable: true,
  position: '',
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
  fetcher.post(`http://${window.location.host}/api/game?command=start`)
  .then(response => {
    gameSocket = new GameSocket(messageHandler);
    console.log(response);
  })
  .catch(err => {
    console.error(err);
  })
});

function messageHandler(event) {
  const message = JSON.parse(event.data);
  const data = JSON.parse(message.data);

  console.log('Received Data:', data);

  switch (message.type) {
    case SocketConstants.GameState:
      // TODO: Update rest of other states (num of moves, etc..)
      board.position(boardMatrixToObj(data.currentBoard), false);
      board.orientation(data.humanColor.toLowerCase());
      break;
    case SocketConstants.GameFull:
      // TODO: Update UI
      alert('Game is currently in progress!');
      break;
    default:
      return;
  }
}
