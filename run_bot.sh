#!/bin/bash
cd /home/vadim/code/GolangChessAI
while true; do
  echo "$(date): starting bot..." >> /tmp/chess.lichess.log
  LICHESS_TOKEN=${LICHESS_TOKEN} ./main lichess >> /tmp/chess.lichess.log 2>&1
  EXIT_CODE=$?
  echo "$(date): bot exited with code $EXIT_CODE, restarting in 30s..." >> /tmp/chess.lichess.log
  sleep 30
done
