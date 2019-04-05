window['jQuery'] = require('jquery');
require('oakmac-chessboard');

const config = {
  draggable: true,
  position: 'start',
};

const board = ChessBoard('board', config);
