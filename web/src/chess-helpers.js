export const BOARD_SIZE = 8;

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

const charToNumber = {
  a: 0,
  b: 1,
  c: 2,
  d: 3,
  e: 4,
  f: 5,
  g: 6,
  h: 7,
}

export const colorToChar = {
  Black: 'b',
  White: 'w',
};

export const charToColor = {
  b: 'Black',
  w: 'White',
};

/*
  Converts (2, 1) to 'a2'
  Row & Col are zero indexed
*/
export function rowColToChess(row, col) {
  return `${numberToChar[col]}${row + 1}`;
}

export function chessToRowCol(chessLoc) {
  return [parseInt(chessLoc[1]) - 1, charToNumber[chessLoc[0]]];
}

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
      boardObj[rowColToChess(r, c)] = pieceStr;
    }
  }

  return boardObj;
}
