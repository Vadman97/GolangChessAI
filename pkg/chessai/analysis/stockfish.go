package analysis

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// StockfishEngine wraps a Stockfish process communicating via UCI.
type StockfishEngine struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
}

// NewStockfishEngine starts Stockfish and completes the UCI handshake.
func NewStockfishEngine(binaryPath string) (*StockfishEngine, error) {
	if _, err := os.Stat(binaryPath); err != nil {
		return nil, fmt.Errorf("stockfish binary not found at %s: %w", binaryPath, err)
	}
	cmd := exec.Command(binaryPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	sf := &StockfishEngine{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdoutPipe),
	}
	sf.send("uci")
	sf.waitFor("uciok")
	sf.send("isready")
	sf.waitFor("readyok")
	return sf, nil
}

func (sf *StockfishEngine) send(cmd string) {
	fmt.Fprintln(sf.stdin, cmd)
}

func (sf *StockfishEngine) waitFor(token string) {
	for sf.stdout.Scan() {
		if strings.HasPrefix(sf.stdout.Text(), token) {
			return
		}
	}
}

// EvalResult holds Stockfish's analysis of a position.
type EvalResult struct {
	// BestMove in UCI notation (e.g. "e2e4").
	BestMove string
	// CentipawnsSTM is the score in centipawns from the side-to-move's perspective.
	// This is the raw UCI score: +100 means the side to move is ahead by one pawn.
	// Mate scores are ±100000.
	CentipawnsSTM int
	IsMate        bool
	MateIn        int
}

// Analyze asks Stockfish to evaluate the given FEN at the specified depth
// and returns the best move + score from White's perspective.
func (sf *StockfishEngine) Analyze(fen string, depth int) EvalResult {
	sf.send("position fen " + fen)
	sf.send(fmt.Sprintf("go depth %d", depth))

	result := EvalResult{}
	for sf.stdout.Scan() {
		line := sf.stdout.Text()
		if strings.HasPrefix(line, "bestmove") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				result.BestMove = parts[1]
			}
			break
		}
		// Parse the last "info ... score cp/mate ..." line
		if strings.HasPrefix(line, "info") && strings.Contains(line, " score ") {
			result = parseInfoLine(line)
		}
	}
	return result
}

// AnalyzeMove asks Stockfish to evaluate only one candidate move from a FEN.
// The returned score uses the same side-to-move perspective as Analyze.
func (sf *StockfishEngine) AnalyzeMove(fen, move string, depth int) EvalResult {
	sf.send("position fen " + fen)
	sf.send(fmt.Sprintf("go depth %d searchmoves %s", depth, move))

	result := EvalResult{BestMove: move}
	for sf.stdout.Scan() {
		line := sf.stdout.Text()
		if strings.HasPrefix(line, "bestmove") {
			break
		}
		if strings.HasPrefix(line, "info") && strings.Contains(line, " score ") {
			result = parseInfoLine(line)
			result.BestMove = move
		}
	}
	return result
}

func parseInfoLine(line string) EvalResult {
	r := EvalResult{}
	parts := strings.Fields(line)
	for i, tok := range parts {
		switch tok {
		case "cp":
			if i+1 < len(parts) {
				if v, err := strconv.Atoi(parts[i+1]); err == nil {
					r.CentipawnsSTM = v
					r.IsMate = false
				}
			}
		case "mate":
			if i+1 < len(parts) {
				if v, err := strconv.Atoi(parts[i+1]); err == nil {
					r.MateIn = v
					r.IsMate = true
					if v > 0 {
						r.CentipawnsSTM = 100000
					} else {
						r.CentipawnsSTM = -100000
					}
				}
			}
		}
	}
	return r
}

// Close shuts down the Stockfish process.
func (sf *StockfishEngine) Close() {
	sf.send("quit")
	_ = sf.cmd.Wait()
}
