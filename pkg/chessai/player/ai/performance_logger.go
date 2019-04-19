package ai

import (
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"log"
	"os"
)

type PerformanceLogger struct {
	MakeExcel     bool
	MakeLog       bool
	ExcelFileName string
	LogFileName   string
	ExcelFile     *excelize.File
}

var startingColPruning byte = 'A'
var startingColMoveCache byte = 'K'
var startingColAttackableCache byte = 'T'

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

/**
 * Sets up all excel table row headings.
 */
func (logger *PerformanceLogger) setupExcelRowHeadings(sheet string) {
	logger.setupExcelRowHeadingsForTable(sheet,
		"Pruning Statistics",
		[]string{"Turn", "Considered", "Pruned", "Pruned AB", "Pruned Trans", "AB Improved Trans"},
		startingColPruning)
	cacheHeadings := []string{"Turn", "Entries", "Reads", "Locks used", "Writes", "Hit Ratio", "Read Ratio"}
	logger.setupExcelRowHeadingsForTable(sheet, "Move Cache Statistics", cacheHeadings, startingColMoveCache)
	logger.setupExcelRowHeadingsForTable(sheet, "Attackable Cache Statistics", cacheHeadings,
		startingColAttackableCache)
}

/**
 * Creates row headings for a logging table.
 */
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
func (logger *PerformanceLogger) MarkPerformance(b *board.Board, m *ScoredMove, p *AIPlayer) {
	if logger.MakeLog {
		logger.markPerformanceToLog(b, m, p)
	}
	if logger.MakeExcel {
		logger.markPerformanceToExcel(b, m, p)
	}
}

/**
 * Call this function after the game is complete and no more logging is desired. It will generate all charts and save
 * the excel file.
 */
func (logger *PerformanceLogger) CompletePerformanceLog(aiPlayers []*AIPlayer) {
	for _, ai := range aiPlayers {
		logger.generateChartsForPlayer(ai)
	}

	err := logger.ExcelFile.SaveAs(logger.ExcelFileName)
	if err != nil {
		log.Fatal("Cannot save excel performance log.", err)
	}
}

/**
 * Generates all charts for one player.
 */
func (logger *PerformanceLogger) generateChartsForPlayer(p *AIPlayer) {
	logger.generatePruningBreakdownChart(p)
	logger.generateCacheCharts(p, "Move", startingColMoveCache)
	logger.generateCacheCharts(p, "Attackable", startingColAttackableCache)
}

/**
 * Generates pruning breakdown - AB vs Transposition.
 */
func (logger *PerformanceLogger) generatePruningBreakdownChart(p *AIPlayer) {
	logger.generateChart("barPercentStacked", "Pruning Breakdown", p, startingColPruning,
		p.TurnCount+4, []byte{startingColPruning + byte(3), startingColPruning + byte(4)})
}

/**
 * Generates two charts for a cache.
 * Chart 1 - "Utilization" - Hit and Read ratios
 * Chart 2 - "Size" - Entries, Reads, Writes, and Lock Usage
 */
func (logger *PerformanceLogger) generateCacheCharts(p *AIPlayer, cacheName string, startingCol byte) {
	logger.generateChart("scatter", cacheName+" Cache Utilization", p, startingCol, p.TurnCount+4,
		[]byte{startingColMoveCache + byte(5), startingColMoveCache + byte(6)},
	)
	logger.generateChart("scatter", cacheName+" Cache Size", p, startingCol, p.TurnCount+24,
		[]byte{
			startingColMoveCache + byte(1),
			startingColMoveCache + byte(2),
			startingColMoveCache + byte(3),
			startingColMoveCache + byte(4),
		})
}

/**
 * Generates a chart.
 */
