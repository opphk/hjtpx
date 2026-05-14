const express = require('express');
const router = express.Router();
const multer = require('multer');
const path = require('path');
const { v4: uuidv4 } = require('uuid');
const importService = require('../services/importService');
const { authMiddleware } = require('../middleware/auth');
const { requireRole } = require('../middleware/roleCheck');
const fileValidator = require('../utils/fileValidator');

router.use(authMiddleware);
router.use(requireRole(['admin', 'manager']));

const storage = multer.diskStorage({
  destination: (req, file, cb) => {
    const uploadDir = path.join(__dirname, '../../../uploads/imports');
    require('fs').mkdirSync(uploadDir, { recursive: true });
    cb(null, uploadDir);
  },
  filename: (req, file, cb) => {
    const uniqueName = `${uuidv4()}${path.extname(file.originalname)}`;
    cb(null, uniqueName);
  }
});

const upload = multer({
  storage,
  limits: {
    fileSize: importService.MAX_FILE_SIZE
  },
  fileFilter: (req, file, cb) => {
    const ext = path.extname(file.originalname).toLowerCase().slice(1);
    if (importService.ALLOWED_FILE_TYPES.includes(ext)) {
      cb(null, true);
    } else {
      cb(new Error(`File type not supported. Allowed types: ${importService.ALLOWED_FILE_TYPES.join(', ')}`));
    }
  }
});

router.post('/csv', upload.single('file'), async (req, res, next) => {
  try {
    if (!req.file) {
      return res.status(400).json({
        success: false,
        error: 'No file uploaded'
      });
    }

    const { table, schema: schemaString } = req.body;

    if (!table) {
      return res.status(400).json({
        success: false,
        error: 'Table name is required'
      });
    }

    const schema = schemaString ? JSON.parse(schemaString) : {};

    const result = await importService.importFromFile(req.file.path, table, schema, {
      delimiter: req.body.delimiter || ','
    });

    res.json({
      success: true,
      data: result
    });
  } catch (error) {
    next(error);
  }
});

router.post('/json', upload.single('file'), async (req, res, next) => {
  try {
    if (!req.file) {
      return res.status(400).json({
        success: false,
        error: 'No file uploaded'
      });
    }

    const { table, schema: schemaString } = req.body;

    if (!table) {
      return res.status(400).json({
        success: false,
        error: 'Table name is required'
      });
    }

    const schema = schemaString ? JSON.parse(schemaString) : {};

    const result = await importService.importFromFile(req.file.path, table, schema);

    res.json({
      success: true,
      data: result
    });
  } catch (error) {
    next(error);
  }
});

router.post('/preview', upload.single('file'), async (req, res, next) => {
  try {
    if (!req.file) {
      return res.status(400).json({
        success: false,
        error: 'No file uploaded'
      });
    }

    const fileExt = path.extname(req.file.originalname).toLowerCase().slice(1);
    let records;

    if (fileExt === 'csv') {
      records = await importService.parseCSV(req.file.path);
    } else if (fileExt === 'json') {
      records = await importService.parseJSON(req.file.path);
    }

    const preview = records.slice(0, 10);
    const columns = records.length > 0 ? Object.keys(records[0]) : [];

    res.json({
      success: true,
      data: {
        preview,
        columns,
        totalRows: records.length,
        filename: req.file.originalname
      }
    });
  } catch (error) {
    next(error);
  }
});

module.exports = router;
