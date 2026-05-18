package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/jung-kurt/gofpdf/v2"
	"github.com/xuri/excelize/v2"
)

const (
	FormatCSV   = "csv"
	FormatExcel = "xlsx"
	FormatPDF   = "pdf"
	FormatJSON  = "json"
)

// ExportData 定义通用导出数据结构
type ExportData struct {
	Title    string
	Headers  []string
	Rows     [][]interface{}
	Metadata map[string]string
}

// Exporter 导出器接口
type Exporter interface {
	Export(data ExportData) ([]byte, error)
}

// ExcelExporter Excel导出器
type ExcelExporter struct{}

// PDFExporter PDF导出器
type PDFExporter struct{}

// JSONExporter JSON导出器
type JSONExporter struct{}

// CSVExporter CSV导出器
type CSVExporter struct{}

// NewExcelExporter 创建Excel导出器
func NewExcelExporter() *ExcelExporter {
	return &ExcelExporter{}
}

// NewPDFExporter 创建PDF导出器
func NewPDFExporter() *PDFExporter {
	return &PDFExporter{}
}

// NewJSONExporter 创建JSON导出器
func NewJSONExporter() *JSONExporter {
	return &JSONExporter{}
}

// NewCSVExporter 创建CSV导出器
func NewCSVExporter() *CSVExporter {
	return &CSVExporter{}
}

// GetExporter 根据格式获取对应的导出器
func GetExporter(format string) Exporter {
	switch format {
	case FormatExcel:
		return NewExcelExporter()
	case FormatPDF:
		return NewPDFExporter()
	case FormatJSON:
		return NewJSONExporter()
	case FormatCSV:
		return NewCSVExporter()
	default:
		return NewCSVExporter()
	}
}

