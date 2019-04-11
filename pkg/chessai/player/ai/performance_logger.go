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
	logger.setupExcelRowHeadingsForTable(sheet,
		"Pruning Statistics",
		[]string{"Turn", "Considered", "Pruned", "Pruned AB", "Pruned Trans", "AB Improved Trans"},
		'A')
	logger.setupExcelRowHeadingsForTable(sheet,
		"Move Cache Statistics",
		[]string{"Turn", "Entries", "Reads", "Writes", "Hit Ratio", "Read Ratio", "Locks used"},
		'K')
}

func (logger *PerformanceLogger) setupExcelRowHeadingsForTable(sheet string, tableHeading string,
	columnHeadings []string, startColumn byte) {
	excel := logger.ExcelFile
	for index, heading := range columnHeadings {
		cell := fmt.Sprintf("%c%d", startColumn+byte(index), 2)
		excel.SetCellValue(sheet, cell, heading)
	}
	firstCell := fmt.Sprintf("%c%d", startColumn, 1)
	lastCell := fmt.Sprintf("%c%d", startColumn+byte(len(columnHeadings))-1, 1)
	excel.MergeCell(sheet, firstCell, lastCell)
	excel.SetCellValue(sheet, firstCell, tableHeading)
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
	logger.generatePruningBreakdownChart(white)
	logger.generatePruningBreakdownChart(black)
	err := logger.ExcelFile.SaveAs(logger.ExcelFileName)
	if err != nil {
		log.Fatal("Cannot save excel performance log.", err)
	}
}

func (logger *PerformanceLogger) generatePruningBreakdownChart(p *Player) {
	row := strconv.Itoa(p.TurnCount + 5)
	var chartDataString string
	c := color.Names[p.PlayerColor]
	lastTurnRow := p.TurnCount + 2
	series := `{"name":"%s!$%c$1", "categories":"%s!$%c$2:$%c$%d","values":"%s!$%c$2:$%c$%d"}`
	chartDataString += `{"type":"barPercentStacked","series":[`
	chartDataString += fmt.Sprintf(series, c, 'D', c, 'A', 'A', lastTurnRow, c, 'D', 'D', lastTurnRow)
	chartDataString += ","
	chartDataString += fmt.Sprintf(series, c, 'E', c, 'A', 'A', lastTurnRow, c, 'E', 'E', lastTurnRow)
	chartDataString += `],"format":{"x_scale":1.0,"y_scale":1.0,"x_offset":15,"y_offset":10,"print_obj":true,`
	chartDataString += `"lock_aspect_ratio":false,"locked":false},"legend":{"position":"left","show_legend_key":true},`
	chartDataString += `"title":{"name":"Pruning Breakdown"},"plotarea":{"show_bubble_size":true,"show_cat_name":false,`
	chartDataString += `"show_leader_lines":false,"show_percent":true,"show_series_name":false,"show_val":false},`
	chartDataString += `"show_blanks_as":"zero"}`
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
	_, _ = fmt.Fprint(file, result)
}

/**
 * Performs logging to .xlsx file for a graphical representation.
 */
func (logger *PerformanceLogger) markPerformanceToExcel(b *board.Board, m *ScoredMove, p *Player) {
	metrics := p.Metrics
	logger.markMetricsToExcelTable(p,
		[]interface{}{
			p.TurnCount,
			metrics.MovesConsidered,
			metrics.MovesPrunedAB + metrics.MovesPrunedTransposition,
			metrics.MovesPrunedAB,
			metrics.MovesPrunedTransposition,
			metrics.MovesABImprovedTransposition,
		}, 'A')
}

func (logger *PerformanceLogger) markMetricsToExcelTable(p *Player, values []interface{}, startColumn byte) {
	row := p.TurnCount + 3
	sheet := color.Names[p.PlayerColor]
	excel := logger.ExcelFile
	for index, value := range values {
		cell := fmt.Sprintf("%c%d", startColumn+byte(index), row)
		excel.SetCellValue(sheet, cell, value)
	}
}
