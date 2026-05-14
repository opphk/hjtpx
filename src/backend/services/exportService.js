const fs = require('fs');
const path = require('path');
const { Parser } = require('json2csv');

const EXPORT_FORMATS = ['csv', 'excel', 'json'];
const MAX_EXPORT_RECORDS = 10000;

async function exportToCSV(data, options = {}) {
  const { fields, filename = 'export.csv' } = options;

  if (!Array.isArray(data)) {
    throw new Error('Data must be an array for CSV export');
  }

  if (data.length > MAX_EXPORT_RECORDS) {
    throw new Error(`Export exceeds maximum record limit of ${MAX_EXPORT_RECORDS}`);
  }

  const fieldsToUse = fields || extractFieldsFromData(data);

  const parser = new Parser({ fields: fieldsToUse });
  const csv = parser.parse(data);

  return {
    filename,
    content: csv,
    mimeType: 'text/csv',
    size: Buffer.byteLength(csv, 'utf8')
  };
}

async function exportToJSON(data, options = {}) {
  const { filename = 'export.json', pretty = true } = options;

  if (!Array.isArray(data)) {
    throw new Error('Data must be an array for JSON export');
  }

  if (data.length > MAX_EXPORT_RECORDS) {
    throw new Error(`Export exceeds maximum record limit of ${MAX_EXPORT_RECORDS}`);
  }

  const json = pretty ? JSON.stringify(data, null, 2) : JSON.stringify(data);

  return {
    filename,
    content: json,
    mimeType: 'application/json',
    size: Buffer.byteLength(json, 'utf8')
  };
}

async function exportToExcel(data, options = {}) {
  const { filename = 'export.xlsx' } = options;

  if (!Array.isArray(data)) {
    throw new Error('Data must be an array for Excel export');
  }

  if (data.length > MAX_EXPORT_RECORDS) {
    throw new Error(`Export exceeds maximum record limit of ${MAX_EXPORT_RECORDS}`);
  }

  const ExcelJS = require('exceljs');
  const workbook = new ExcelJS.Workbook();
  const worksheet = workbook.addWorksheet('Export');

  if (data.length > 0) {
    const headers = Object.keys(data[0]);
    worksheet.addRow(headers);

    data.forEach(item => {
      const row = headers.map(header => item[header]);
      worksheet.addRow(row);
    });
  }

  const buffer = await workbook.xlsx.writeBuffer();

  return {
    filename,
    content: buffer,
    mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    size: buffer.length
  };
}

function extractFieldsFromData(data) {
  if (!data || data.length === 0) {
    return [];
  }

  const fields = new Set();
  data.forEach(item => {
    if (typeof item === 'object' && item !== null) {
      Object.keys(item).forEach(key => fields.add(key));
    }
  });

  return Array.from(fields).map(field => ({
    label: field,
    value: field
  }));
}

async function exportData(data, format, options = {}) {
  switch (format.toLowerCase()) {
    case 'csv':
      return exportToCSV(data, options);
    case 'excel':
    case 'xlsx':
      return exportToExcel(data, options);
    case 'json':
      return exportToJSON(data, options);
    default:
      throw new Error(`Unsupported export format: ${format}. Supported formats: ${EXPORT_FORMATS.join(', ')}`);
  }
}

async function exportToFile(exportResult, outputDir = 'exports') {
  const fullPath = path.join(outputDir, exportResult.filename);

  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
  }

  if (Buffer.isBuffer(exportResult.content)) {
    fs.writeFileSync(fullPath, exportResult.content);
  } else {
    fs.writeFileSync(fullPath, exportResult.content, 'utf8');
  }

  return fullPath;
}

module.exports = {
  exportToCSV,
  exportToJSON,
  exportToExcel,
  exportData,
  exportToFile,
  extractFieldsFromData,
  EXPORT_FORMATS,
  MAX_EXPORT_RECORDS
};
