package ai

import (
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"log"
	"os"
	"strconv"
)

type PerformanceLogger struct {
	MakeExcel     bool
	MakeLog       bool
	ExcelFileName string
	LogFileName   string
	ExcelFile     *excelize.File
}

/**
 * Creates a new PerformanceLogger.
 */
func CreatePerformanceLogger(MakeExcel bool, MakeLog bool, ExcelFileName string, LogFileName string) *PerformanceLogger {
	logger := &PerformanceLogger{
		MakeExcel:     MakeExcel,
		MakeLog:       MakeLog,
		ExcelFileName: ExcelFileName,
		LogFileName:   LogFileName,
	}
	logger.setupExcelFile()
	return logger
}

/**
 * Sets up excel file with headings and sheet names.
 */
func (logger *PerformanceLogger) setupExcelFile() {
	if logger.MakeExcel {
		logger.ExcelFile = excelize.NewFile()
		logger.ExcelFile.NewSheet(color.Names[color.White])
		logger.ExcelFile.NewSheet(color.Names[color.Black])
		logger.setupExcelRowHeadings(color.Names[color.White])
		logger.setupExcelRowHeadings(color.Names[color.Black])
	}
}

func (logger *PerformanceLogger) setupExcelRowHeadings(sheet string) {
	logger.ExcelFile.SetCellValue(sheet, "A1", "Turn")
	logger.ExcelFile.SetCellValue(sheet, "B1", "Considered")
	logger.ExcelFile.SetCellValue(sheet, "C1", "Pruned")
	logger.ExcelFile.SetCellValue(sheet, "D1", "Pruned AB")
	logger.ExcelFile.SetCellValue(sheet, "E1", "Pruned Trans")
	logger.ExcelFile.SetCellValue(sheet, "F1", "AB Improved Trans")
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
 * Call this function after the game is complete and no more logging is desired.
 */
func (logger *PerformanceLogger) CompletePerformanceLog(white *Player, black *Player) {
	logger.generateCharts(white)
	logger.generateCharts(black)
	err := logger.ExcelFile.SaveAs(logger.ExcelFileName)
	if err != nil {
		log.Fatal("Cannot save excel performance log.", err)
	}
}

func (logger *PerformanceLogger) generateCharts(p *Player) {
	row := strconv.Itoa(p.TurnCount + 4)
	var chartDataString string
	chartDataString += `{"type":"barPercentStacked","series":[`
	chartDataString += `{"name":"` + color.Names[p.PlayerColor] + `!$D$1","categories":"","values":"` + color.Names[p.PlayerColor] + `!$D$2:$D$` + strconv.Itoa(p.TurnCount+2) + `"},`
	chartDataString += `{"name":"` + color.Names[p.PlayerColor] + `!$A$1","categories":"","values":"` + color.Names[p.PlayerColor] + `!$E$2:$E$` + strconv.Itoa(p.TurnCount+2) + `"}`
	chartDataString += `],"format":{"x_scale":1.0,"y_scale":1.0,"x_offset":15,"y_offset":10,"print_obj":true,"lock_aspect_ratio":false,"locked":false},"legend":{"position":"left","show_legend_key":true},"title":{"name":"Pruning Breakdown"},"plotarea":{"show_bubble_size":true,"show_cat_name":false,"show_leader_lines":false,"show_percent":true,"show_series_name":true,"show_val":true},"show_blanks_as":"zero"}`
	fmt.Println(chartDataString)
	err := logger.ExcelFile.AddChart(color.Names[p.PlayerColor], "B"+row, chartDataString)
	if err != nil {
		fmt.Println(err)
	}
}

/**
 * Performs simple logging to log file.
 */
func (logger *PerformanceLogger) markPerformanceToLog(b *board.Board, m *ScoredMove, p *Player) {
	file, err := os.OpenFile(logger.LogFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Cannot write/open/append/create performance log.", err)
	}
	defer func() { _ = file.Close() }()
	var result string
	result += fmt.Sprintf("Turn %d\n", p.TurnCount)
	result += fmt.Sprintf("%s\n", p.Metrics.Print())
	result += fmt.Sprintf("Board evaluation metrics\n")
	result += p.evaluationMap.PrintMetrics()
	result += fmt.Sprintf("Transposition table metrics\n")
	result += p.alphaBetaTable.PrintMetrics()
	result += fmt.Sprintf("Move cache metrics\n")
	result += b.MoveCache.PrintMetrics()
	result += fmt.Sprintf("Attack Move cache metrics\n")
	result += b.AttackableCache.PrintMetrics()
	result += "A" + string(p.TurnCount+1)
	_, _ = fmt.Fprint(file, result)
}

/**
 * Performs logging to .xlsx file for a graphical representation.
 */
func (logger *PerformanceLogger) markPerformanceToExcel(b *board.Board, m *ScoredMove, p *Player) {
	logger.markMetricsPerformanceToExcel(p)
}

/**
 * Writes metrics data to excel.
 */
func (logger *PerformanceLogger) markMetricsPerformanceToExcel(p *Player) {
	metrics := p.Metrics
	row := strconv.Itoa(p.TurnCount + 2)
	sheet := color.Names[p.PlayerColor]
	logger.ExcelFile.SetCellValue(sheet, "A"+row, p.TurnCount)
	logger.ExcelFile.SetCellValue(sheet, "B"+row, metrics.MovesConsidered)
	logger.ExcelFile.SetCellValue(sheet, "C"+row, metrics.MovesPrunedAB+metrics.MovesPrunedTransposition)
	logger.ExcelFile.SetCellValue(sheet, "D"+row, metrics.MovesPrunedAB)
	logger.ExcelFile.SetCellValue(sheet, "E"+row, metrics.MovesPrunedTransposition)
	logger.ExcelFile.SetCellValue(sheet, "F"+row, metrics.MovesABImprovedTransposition)
}
