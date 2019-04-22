import $ from 'jquery'
import fetcher from './fetcher';
import SocketConstants from './socket/constants';
import GameSocket from './socket/GameSocket'
import { rowColToChess, chessToRowCol, boardMatrixToObj } from './chess-helpers';

// window['jQuery'] is required for the chessboard to work (sadly)
// ordering is also important
window['jQuery'] = $;
require('oakmac-chessboard');

const boardConfig = {
  draggable: true,
  pieceTheme: 'img/chesspieces/wikipedia-svg/{piece}.svg',
  onDragStart: onChessboardDragStart,
  onDrop: onChessboardDrop,
};
const board = ChessBoard('board', boardConfig);

let gameSocket;
let availableMoves;

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
      // NOTE: Our server records black on the bottom, and white on the top
      board.flip();
      break;
    case SocketConstants.AvailablePlayerMoves:
      availableMoves = data.availableMoves;
      break;
    case SocketConstants.GameFull:
      // TODO: Update UI
      alert('Game is currently in progress!');
      break;
    default:
      return;
  }
}

function clearBoard() {
  $('.square-highlight-move').removeClass('square-highlight-move');
  $('.square-active').removeClass('square-active');
}

/* Chessboard Events */
function onChessboardDragStart(source, piece) {
  clearBoard();

  const sourceCoord = chessToRowCol(source);
  const movesForPiece = availableMoves[`(${sourceCoord[0]}, ${sourceCoord[1]})`];

  $('.square-' + source).addClass('square-active');

  if (!movesForPiece) {
    return;
  }

  movesForPiece.forEach(move => {
    const endChessLoc = rowColToChess(move.end[0], move.end[1]);
    $('.square-' + endChessLoc).addClass('square-highlight-move');
  });
}

function onChessboardDrop(source, target, piece) {
  if (target !== source) {
    clearBoard();
  }
}
