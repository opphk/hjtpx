const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { v4: uuidv4 } = require('uuid');
const pool = require('../../../config/database/db');

const STORAGE_PROVIDERS = {
  local: 'local',
  s3: 's3',
  gcs: 'gcs',
  azure: 'azure'
};

async function uploadFile(file, options = {}) {
  const {
    userId,
    folder = 'general',
    allowedTypes = ['image', 'document'],
    maxSize = 10 * 1024 * 1024
  } = options;

  if (!file) {
    throw new Error('No file provided');
  }

  if (file.size > maxSize) {
    throw new Error(`File size exceeds maximum limit of ${maxSize / (1024 * 1024)}MB`);
  }

  const fileId = uuidv4();
  const fileExt = path.extname(file.originalname).toLowerCase();
  const storedFilename = `${fileId}${fileExt}`;

  const uploadDir = path.join(__dirname, '../../../uploads', folder);
  if (!fs.existsSync(uploadDir)) {
    fs.mkdirSync(uploadDir, { recursive: true });
  }

  const filePath = path.join(uploadDir, storedFilename);
  fs.writeFileSync(filePath, file.buffer);

  const fileRecord = {
    id: fileId,
    originalName: file.originalname,
    storedName: storedFilename,
    mimeType: file.mimetype,
    size: file.size,
    folder,
    path: filePath,
    userId,
    checksum: crypto.createHash('md5').update(file.buffer).digest('hex')
  };

  await saveFileRecord(fileRecord);

  return {
    id: fileId,
    originalName: file.originalname,
    storedName: storedFilename,
    url: `/uploads/${folder}/${storedFilename}`,
    mimeType: file.mimetype,
    size: file.size,
    checksum: fileRecord.checksum
  };
}

async function saveFileRecord(fileRecord) {
  const query = `
    INSERT INTO files (
      id, original_name, stored_name, mime_type, size,
      folder, path, user_id, checksum, created_at
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP)
    RETURNING *
  `;

  const values = [
    fileRecord.id,
    fileRecord.originalName,
    fileRecord.storedName,
    fileRecord.mimeType,
    fileRecord.size,
    fileRecord.folder,
    fileRecord.path,
    fileRecord.userId,
    fileRecord.checksum
  ];

  const result = await pool.query(query, values);
  return result.rows[0];
}

async function getFile(fileId) {
  const query = 'SELECT * FROM files WHERE id = $1';
  const result = await pool.query(query, [fileId]);

  if (result.rows.length === 0) {
    throw new Error('File not found');
  }

  return result.rows[0];
}

async function getUserFiles(userId, options = {}) {
  const { folder, page = 1, limit = 20 } = options;
  const offset = (page - 1) * limit;

  let query = 'SELECT * FROM files WHERE user_id = $1';
  const values = [userId];

  if (folder) {
    query += ' AND folder = $2';
    values.push(folder);
  }

  query += ' ORDER BY created_at DESC LIMIT $' + (values.length + 1) + ' OFFSET $' + (values.length + 2);
  values.push(limit, offset);

  const [files, countResult] = await Promise.all([
    pool.query(query, values),
    pool.query(
      `SELECT COUNT(*) FROM files WHERE user_id = $1${folder ? ' AND folder = $2' : ''}`,
      folder ? [userId, folder] : [userId]
    )
  ]);

  const total = parseInt(countResult.rows[0].count);

  return {
    files: files.rows,
    pagination: {
      page,
      limit,
      total,
      pages: Math.ceil(total / limit)
    }
  };
}

async function deleteFile(fileId, userId) {
  const file = await getFile(fileId);

  if (file.user_id !== userId) {
    throw new Error('Access denied');
  }

  if (fs.existsSync(file.path)) {
    fs.unlinkSync(file.path);
  }

  await pool.query('DELETE FROM files WHERE id = $1', [fileId]);

  return { deleted: true, fileId };
}

async function deleteFolder(userId, folder) {
  const result = await pool.query(
    'SELECT path FROM files WHERE user_id = $1 AND folder = $2',
    [userId, folder]
  );

  for (const file of result.rows) {
    if (fs.existsSync(file.path)) {
      fs.unlinkSync(file.path);
    }
  }

  await pool.query('DELETE FROM files WHERE user_id = $1 AND folder = $2', [userId, folder]);

  return { deleted: true, folder, count: result.rows.length };
}

async function getStorageStats(userId) {
  const result = await pool.query(
    `SELECT
      COUNT(*) as total_files,
      COALESCE(SUM(size), 0) as total_size,
      folder,
      mime_type
    FROM files
    WHERE user_id = $1
    GROUP BY folder, mime_type`,
    [userId]
  );

  const stats = {
    totalFiles: 0,
    totalSize: 0,
    byFolder: {},
    byType: {}
  };

  for (const row of result.rows) {
    stats.totalFiles += parseInt(row.total_files);
    stats.totalSize += parseInt(row.total_size);

    if (!stats.byFolder[row.folder]) {
      stats.byFolder[row.folder] = { count: 0, size: 0 };
    }
    stats.byFolder[row.folder].count += parseInt(row.total_files);
    stats.byFolder[row.folder].size += parseInt(row.total_size);

    if (!stats.byType[row.mime_type]) {
      stats.byType[row.mime_type] = { count: 0, size: 0 };
    }
    stats.byType[row.mime_type].count += parseInt(row.total_files);
    stats.byType[row.mime_type].size += parseInt(row.total_size);
  }

  return stats;
}

async function copyFile(fileId, targetFolder, userId) {
  const file = await getFile(fileId);

  if (file.user_id !== userId) {
    throw new Error('Access denied');
  }

  const newFileId = uuidv4();
  const newStoredName = `${newFileId}${path.extname(file.stored_name)}`;
  const targetPath = path.join(__dirname, '../../../uploads', targetFolder, newStoredName);

  if (!fs.existsSync(path.dirname(targetPath))) {
    fs.mkdirSync(path.dirname(targetPath), { recursive: true });
  }

  fs.copyFileSync(file.path, targetPath);

  const fileRecord = {
    id: newFileId,
    originalName: file.original_name,
    storedName: newStoredName,
    mimeType: file.mime_type,
    size: file.size,
    folder: targetFolder,
    path: targetPath,
    userId,
    checksum: file.checksum
  };

  await saveFileRecord(fileRecord);

  return {
    id: newFileId,
    originalName: file.original_name,
    url: `/uploads/${targetFolder}/${newStoredName}`,
    mimeType: file.mime_type,
    size: file.size
  };
}

async function moveFile(fileId, targetFolder, userId) {
  const copiedFile = await copyFile(fileId, targetFolder, userId);
  await deleteFile(fileId, userId);

  return copiedFile;
}

module.exports = {
  uploadFile,
  getFile,
  getUserFiles,
  deleteFile,
  deleteFolder,
  getStorageStats,
  copyFile,
  moveFile,
  saveFileRecord,
  STORAGE_PROVIDERS
};
