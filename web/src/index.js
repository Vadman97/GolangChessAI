import $ from 'jquery'
import Popper from 'popper.js';
import fetcher from './fetcher';
import Game, {GameStatus} from './Game';
import SocketConstants from './socket/constants';
import GameSocket from './socket/GameSocket'
import {BOARD_SIZE, boardMatrixToObj, charToColor, chessToRowCol, colorToChar, rowColToChess} from './chess-helpers';

// window['jQuery'] is required for the chessboard to work (sadly)
// ordering is also important
window['jQuery'] = $;
require('oakmac-chessboard');

const boardConfig = {
  draggable: true,
  moveSpeed: 'slow',
  pieceTheme: 'img/chesspieces/wikipedia-svg/{piece}.svg',
  onDragStart: onChessboardDragStart,
  onDrop: onChessboardDrop,
};
const board = ChessBoard('board', boardConfig);


let pawnPromotionPopper;
let promotionMove;

let game;
let gameSocket;
let availableMoves;

// TODO: Keep track of other stats
// CurrentTurn, MoveCount, Think Time

// Initial UI
setup();

$(document).ready(() => {
  // const reference = document.querySelector('.test');
  // const popper = document.querySelector('.popper');
  // pawnPromotionPopper = new Popper(reference, popper, {
  //   placement: 'top',
  // });
});

/* UI Functions */
function setup() {
  $('.pawn-promotion').hide();
  $('.white-promotion').hide();
  $('.black-promotion').hide();
  $('#concede-btn').hide();
  $('.chessboard-63f37').addClass('inactive');
  $('.game-error').hide();
  $('.game-status').hide();
  $('.game-status .status-alert').hide();
  $('.ai-thinking').hide();
}

function updateGameStatus() {
  $('.game-status .status span').text(game.status);
  $('.game-status .current-turn span').text(game.currentTurn);
  $('.game-status .moves-played span').text(game.movesPlayed);
  $('.game-status .move-limit span').text(game.moveLimit);

  // Show AI Thinking Animation
  if (game.currentTurn.toLowerCase() !== game.humanColor) {
    $('.ai-thinking').show();
  }
  else {
    $('.ai-thinking').hide();
  }

  // Update Game based on Game Status (Active, Stalemate)
  if (game.status !== GameStatus.Active) {
    let alertText;

    switch (game.status) {
      case GameStatus.WhiteWin:
      case GameStatus.BlackWin:
        alertText = 'Checkmate!';
        break;
      case GameStatus.Stalemate:
      case GameStatus.FiftyMoveDraw:
      case GameStatus.RepeatActionDraw:
        alertText = 'Draw';
        break;
      case GameStatus.Aborted:
        alertText = 'Aborted';
    }

    $('.game-status .status-alert').text(alertText).show();
    $('.chessboard-63f37').addClass('inactive');
    $('.ai-thinking').hide();

    window.speechSynthesis.speak(new SpeechSynthesisUtterance(alertText));
    if (game.status !== GameStatus.Aborted) {
      window.speechSynthesis.speak(new SpeechSynthesisUtterance(game.status));
    }
  }
}

function clearBoard() {
  $('.square-highlight-move').removeClass('square-highlight-move');
  $('.square-active').removeClass('square-active');
}

/* Button Events */
$('#start-btn').click(() => {
  fetcher.post(`http://${window.location.host}/api/game?command=start`)
  .then(response => {
    gameSocket = new GameSocket(messageHandler);

    $('.game-status').show();
    $('.game-error').text('').hide();
    $('#concede-btn').show();
    $('#start-btn').hide();
    $('.chessboard-63f37').removeClass('inactive');
    console.log(response);
  })
  .catch(err => {
    $('.game-error').text(err.error).show();
    console.error(err);
  })
});

$('.promotion-piece').click((event) => {
  if (!promotionMove) {
    console.warn('Promotion was called without a move saved');
    return;
  }

  // Send Move with Promotion Piece over Socket
  const {piece} = event.target.dataset;
  promotionMove.promotionPiece = {
    color: charToColor[piece[0]],
    type: piece[1],
  };
  gameSocket.send(SocketConstants.PlayerMove, promotionMove);

  // Update Promotion Piece on Game Board
  const endLoc = rowColToChess(promotionMove.end[0], promotionMove.end[1]);
  const currentBoard = board.position();
  board.position({
    ...currentBoard,
    [endLoc]: piece,
  }, false);

  // Clean up
  promotionMove = null;
  $('.pawn-promotion').hide();
});

function messageHandler(event) {
  const message = JSON.parse(event.data);
  const data = JSON.parse(message.data);

  console.log('Received Data:', data);

  switch (message.type) {
    case SocketConstants.GameState:
      game = new Game(
        data.humanColor.toLowerCase(),
        data.gameStatus,
        data.moveLimit,
        data.timeLimit,
      );
      game.currentTurn = data.currentTurn;
      game.movesPlayed = data.movesPlayed;

      board.position(boardMatrixToObj(data.currentBoard), false);
      board.orientation(game.humanColor);
      // DEPRECATED: Our server records black on the bottom, and white on the top
      // board.flip();
      updateGameStatus();
      break;

    case SocketConstants.GameStatus:
      game.currentTurn = data.currentTurn;
      game.movesPlayed = data.movesPlayed;
      game.status = data.gameStatus;

      if (game.status === GameStatus.Active && data.kingInCheck) {
        $('.game-status .status-alert').text('Check!').show();
        window.speechSynthesis.speak(new SpeechSynthesisUtterance('Check'));
      }
      else {
        $('.game-status .status-alert').hide();
      }

      updateGameStatus();
      break;

    case SocketConstants.AvailablePlayerMoves:
      availableMoves = data.availableMoves;
      break;

    case SocketConstants.AIMove:
      makeAIMove(data.start, data.end, data.piece, data.promotionPiece);
      break;

    case SocketConstants.GameFull:
      $('.game-error').text('A game is currently in progress...')
      $("#start-btn").attr("disabled", true);
      break;

    default:
      return;
  }
}