func (logger *PerformanceLogger) generateChart(chartType string, chartTitle string, p *AIPlayer, tableStartCol byte,
	chartRow int, seriesCols []byte) {
	chartCell := fmt.Sprintf("%c%d", tableStartCol, chartRow)
	sheet := color.Names[p.PlayerColor]
	lastTurnRow := p.TurnCount + 2
	var chartData string
	chartData += fmt.Sprintf(`{"type":"%s","series":[`, chartType)
	for index, element := range seriesCols {
		chartData += logger.generateSeriesString(sheet, lastTurnRow, element, tableStartCol)
		if index+1 != len(seriesCols) {
			chartData += `,`
		}
	}
	chartData += `],"format":{"x_scale":1.0,"y_scale":1.0,"x_offset":15,"y_offset":10,"print_obj":true,`
	chartData += `"lock_aspect_ratio":false,"locked":false},"legend":{"position":"top","show_legend_key":true},`
	chartData += fmt.Sprintf(`"title":{"name":"%s"},`, chartTitle)
	chartData += `"plotarea":{"show_bubble_size":true,"show_cat_name":false,`
	chartData += `"show_leader_lines":false,"show_percent":false,"show_series_name":false,"show_val":false},`
	chartData += `"show_blanks_as":"zero"}`
	err := logger.ExcelFile.AddChart(color.Names[p.PlayerColor], chartCell, chartData)
	if err != nil {
		log.Fatal(err)
	}
}

/**
 * Generates a series string for a chart.
 */
func (logger *PerformanceLogger) generateSeriesString(sheet string, lastTurnRow int, seriesCol byte,
	categoriesCol byte) string {
	series := `{"name":"%s!$%c$2", "categories":"%s!$%c$3:$%c$%d","values":"%s!$%c$3:$%c$%d"}`
	return fmt.Sprintf(series, sheet, seriesCol, sheet, categoriesCol, categoriesCol, lastTurnRow, sheet, seriesCol,
		seriesCol, lastTurnRow)
}

/**
 * Performs simple logging to log file.
 */
func (logger *PerformanceLogger) markPerformanceToLog(b *board.Board, m *ScoredMove, p *AIPlayer) {
	file, err := os.OpenFile(logger.LogFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Cannot write/open/append/create performance log.", err)
	}
	defer func() { _ = file.Close() }()
	var result string
	result += fmt.Sprintf("Turn %d\n", p.TurnCount)
	result += fmt.Sprintf("%s\n", p.Metrics)
	result += fmt.Sprintf("Board evaluation metrics\n")
	result += p.evaluationMap.String()
	result += fmt.Sprintf("Transposition table metrics\n")
	result += p.transpositionTable.PrintMetrics()
	result += fmt.Sprintf("Move cache metrics\n")
	result += b.MoveCache.String()
	result += fmt.Sprintf("Attack Move cache metrics\n")
	result += b.AttackableCache.String()
	_, _ = fmt.Fprint(file, result)
}

/**
 * Performs logging to .xlsx file to build various tables.
 */
func (logger *PerformanceLogger) markPerformanceToExcel(b *board.Board, m *ScoredMove, p *AIPlayer) {
	logger.markMetricsToExcelTable(p,
		[]interface{}{
			p.TurnCount,
			p.Metrics.MovesConsidered,
			p.Metrics.MovesPrunedAB + p.Metrics.MovesPrunedTransposition,
			p.Metrics.MovesPrunedAB,
			p.Metrics.MovesPrunedTransposition,
			p.Metrics.MovesABImprovedTransposition,
		}, startingColPruning)
	logger.markMetricsToExcelTable(p,
		[]interface{}{
			p.TurnCount,
			b.MoveCache.GetTotalWrites(),
			b.MoveCache.GetTotalReads(),
			b.MoveCache.GetTotalWrites(),
			b.MoveCache.GetTotalLockUsage(),
			b.MoveCache.GetHitRatio(),
			b.MoveCache.GetReadRatio(),
		}, startingColMoveCache)
	logger.markMetricsToExcelTable(p,
		[]interface{}{
			p.TurnCount,
			b.AttackableCache.GetTotalWrites(),
			b.AttackableCache.GetTotalReads(),
			b.AttackableCache.GetTotalWrites(),
			b.AttackableCache.GetTotalLockUsage(),
			b.AttackableCache.GetHitRatio(),
			b.AttackableCache.GetReadRatio(),
		}, startingColAttackableCache)
}

/**
 * Prints one turn of metrics.
 */
func (logger *PerformanceLogger) markMetricsToExcelTable(p *AIPlayer, values []interface{}, startColumn byte) {
	row := p.TurnCount + 3
	sheet := color.Names[p.PlayerColor]
	excel := logger.ExcelFile
	for index, value := range values {
		cell := fmt.Sprintf("%c%d", startColumn+byte(index), row)
		excel.SetCellValue(sheet, cell, value)
	}
}
