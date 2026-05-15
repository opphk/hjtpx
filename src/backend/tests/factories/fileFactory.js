const fs = require('fs');
const path = require('path');

const pool = require('../../../config/database/db');

const defaultFileAttributes = {
  filename: 'test-file.txt',
  originalName: 'test-file.txt',
  mimeType: 'text/plain',
  size: 1024,
  folder: 'test'
};

async function createFile(userId, overrides = {}) {
  const attributes = { ...defaultFileAttributes, ...overrides };

  const result = await pool.query(
    `INSERT INTO files (user_id, filename, original_name, mime_type, size, folder, file_path) 
     VALUES ($1, $2, $3, $4, $5, $6, $7) 
     RETURNING *`,
    [
      userId,
      attributes.filename,
      attributes.originalName,
      attributes.mimeType,
      attributes.size,
      attributes.folder,
      `/uploads/${attributes.folder}/${attributes.filename}`
    ]
  );

  return result.rows[0];
}

async function createMultipleFiles(userId, count, overrides = {}) {
  const files = [];
  for (let i = 0; i < count; i++) {
    const file = await createFile(userId, {
      ...overrides,
      filename: `test-file-${i}-${Date.now()}.txt`,
      originalName: `test-file-${i}.txt`
    });
    files.push(file);
  }
  return files;
}

async function deleteFile(fileId) {
  try {
    await pool.query('DELETE FROM files WHERE id = $1', [fileId]);
  } catch (error) {
    console.error('Error deleting file:', error);
  }
}

async function deleteFiles(fileIds) {
  for (const fileId of fileIds) {
    await deleteFile(fileId);
  }
}

async function deleteUserFiles(userId) {
  try {
    await pool.query('DELETE FROM files WHERE user_id = $1', [userId]);
  } catch (error) {
    console.error('Error deleting user files:', error);
  }
}

module.exports = {
  createFile,
  createMultipleFiles,
  deleteFile,
  deleteFiles,
  deleteUserFiles,
  defaultFileAttributes
};
