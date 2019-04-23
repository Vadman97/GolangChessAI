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

export default Game;
