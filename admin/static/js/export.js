class DataExporter {
    constructor() {
        this.supportedFormats = ['csv', 'xlsx', 'pdf', 'json', 'xml'];
        this.defaultFilename = 'export';
    }

    exportToCSV(data, filename = 'export', options = {}) {
        if (!data || !Array.isArray(data) || data.length === 0) {
            throw new Error('数据格式错误或为空');
        }

        const columns = options.columns || Object.keys(data[0]);
        const headers = options.headers || columns;
        const delimiter = options.delimiter || ',';

        let csvContent = this.formatCSVRow(headers, delimiter);

        data.forEach(row => {
            const values = columns.map(col => {
                let value = row[col];
                if (value === null || value === undefined) {
                    value = '';
                }
                if (typeof value === 'object') {
                    value = JSON.stringify(value);
                }
                return value;
            });
            csvContent += this.formatCSVRow(values, delimiter);
        });

        const BOM = '\uFEFF';
        const blob = new Blob([BOM + csvContent], { type: 'text/csv;charset=utf-8' });
        this.downloadBlob(blob, `${filename}.csv`);

        return true;
    }

    formatCSVRow(values, delimiter) {
        return values.map(value => {
            const str = String(value);
            if (str.includes(delimiter) || str.includes('"') || str.includes('\n')) {
                return `"${str.replace(/"/g, '""')}"`;
            }
            return str;
        }).join(delimiter) + '\n';
    }

    async exportToExcel(data, filename = 'export', options = {}) {
        if (!data || !Array.isArray(data) || data.length === 0) {
            throw new Error('数据格式错误或为空');
        }

        const columns = options.columns || Object.keys(data[0]);
        const headers = options.headers || columns;
        const sheetName = options.sheetName || 'Sheet1';

        const wsData = [headers];

        data.forEach(row => {
            const values = columns.map(col => {
                let value = row[col];
                if (value === null || value === undefined) {
                    value = '';
                }
                if (typeof value === 'object') {
                    value = JSON.stringify(value);
                }
                return value;
            });
            wsData.push(values);
        });

        const ws = this.createWorksheet(wsData, options);

        const wb = this.createWorkbook();
        const wsSheet = this.addSheetToWorkbook(wb, ws, sheetName);

        if (options.autoSize !== false) {
            this.setColumnWidths(wsSheet, headers);
        }

        const wbout = this.writeWorkbook(wb);

        const blob = new Blob([wbout], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' });
        this.downloadBlob(blob, `${filename}.xlsx`);

        return true;
    }

    createWorksheet(data, options = {}) {
        const ws = [];
        const merges = [];

        data.forEach((row, rowIndex) => {
            const wsRow = [];
            row.forEach((cell, colIndex) => {
                const cellRef = this.getCellRef(rowIndex, colIndex);
                let cellValue = { t: 's', v: String(cell) };

                if (rowIndex === 0 && options.headerStyle) {
                    cellValue.s = options.headerStyle;
                }

                wsRow.push({ c: cellRef, v: cellValue });
            });
            ws.push(wsRow);
        });

        return { data: ws, merges: merges };
    }

    getCellRef(row, col) {
        let cellRef = '';
        let n = col;
        while (n >= 0) {
            cellRef = String.fromCharCode((n % 26) + 65) + cellRef;
            n = Math.floor(n / 26) - 1;
        }
        cellRef += (row + 1);
        return cellRef;
    }

    createWorkbook() {
        return {
            SheetNames: [],
            Sheets: {}
        };
    }

    addSheetToWorkbook(workbook, worksheet, name) {
        workbook.SheetNames.push(name);
        workbook.Sheets[name] = worksheet;
        return worksheet;
    }

    setColumnWidths(worksheet, headers) {
        worksheet['!cols'] = headers.map(header => ({
            wch: Math.max(header.length * 2, 10)
        }));
    }

    writeWorkbook(workbook) {
        const XLSX = window.XLSX;
        return XLSX.write(workbook, { bookType: 'xlsx', type: 'array' });
    }

    async exportToPDF(data, filename = 'export', options = {}) {
        if (!data || !Array.isArray(data) || data.length === 0) {
            throw new Error('数据格式错误或为空');
        }

        const columns = options.columns || Object.keys(data[0]);
        const headers = options.headers || columns;
        const title = options.title || '导出报表';

        const tableData = data.map(row => {
            return columns.map(col => {
                let value = row[col];
                if (value === null || value === undefined) {
                    return '';
                }
                if (typeof value === 'object') {
                    return JSON.stringify(value);
                }
                return String(value);
            });
        });

        const jsPDF = window.jspdf.jsPDF;
        const doc = new jsPDF({
            orientation: options.orientation || 'landscape',
            unit: 'mm',
            format: 'a4'
        });

        doc.setFontSize(18);
        doc.text(title, 14, 22);

        doc.setFontSize(10);
        doc.setTextColor(100);
        doc.text(`生成时间: ${new Date().toLocaleString()}`, 14, 30);

        if (window.jspdf-autotable) {
            doc.autoTable({
                head: [headers],
                body: tableData,
                startY: 35,
                theme: options.theme || 'striped',
                headStyles: {
                    fillColor: [41, 128, 185],
                    textColor: 255,
                    fontSize: 10,
                    fontStyle: 'bold'
                },
                bodyStyles: {
                    fontSize: 9
                },
                alternateRowStyles: {
                    fillColor: [245, 245, 245]
                },
                margin: { top: 10, right: 14, bottom: 20, left: 14 },
                didDrawPage: function(data) {
                    doc.setFontSize(8);
                    doc.setTextColor(150);
                    doc.text(
                        `第 ${data.pageNumber} 页`,
                        data.settings.margin.left,
                        doc.internal.pageSize.height - 10
                    );
                }
            });
        } else {
            const finalY = 35;
            let yPos = finalY;

            doc.setFillColor(41, 128, 185);
            doc.rect(14, yPos, doc.internal.pageSize.width - 28, 8, 'F');
            doc.setTextColor(255);
            doc.setFontSize(10);

            const colWidth = (doc.internal.pageSize.width - 28) / headers.length;
            headers.forEach((header, i) => {
                doc.text(String(header).substring(0, 20), 16 + i * colWidth, yPos + 6);
            });

            yPos += 12;
            doc.setTextColor(0);

            tableData.forEach((row, rowIndex) => {
                if (yPos > doc.internal.pageSize.height - 20) {
                    doc.addPage();
                    yPos = 20;
                }

                if (rowIndex % 2 === 1) {
                    doc.setFillColor(245, 245, 245);
                    doc.rect(14, yPos - 4, doc.internal.pageSize.width - 28, 6, 'F');
                }

                row.forEach((cell, i) => {
                    doc.text(String(cell).substring(0, 25), 16 + i * colWidth, yPos);
                });

                yPos += 6;
            });
        }

        doc.save(`${filename}.pdf`);
        return true;
    }

    exportToJSON(data, filename = 'export', options = {}) {
        const jsonContent = options.pretty !== false
            ? JSON.stringify(data, null, 2)
            : JSON.stringify(data);

        const blob = new Blob([jsonContent], { type: 'application/json' });
        this.downloadBlob(blob, `${filename}.json`);

        return true;
    }

    exportToXML(data, filename = 'export', options = {}) {
        const rootName = options.rootName || 'data';
        const itemName = options.itemName || 'item';

        let xmlContent = `<?xml version="1.0" encoding="UTF-8"?>\n<${rootName}>\n`;

        data.forEach(item => {
            xmlContent += `  <${itemName}>\n`;
            for (const [key, value] of Object.entries(item)) {
                const safeKey = this.sanitizeXMLTag(key);
                const safeValue = this.escapeXML(value);
                xmlContent += `    <${safeKey}>${safeValue}</${safeKey}>\n`;
            }
            xmlContent += `  </${itemName}>\n`;
        });

        xmlContent += `</${rootName}>`;

        const blob = new Blob([xmlContent], { type: 'application/xml' });
        this.downloadBlob(blob, `${filename}.xml`);

        return true;
    }

    sanitizeXMLTag(tag) {
        return tag.replace(/[^a-zA-Z0-9_]/g, '_').replace(/^[0-9]/, '_$&');
    }

    escapeXML(value) {
        if (value === null || value === undefined) {
            return '';
        }
        const str = String(value);
        return str
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&apos;');
    }

    exportTable(tableId, filename = 'export', format = 'csv', options = {}) {
        const table = document.getElementById(tableId);
        if (!table) {
            throw new Error(`表格元素 ${tableId} 不存在`);
        }

        const headers = [];
        const data = [];

        const headerCells = table.querySelectorAll('thead th');
        headerCells.forEach(th => {
            headers.push(th.textContent.trim());
        });

        const rows = table.querySelectorAll('tbody tr');
        rows.forEach(tr => {
            const rowData = [];
            const cells = tr.querySelectorAll('td');
            cells.forEach(td => {
                let value = td.textContent.trim();
                const input = td.querySelector('input, select, textarea');
                if (input) {
                    value = input.value;
                }
                rowData.push(value);
            });
            if (rowData.length > 0) {
                data.push(rowData);
            }
        });

        if (options.excludeColumns) {
            options.excludeColumns.forEach(colIndex => {
                headers.splice(colIndex, 1);
                data.forEach(row => row.splice(colIndex, 1));
            });
        }

        const exportData = data.map(row => {
            const obj = {};
            headers.forEach((header, i) => {
                obj[`col${i}`] = row[i];
            });
            return obj;
        });

        const headerMap = {};
        headers.forEach((h, i) => {
            headerMap[`col${i}`] = h;
        });

        const formattedData = exportData.map(row => {
            const formatted = {};
            for (const [key, value] of Object.entries(row)) {
                formatted[headerMap[key] || key] = value;
            }
            return formatted;
        });

        return this.export(formattedData, filename, format, options);
    }

    export(data, filename = 'export', format = 'csv', options = {}) {
        format = format.toLowerCase();

        switch (format) {
            case 'csv':
                return this.exportToCSV(data, filename, options);
            case 'xlsx':
            case 'excel':
                return this.exportToExcel(data, filename, options);
            case 'pdf':
                return this.exportToPDF(data, filename, options);
            case 'json':
                return this.exportToJSON(data, filename, options);
            case 'xml':
                return this.exportToXML(data, filename, options);
            default:
                throw new Error(`不支持的导出格式: ${format}`);
        }
    }

    async exportBatch(items, filename = 'batch_export', format = 'csv', options = {}) {
        const results = [];
        const batchSize = options.batchSize || 100;
        const totalBatches = Math.ceil(items.length / batchSize);

        for (let i = 0; i < items.length; i += batchSize) {
            const batch = items.slice(i, i + batchSize);
            results.push(...batch);

            if (options.onProgress) {
                const progress = Math.round(((i + batch.length) / items.length) * 100);
                options.onProgress(progress, i + batch.length, items.length);
            }

            if (i + batchSize < items.length) {
                await this.sleep(10);
            }
        }

        if (options.merge !== false) {
            return this.export(results, filename, format, options);
        }

        return results;
    }

    sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    downloadBlob(blob, filename) {
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
    }

    getExportStats(data) {
        return {
            totalRows: Array.isArray(data) ? data.length : 0,
            columns: data && data.length > 0 ? Object.keys(data[0]).length : 0,
            size: JSON.stringify(data).length,
            formats: this.supportedFormats
        };
    }

    validateData(data, rules = {}) {
        const errors = [];

        if (!data) {
            errors.push('数据为空');
            return { valid: false, errors };
        }

        if (!Array.isArray(data)) {
            errors.push('数据必须是数组');
            return { valid: false, errors };
        }

        if (data.length === 0) {
            errors.push('数据为空数组');
        }

        if (rules.minRows && data.length < rules.minRows) {
            errors.push(`数据行数不足，最少需要 ${rules.minRows} 行`);
        }

        if (rules.maxRows && data.length > rules.maxRows) {
            errors.push(`数据行数过多，最多支持 ${rules.maxRows} 行`);
        }

        if (rules.requiredColumns) {
            const missingColumns = rules.requiredColumns.filter(col => {
                return !data[0] || !(col in data[0]);
            });
            if (missingColumns.length > 0) {
                errors.push(`缺少必需列: ${missingColumns.join(', ')}`);
            }
        }

        return {
            valid: errors.length === 0,
            errors
        };
    }
}

const dataExporter = new DataExporter();
