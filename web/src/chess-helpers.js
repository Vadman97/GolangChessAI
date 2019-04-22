const BOARD_SIZE = 8;

// Could use String.fromCharCode but this is faster
const numberToChar = {
  0: 'a',
  1: 'b',
  2: 'c',
  3: 'd',
  4: 'e',
  5: 'f',
  6: 'g',
  7: 'h',
};

const colorToChar = {
  Black: 'b',
  White: 'w',
};

/*
  Converts a board state received from the server to a object parsable by the chessboard gui
*/
export function boardMatrixToObj(boardMatrix) {
  const boardObj = {};

  for (let r = 0; r < BOARD_SIZE; ++r) {
    for (let c = 0; c < BOARD_SIZE; ++c) {
      const piece = boardMatrix[r][c];
      if (!piece) {
        continue;
      }

      // NOTE: Row needs to be inverted since the top starts from Row 8
      const pieceStr = `${colorToChar[piece.color]}${piece.type}`;
      boardObj[`${numberToChar[c]}${7 - r + 1}`] = pieceStr;
    }
  }

  return boardObj;
}
