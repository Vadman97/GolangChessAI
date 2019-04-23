class Game {
  humanColor = null;
  currentTurn = null;
  status = null;
  moveLimit = 0;
  timeLimit = 0;
  movesPlayed = 0;

  constructor(humanColor, status, moveLimit, timeLimit) {
    this.humanColor = humanColor;
    this.status = status;
    this.moveLimit = moveLimit;
    this.timeLimit = timeLimit;
  }
}

export const GameStatus = {
  Active: 'Active',
  WhiteWin: 'White Win',
  BlackWin: 'Black Win',
  Stalemate: 'Stalemate',
  FiftyMoveDraw: 'Fifty Move Draw',
  RepeatActionDraw: 'Repeated Action Three Times Draw',
  Aborted: 'Aborted',
};

export default Game;
