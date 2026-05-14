const multer = require('multer');
const path = require('path');
const crypto = require('crypto');
const { v4: uuidv4 } = require('uuid');
const { validateFile } = require('../utils/fileValidator');

const ALLOWED_FILE_TYPES = ['image', 'document', 'csv', 'json'];
const DEFAULT_MAX_SIZE = 10 * 1024 * 1024;

const storage = multer.memoryStorage();

function fileFilter(req, file, cb) {
  const allowedExtensions = {
    image: ['.jpg', '.jpeg', '.png', '.gif', '.webp', '.svg'],
    document: ['.pdf', '.doc', '.docx', '.txt', '.rtf'],
    csv: ['.csv'],
    json: ['.json']
  };

  const ext = path.extname(file.originalname).toLowerCase();
  const fileType = req.body.fileType || 'document';

  const allowedExts = allowedExtensions[fileType] || [];

  if (allowedExts.includes(ext)) {
    cb(null, true);
  } else {
    const error = new Error(
      `Invalid file type. Allowed extensions for ${fileType}: ${allowedExts.join(', ')}`
    );
    error.code = 'INVALID_FILE_TYPE';
    cb(error, false);
  }
}

const upload = multer({
  storage,
  fileFilter,
  limits: {
    fileSize: DEFAULT_MAX_SIZE
  }
});

function createUploadMiddleware(options = {}) {
  const {
    fieldName = 'file',
    maxCount = 1,
    maxSize = DEFAULT_MAX_SIZE,
    allowedTypes = ALLOWED_FILE_TYPES
  } = options;

  return multer({
    storage: multer.memoryStorage(),
    fileFilter: (req, file, cb) => {
      const validation = validateFile(
        file.originalname,
        file.buffer,
        allowedTypes
      );

      if (!validation.valid) {
        const error = new Error(validation.errors.join(', '));
        error.code = 'VALIDATION_ERROR';
        return cb(error, false);
      }

      cb(null, true);
    },
    limits: {
      fileSize: maxSize
    }
  }).array(fieldName, maxCount);
}

function handleUploadError(err, req, res, next) {
  if (err instanceof multer.MulterError) {
    if (err.code === 'LIMIT_FILE_SIZE') {
      return res.status(400).json({
        success: false,
        error: `File size exceeds maximum limit of ${DEFAULT_MAX_SIZE / (1024 * 1024)}MB`
      });
    }

    if (err.code === 'LIMIT_FILE_COUNT') {
      return res.status(400).json({
        success: false,
        error: 'Too many files uploaded'
      });
    }

    return res.status(400).json({
      success: false,
      error: err.message
    });
  }

  if (err.code === 'INVALID_FILE_TYPE' || err.code === 'VALIDATION_ERROR') {
    return res.status(400).json({
      success: false,
      error: err.message
    });
  }

  next(err);
}

const uploadMiddleware = upload.array('files', 10);

const uploadSingle = multer({
  storage,
  fileFilter,
  limits: { fileSize: DEFAULT_MAX_SIZE }
}).single('file');

const uploadMultiple = multer({
  storage,
  fileFilter,
  limits: { fileSize: DEFAULT_MAX_SIZE }
}).array('files', 10);

const uploadFields = multer({
  storage,
  fileFilter,
  limits: { fileSize: DEFAULT_MAX_SIZE }
}).fields([
  { name: 'documents', maxCount: 5 },
  { name: 'images', maxCount: 10 }
]);

module.exports = {
  uploadMiddleware,
  uploadSingle,
  uploadMultiple,
  uploadFields,
  createUploadMiddleware,
  handleUploadError,
  ALLOWED_FILE_TYPES,
  DEFAULT_MAX_SIZE
};
