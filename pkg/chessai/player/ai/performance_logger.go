package ai

import (
	//"fmt"
	//"github.com/360EntSecGroup-Skylar/excelize"
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"log"
	"os"
)

type PerformanceLogger struct {
	MakeExcel     bool
	MakeLog       bool
	ExcelFileName string
	LogFileName   string
}

/**
 * Creates a new PerformanceLogger.
 */
func CreatePerformanceLogger(MakeExcel bool, MakeLog bool, ExcelFileName string, LogFileName string) *PerformanceLogger {
	return &PerformanceLogger{
		MakeExcel:     MakeExcel,
		MakeLog:       MakeLog,
		ExcelFileName: ExcelFileName,
		LogFileName:   LogFileName,
	}
}

/**
 * Performs logging as desired.
 */
func (logger *PerformanceLogger) MarkPerformance(b *board.Board, m *ScoredMove, p *Player) {
	if logger.MakeLog {
		logger.markPerformanceToLog(b, m, p)
	}
	if logger.MakeExcel {
		logger.markPerformanceToExcel(b, m, p)
	}
}

/**
 * Performs simple logging to log file.
 */
func (logger *PerformanceLogger) markPerformanceToLog(b *board.Board, m *ScoredMove, p *Player) {
	file, err := os.OpenFile(logger.LogFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Cannot open file", err)
	}
	defer func() { _ = file.Close() }()
	var result string
	result += fmt.Sprintf("%s\n", p.Metrics.Print())
	result += fmt.Sprintf("Board evaluation metrics\n")
	result += p.evaluationMap.PrintMetrics()
	result += fmt.Sprintf("Transposition table metrics\n")
	result += p.alphaBetaTable.PrintMetrics()
	result += fmt.Sprintf("Move cache metrics\n")
	result += b.MoveCache.PrintMetrics()
	result += fmt.Sprintf("Attack Move cache metrics\n")
	result += b.AttackableCache.PrintMetrics()
	_, _ = fmt.Fprint(file, result)
}

/**
 * Performs logging to .xlsx file for a graphical representation.
 */
func (logger *PerformanceLogger) markPerformanceToExcel(b *board.Board, m *ScoredMove, p *Player) {
	//TODO
}