// Export 实现导出接口
func (e *ExcelExporter) Export(data ExportData) ([]byte, error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("关闭Excel文件失败: %v", err)
		}
	}()

	sheetName := "Sheet1"
	if data.Title != "" {
		f.SetSheetName("Sheet1", data.Title)
		sheetName = data.Title
	}

	// 写入标题
	if data.Title != "" {
		if err := f.SetCellValue(sheetName, "A1", data.Title); err != nil {
			log.Printf("设置标题失败: %v", err)
		}
		style, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Size: 16},
		})
		if err := f.SetCellStyle(sheetName, "A1", getColumnName(len(data.Headers))+"1", style); err != nil {
			log.Printf("设置标题样式失败: %v", err)
		}
		if err := f.MergeCell(sheetName, "A1", getColumnName(len(data.Headers))+"1"); err != nil {
			log.Printf("合并标题单元格失败: %v", err)
		}
	}

	// 写入表头
	headerRow := 2
	if data.Title != "" {
		headerRow = 3
	}
	for i, header := range data.Headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRow)
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			log.Printf("设置表头失败: %v", err)
		}
	}
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#EFEFEF"}, Pattern: 1},
	})
	if err := f.SetCellStyle(sheetName, fmt.Sprintf("A%d", headerRow), fmt.Sprintf("%s%d", getColumnName(len(data.Headers)), headerRow), headerStyle); err != nil {
		log.Printf("设置表头样式失败: %v", err)
	}

	// 写入数据
	for rowIdx, row := range data.Rows {
		for colIdx, cellValue := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, headerRow+rowIdx+1)
			if err := f.SetCellValue(sheetName, cell, cellValue); err != nil {
				log.Printf("设置单元格值失败: %v", err)
			}
		}
	}

	// 自动调整列宽
	for i := range data.Headers {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		if err := f.SetColWidth(sheetName, colName, colName, 20); err != nil {
			log.Printf("设置列宽失败: %v", err)
		}
	}

	// 写入元数据
	if len(data.Metadata) > 0 {
		metaSheet := "Metadata"
		_, _ = f.NewSheet(metaSheet)
		row := 1
		for k, v := range data.Metadata {
			if err := f.SetCellValue(metaSheet, fmt.Sprintf("A%d", row), k); err != nil {
				log.Printf("设置元数据键失败: %v", err)
			}
			if err := f.SetCellValue(metaSheet, fmt.Sprintf("B%d", row), v); err != nil {
				log.Printf("设置元数据值失败: %v", err)
			}
			row++
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Export 实现PDF导出
func (p *PDFExporter) Export(data ExportData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// 设置字体
	pdf.SetFont("Arial", "B", 16)
	if data.Title != "" {
		pdf.Cell(40, 10, data.Title)
		pdf.Ln(15)
	}

	// 设置表头样式
	pdf.SetFont("Arial", "B", 10)
	colWidth := 190.0 / float64(len(data.Headers))
	for _, header := range data.Headers {
		pdf.CellFormat(colWidth, 7, header, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	// 设置数据行样式
	pdf.SetFont("Arial", "", 9)
	for _, row := range data.Rows {
		for _, cell := range row {
			cellStr := fmt.Sprintf("%v", cell)
			pdf.CellFormat(colWidth, 6, cellStr, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	// 添加元数据
	if len(data.Metadata) > 0 {
		pdf.Ln(10)
		pdf.SetFont("Arial", "I", 8)
		for k, v := range data.Metadata {
			pdf.Cell(0, 5, fmt.Sprintf("%s: %s", k, v))
			pdf.Ln(5)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Export 实现JSON导出
func (j *JSONExporter) Export(data ExportData) ([]byte, error) {
	result := map[string]interface{}{
		"title":       data.Title,
		"headers":     data.Headers,
		"rows":        data.Rows,
		"metadata":    data.Metadata,
		"exported_at": time.Now().Format(time.RFC3339),
	}
	return json.MarshalIndent(result, "", "  ")
}

// Export 实现CSV导出
func (c *CSVExporter) Export(data ExportData) ([]byte, error) {
	var buf bytes.Buffer

	// 写入标题
	if data.Title != "" {
		_, _ = buf.WriteString(fmt.Sprintf("# %s\n", data.Title))
	}

	// 写入表头
	for i, header := range data.Headers {
		if i > 0 {
			_, _ = buf.WriteString(",")
		}
		_, _ = buf.WriteString(fmt.Sprintf("%q", header))
	}
	_, _ = buf.WriteString("\n")

	// 写入数据行
	for _, row := range data.Rows {
		for i, cell := range row {
			if i > 0 {
				_, _ = buf.WriteString(",")
			}
			cellStr := fmt.Sprintf("%v", cell)
			_, _ = buf.WriteString(fmt.Sprintf("%q", cellStr))
		}
		_, _ = buf.WriteString("\n")
	}

	return buf.Bytes(), nil
}

// ConvertLogsToExportData 将验证日志转换为导出数据格式
func ConvertLogsToExportData(logs []models.VerificationLog, title string) ExportData {
	headers := []string{
		"ID",
		"Session ID",
		"Application Name",
		"Captcha Type",
		"Status",
		"IP Address",
		"Risk Score",
		"Duration (ms)",
		"Created At",
		"User Agent",
	}

	rows := make([][]interface{}, len(logs))
	for i, log := range logs {
		appName := ""
		if log.Application.ID > 0 {
			appName = log.Application.Name
		}
		rows[i] = []interface{}{
			log.ID,
			log.SessionID,
			appName,
			log.CaptchaType,
			log.Status,
			log.IPAddress,
			log.RiskScore,
			log.Duration,
			log.CreatedAt.Format("2006-01-02 15:04:05"),
			log.UserAgent,
		}
	}

	metadata := map[string]string{
		"Export Time":  time.Now().Format("2006-01-02 15:04:05"),
		"Record Count": fmt.Sprintf("%d", len(logs)),
	}

	return ExportData{
		Title:    title,
		Headers:  headers,
		Rows:     rows,
		Metadata: metadata,
	}
}

// 辅助函数：获取列名
func getColumnName(index int) string {
	name, _ := excelize.ColumnNumberToName(index)
	return name
}
