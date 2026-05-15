const express = require('express');

const router = express.Router();
const { auth } = require('../middleware/auth');
const exportService = require('../services/exportService');

router.use(auth);

router.get('/:format', async (req, res, next) => {
  try {
    const { format } = req.params;
    const { table, fields, filename } = req.query;

    if (!table) {
      return res.status(400).json({
        success: false,
        error: 'Table name is required'
      });
    }

    if (!exportService.EXPORT_FORMATS.includes(format.toLowerCase())) {
      return res.status(400).json({
        success: false,
        error: `Invalid format. Supported formats: ${exportService.EXPORT_FORMATS.join(', ')}`
      });
    }

    const data = await fetchDataForExport(table, fields);

    const exportOptions = {
      filename: filename || `${table}_export.${format}`,
      fields: fields ? fields.split(',') : undefined
    };

    const exportResult = await exportService.exportData(data, format, exportOptions);

    res.setHeader('Content-Type', exportResult.mimeType);
    res.setHeader('Content-Disposition', `attachment; filename="${exportResult.filename}"`);
    res.setHeader('Content-Length', exportResult.size);

    if (Buffer.isBuffer(exportResult.content)) {
      res.send(exportResult.content);
    } else {
      res.send(exportResult.content);
    }
  } catch (error) {
    next(error);
  }
});

router.post('/batch', async (req, res, next) => {
  try {
    const { tables, format = 'json' } = req.body;

    if (!tables || !Array.isArray(tables) || tables.length === 0) {
      return res.status(400).json({
        success: false,
        error: 'Tables array is required'
      });
    }

    const exportResults = {};

    for (const tableName of tables) {
      try {
        const data = await fetchDataForExport(tableName);
        exportResults[tableName] = {
          success: true,
          count: data.length
        };
      } catch (error) {
        exportResults[tableName] = {
          success: false,
          error: error.message
        };
      }
    }

    const exportResult = await exportService.exportData(
      Object.entries(exportResults).map(([table, result]) => ({
        table,
        ...result
      })),
      format
    );

    res.setHeader('Content-Type', exportResult.mimeType);
    res.setHeader('Content-Disposition', `attachment; filename="batch_export.${format}"`);
    res.send(exportResult.content);
  } catch (error) {
    next(error);
  }
});

async function fetchDataForExport(tableName, fields) {
  const pool = require('../../../config/database/db');

  const allowedTables = ['users', 'products', 'orders', 'notifications'];
  if (!allowedTables.includes(tableName)) {
    throw new Error(`Table "${tableName}" is not allowed for export`);
  }

  let query = 'SELECT';
  if (fields) {
    const fieldList = fields
      .split(',')
      .map(f => f.trim())
      .join(', ');
    query += ` ${fieldList}`;
  } else {
    query += ' *';
  }
  query += ` FROM ${tableName} LIMIT ${exportService.MAX_EXPORT_RECORDS}`;

  const result = await pool.query(query);
  return result.rows;
}

module.exports = router;