function makeAIMove(start, end, piece, promotionPiece) {
  if (promotionPiece.type && promotionPiece.color) {
    const endLoc = rowColToChess(end[0], end[1]);
    const currentBoard = board.position();
    board.position({
      ...currentBoard,
      [endLoc]: `${colorToChar[promotionPiece.color.toLowerCase()]}${promotionPiece.type}`,
    }, false);

    window.speechSynthesis.speak(new SpeechSynthesisUtterance('Pawn Promotion'));
    return;
  }
  // Check if it's a Castle Move (King and moved 2 columns)
  if (piece.type === 'K') {
    // Queen-side Castle
    if (end[1] - start[1] === 2) {
      const rookStartLoc = rowColToChess(end[0], end[1] + 2);
      const rookEndLoc = rowColToChess(end[0], end[1] - 1);
      setTimeout(() => board.move(`${rookStartLoc}-${rookEndLoc}`), 150);
      window.speechSynthesis.speak(new SpeechSynthesisUtterance('Queen-side Castle'));
    }
    // King-side Castle
    else if (start[1] - end[1] === 2) {
      const rookStartLoc = rowColToChess(end[0], end[1] - 1);
      const rookEndLoc = rowColToChess(end[0], end[1] + 1);
      setTimeout(() => board.move(`${rookStartLoc}-${rookEndLoc}`), 150);
      window.speechSynthesis.speak(new SpeechSynthesisUtterance('King-side Castle'));
    }
  }
  const startChessLoc = rowColToChess(start[0], start[1]);
  const endChessLoc = rowColToChess(end[0], end[1]);
  // For some weird reason, timing out the move fixes a UI glitch
  setTimeout(() => board.move(`${startChessLoc}-${endChessLoc}`), 150);
  window.speechSynthesis.speak(new SpeechSynthesisUtterance(`${startChessLoc} to ${endChessLoc}`));
}

/* Chessboard Events */
function onChessboardDragStart(source, piece) {
  clearBoard();

  if (
    colorToChar[game.humanColor] !== piece[0] ||
    game.currentTurn.toLowerCase() !== game.humanColor
  ) {
    return false;
  }

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
  // Validate Move
  const sourceCoord = chessToRowCol(source);
  const targetCoord = chessToRowCol(target);
  const movesForPiece = availableMoves[`(${sourceCoord[0]}, ${sourceCoord[1]})`];

  let isValidMove = false;
  if (movesForPiece) {
    movesForPiece.forEach(move => {
      if (targetCoord[0] == move.end[0] && targetCoord[1] == move.end[1]) {
        isValidMove = true;
      }
    });
  }

  if (!isValidMove) {
    return 'snapback';
  }

  if (target !== source) {
    clearBoard();
  }

  // Check if it's a Castle Move (King and moved 2 columns)
  if (piece[1] == 'K') {
    // Queen-side Castle
    if (targetCoord[1] - sourceCoord[1] == 2) {
      const rookStartLoc = rowColToChess(targetCoord[0], targetCoord[1] + 2);
      const rookEndLoc = rowColToChess(targetCoord[0], targetCoord[1] - 1);
      setTimeout(() => board.move(`${rookStartLoc}-${rookEndLoc}`), 150);
      window.speechSynthesis.speak(new SpeechSynthesisUtterance('Queen-side Castle'));
    }
    // King-side Castle
    else if (sourceCoord[1] - targetCoord[1] == 2) {
      const rookStartLoc = rowColToChess(targetCoord[0], targetCoord[1] - 1);
      const rookEndLoc = rowColToChess(targetCoord[0], targetCoord[1] + 1);
      setTimeout(() => board.move(`${rookStartLoc}-${rookEndLoc}`), 150);
      window.speechSynthesis.speak(new SpeechSynthesisUtterance('King-side Castle'));
    }
  }

  // Check for Pawn Promotion
  if (piece[1] == 'P') {
    // NOTE: Don't need any other checks since pawns can only move forward
    if (targetCoord[0] == 0 || targetCoord[0] == BOARD_SIZE - 1) {
      console.log('Pawn Promotion');

      // Show Popper (Popover)
      const reference = document.querySelector('.square-' + target);
      const popper = document.querySelector('.popper');
      pawnPromotionPopper = new Popper(reference, popper, {
        placement: 'top',
      });

      $(`.${game.humanColor.toLowerCase()}-promotion`).show();
      $('.pawn-promotion').show();

      // Save Move Data (before we lose it)
      promotionMove = {
        start: sourceCoord,
        end: targetCoord,
      };

      return;
    }
  }

  gameSocket.send(SocketConstants.PlayerMove, {
    start: sourceCoord,
    end: targetCoord,
  });
}
