const path = require('path');
const { fileTypeFromBuffer } = require('file-type');

const ALLOWED_EXTENSIONS = {
  csv: ['csv'],
  json: ['json'],
  excel: ['xlsx', 'xls'],
  image: ['jpg', 'jpeg', 'png', 'gif', 'webp'],
  document: ['pdf', 'doc', 'docx', 'txt'],
  archive: ['zip', 'tar', 'gz']
};

const ALLOWED_MIME_TYPES = {
  csv: ['text/csv', 'application/csv'],
  json: ['application/json'],
  excel: [
    'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    'application/vnd.ms-excel'
  ],
  image: ['image/jpeg', 'image/png', 'image/gif', 'image/webp'],
  document: [
    'application/pdf',
    'application/msword',
    'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    'text/plain'
  ],
  archive: [
    'application/zip',
    'application/x-tar',
    'application/gzip'
  ]
};

const MAX_FILE_SIZES = {
  csv: 10 * 1024 * 1024,
  json: 10 * 1024 * 1024,
  excel: 25 * 1024 * 1024,
  image: 5 * 1024 * 1024,
  document: 10 * 1024 * 1024,
  archive: 50 * 1024 * 1024
};

function validateExtension(filename, allowedTypes) {
  const ext = path.extname(filename).toLowerCase().slice(1);

  for (const type of allowedTypes) {
    if (ALLOWED_EXTENSIONS[type]?.includes(ext)) {
      return { valid: true, extension: ext };
    }
  }

  return {
    valid: false,
    error: `File extension ".${ext}" is not allowed. Allowed extensions: ${getAllowedExtensions(allowedTypes).join(', ')}`
  };
}

function validateMimeType(buffer, allowedTypes) {
  const fileType = fileTypeFromBuffer(buffer);

  if (!fileType) {
    return { valid: true, mimeType: 'unknown' };
  }

  const allowedMimes = getAllowedMimeTypes(allowedTypes);

  if (allowedMimes.includes(fileType.mime)) {
    return { valid: true, mimeType: fileType.mime };
  }

  return {
    valid: false,
    error: `File type "${fileType.mime}" is not allowed. Allowed types: ${allowedMimes.join(', ')}`
  };
}

function validateFileSize(size, allowedTypes) {
  for (const type of allowedTypes) {
    const maxSize = MAX_FILE_SIZES[type];
    if (maxSize && size > maxSize) {
      return {
        valid: false,
        error: `File size exceeds maximum limit of ${formatBytes(maxSize)} for ${type} files`
      };
    }
  }

  return { valid: true };
}

function validateFile(filename, buffer, allowedTypes) {
  const errors = [];

  const extResult = validateExtension(filename, allowedTypes);
  if (!extResult.valid) {
    errors.push(extResult.error);
  }

  const mimeResult = validateMimeType(buffer, allowedTypes);
  if (!mimeResult.valid) {
    errors.push(mimeResult.error);
  }

  const sizeResult = validateFileSize(buffer.length, allowedTypes);
  if (!sizeResult.valid) {
    errors.push(sizeResult.error);
  }

  return {
    valid: errors.length === 0,
    errors,
    details: {
      extension: extResult.extension,
      mimeType: mimeResult.mimeType,
      size: buffer.length,
      formattedSize: formatBytes(buffer.length)
    }
  };
}

function getAllowedExtensions(allowedTypes) {
  const extensions = [];
  for (const type of allowedTypes) {
    extensions.push(...(ALLOWED_EXTENSIONS[type] || []));
  }
  return [...new Set(extensions)];
}

function getAllowedMimeTypes(allowedTypes) {
  const mimeTypes = [];
  for (const type of allowedTypes) {
    mimeTypes.push(...(ALLOWED_MIME_TYPES[type] || []));
  }
  return [...new Set(mimeTypes)];
}

function formatBytes(bytes) {
  if (bytes === 0) return '0 Bytes';

  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function sanitizeFilename(filename) {
  return filename.replace(/[^a-zA-Z0-9._-]/g, '_').substring(0, 255);
}

module.exports = {
  validateExtension,
  validateMimeType,
  validateFileSize,
  validateFile,
  sanitizeFilename,
  formatBytes,
  getAllowedExtensions,
  getAllowedMimeTypes,
  ALLOWED_EXTENSIONS,
  ALLOWED_MIME_TYPES,
  MAX_FILE_SIZES
};
