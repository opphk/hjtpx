const express = require('express');
const router = express.Router();
const path = require('path');
const fileService = require('../services/fileService');
const { authMiddleware } = require('../middleware/auth');
const { uploadMiddleware } = require('../middleware/upload');
const { requireRole } = require('../middleware/roleCheck');

router.use(authMiddleware);

router.post('/upload', uploadMiddleware.array('files', 10), async (req, res, next) => {
  try {
    const { folder = 'general' } = req.body;
    const files = req.files;

    if (!files || files.length === 0) {
      return res.status(400).json({
        success: false,
        error: 'No files uploaded'
      });
    }

    const uploadPromises = files.map(file =>
      fileService.uploadFile(file, {
        userId: req.user.id,
        folder
      })
    );

    const uploadedFiles = await Promise.all(uploadPromises);

    res.json({
      success: true,
      data: {
        files: uploadedFiles,
        count: uploadedFiles.length
      }
    });
  } catch (error) {
    next(error);
  }
});

router.get('/', async (req, res, next) => {
  try {
    const { folder, page, limit } = req.query;

    const result = await fileService.getUserFiles(req.user.id, {
      folder,
      page: parseInt(page) || 1,
      limit: parseInt(limit) || 20
    });

    res.json({
      success: true,
      data: result.files,
      pagination: result.pagination
    });
  } catch (error) {
    next(error);
  }
});

router.get('/stats', async (req, res, next) => {
  try {
    const stats = await fileService.getStorageStats(req.user.id);

    res.json({
      success: true,
      data: stats
    });
  } catch (error) {
    next(error);
  }
});

router.get('/:id', async (req, res, next) => {
  try {
    const file = await fileService.getFile(req.params.id);

    if (file.user_id !== req.user.id && req.user.role !== 'admin') {
      return res.status(403).json({
        success: false,
        error: 'Access denied'
      });
    }

    res.json({
      success: true,
      data: file
    });
  } catch (error) {
    next(error);
  }
});

router.get('/:id/download', async (req, res, next) => {
  try {
    const file = await fileService.getFile(req.params.id);

    if (file.user_id !== req.user.id && req.user.role !== 'admin') {
      return res.status(403).json({
        success: false,
        error: 'Access denied'
      });
    }

    res.download(file.path, file.original_name);
  } catch (error) {
    next(error);
  }
});

router.post('/:id/copy', async (req, res, next) => {
  try {
    const { targetFolder } = req.body;

    if (!targetFolder) {
      return res.status(400).json({
        success: false,
        error: 'Target folder is required'
      });
    }

    const copiedFile = await fileService.copyFile(req.params.id, targetFolder, req.user.id);

    res.json({
      success: true,
      data: copiedFile
    });
  } catch (error) {
    next(error);
  }
});

router.post('/:id/move', async (req, res, next) => {
  try {
    const { targetFolder } = req.body;

    if (!targetFolder) {
      return res.status(400).json({
        success: false,
        error: 'Target folder is required'
      });
    }

    const movedFile = await fileService.moveFile(req.params.id, targetFolder, req.user.id);

    res.json({
      success: true,
      data: movedFile
    });
  } catch (error) {
    next(error);
  }
});

router.delete('/:id', async (req, res, next) => {
  try {
    await fileService.deleteFile(req.params.id, req.user.id);

    res.json({
      success: true,
      message: 'File deleted successfully'
    });
  } catch (error) {
    next(error);
  }
});

router.delete('/folder/:folder', async (req, res, next) => {
  try {
    const result = await fileService.deleteFolder(req.user.id, req.params.folder);

    res.json({
      success: true,
      data: result
    });
  } catch (error) {
    next(error);
  }
});

module.exports = router;
