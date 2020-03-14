# Go Chess - Exploring Parallel Search Techniques with a Novel Golang Chess Engine
[![Build Status](https://travis-ci.com/Vadman97/GolangChessAI.svg?branch=master)](https://travis-ci.com/Vadman97/GolangChessAI)
[![Coverage](https://codecov.io/gh/Vadman97/GolangChessAI/branch/master/graph/badge.svg?token=IGeQbLUCCM)](https://codecov.io/gh/Vadman97/GolangChessAI)

**Developed by:** Devan Adhia, Vadim Korolik, Alexander Lee, and Suveena Thanawala

Go implementation of Chess AI engine with serial and parallel algorithms.

Core code structure based on [c++ version](https://github.com/Vadman97/ChessAI2).

## Running the code
To run a 10-game 5-second think time competition between ABDADA parallel algorithm and MTDf serial algorithm, run `./main competition`

Running the full frontend is more difficult and will require building from source.

To run the frontend, clone the repo and run `npm install; npm start; go build -o main FOLDER_WHERE_YOU_CLONED_TO/cmd/main.go; ./main`