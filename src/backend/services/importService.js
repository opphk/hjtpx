const fs = require('fs');
const path = require('path');
const { parse } = require('csv-parse/sync');
const pool = require('../../../config/database/db');

const ALLOWED_FILE_TYPES = ['csv', 'json'];
const MAX_FILE_SIZE = 10 * 1024 * 1024;
const MAX_RECORDS = 5000;

async function parseCSV(filePath, options = {}) {
  const { delimiter = ',', encoding = 'utf8' } = options;

  const fileContent = fs.readFileSync(filePath, { encoding });
  const records = parse(fileContent, {
    delimiter,
    columns: true,
    skip_empty_lines: true,
    trim: true
  });

  return records;
}

async function parseJSON(filePath) {
  const fileContent = fs.readFileSync(filePath, { encoding: 'utf8' });
  const data = JSON.parse(fileContent);

  if (Array.isArray(data)) {
    return data;
  }

  if (data.data && Array.isArray(data.data)) {
    return data.data;
  }

  return [data];
}

async function validateRecords(records, schema) {
  const errors = [];
  const validRecords = [];

  for (let i = 0; i < records.length; i++) {
    const record = records[i];
    const recordErrors = [];

    for (const [field, rules] of Object.entries(schema)) {
      const value = record[field];

      if (rules.required && (value === undefined || value === null || value === '')) {
        recordErrors.push(`${field} is required`);
      }

      if (value !== undefined && value !== null && value !== '') {
        if (rules.type) {
          const actualType = typeof value;
          if (rules.type === 'number' && isNaN(Number(value))) {
            recordErrors.push(`${field} must be a number`);
          } else if (rules.type === 'string' && actualType !== 'string') {
            recordErrors.push(`${field} must be a string`);
          } else if (rules.type === 'boolean' && value !== 'true' && value !== 'false') {
            recordErrors.push(`${field} must be a boolean`);
          }
        }

        if (rules.minLength && value.length < rules.minLength) {
          recordErrors.push(`${field} must be at least ${rules.minLength} characters`);
        }

        if (rules.maxLength && value.length > rules.maxLength) {
          recordErrors.push(`${field} must be at most ${rules.maxLength} characters`);
        }

        if (rules.pattern && !new RegExp(rules.pattern).test(value)) {
          recordErrors.push(`${field} format is invalid`);
        }
      }
    }

    if (recordErrors.length > 0) {
      errors.push({
        row: i + 1,
        errors: recordErrors
      });
    } else {
      validRecords.push(record);
    }
  }

  return { validRecords, errors };
}

async function importRecords(records, tableName, options = {}) {
  const { onProgress, batchSize = 100 } = options;

  if (records.length > MAX_RECORDS) {
    throw new Error(`Import exceeds maximum record limit of ${MAX_RECORDS}`);
  }

  if (records.length === 0) {
    return { imported: 0, failed: 0, errors: [] };
  }

  const columns = Object.keys(records[0]);
  const importErrors = [];
  let imported = 0;

  for (let i = 0; i < records.length; i += batchSize) {
    const batch = records.slice(i, i + batchSize);

    try {
      const values = batch.map(record =>
        columns.map(col => record[col])
      );

      const placeholders = batch.map((_, rowIndex) =>
        `(${columns.map((_, colIndex) =>
          `$${rowIndex * columns.length + colIndex + 1}`
        ).join(', ')})`
      ).join(', ');

      const query = `
        INSERT INTO ${tableName} (${columns.join(', ')})
        VALUES ${placeholders}
        ON CONFLICT DO NOTHING
      `;

      const flatValues = values.flat();
      await pool.query(query, flatValues);

      imported += batch.length;

      if (onProgress) {
        onProgress({
          imported,
          total: records.length,
          percentage: Math.round((imported / records.length) * 100)
        });
      }
    } catch (error) {
      importErrors.push({
        batch: Math.floor(i / batchSize) + 1,
        error: error.message
      });
    }
  }

  return {
    imported,
    failed: records.length - imported,
    errors: importErrors
  };
}

async function importFromFile(filePath, tableName, schema, options = {}) {
  const fileExt = path.extname(filePath).toLowerCase().slice(1);

  if (!ALLOWED_FILE_TYPES.includes(fileExt)) {
    throw new Error(`File type not supported. Allowed types: ${ALLOWED_FILE_TYPES.join(', ')}`);
  }

  const stats = fs.statSync(filePath);
  if (stats.size > MAX_FILE_SIZE) {
    throw new Error(`File size exceeds maximum limit of ${MAX_FILE_SIZE / (1024 * 1024)}MB`);
  }

  let records;
  if (fileExt === 'csv') {
    records = await parseCSV(filePath, options);
  } else if (fileExt === 'json') {
    records = await parseJSON(filePath);
  }

  const { validRecords, errors: validationErrors } = await validateRecords(records, schema);

  if (validRecords.length === 0) {
    return {
      imported: 0,
      failed: records.length,
      validationErrors,
      importErrors: []
    };
  }

  const importResult = await importRecords(validRecords, tableName, options);

  return {
    ...importResult,
    validationErrors,
    totalProcessed: records.length
  };
}

module.exports = {
  parseCSV,
  parseJSON,
  validateRecords,
  importRecords,
  importFromFile,
  ALLOWED_FILE_TYPES,
  MAX_FILE_SIZE,
  MAX_RECORDS
};
