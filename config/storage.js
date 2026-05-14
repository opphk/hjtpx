module.exports = {
  storage: {
    provider: process.env.STORAGE_PROVIDER || 'local',
    local: {
      uploadDir: process.env.LOCAL_UPLOAD_DIR || './uploads',
      maxFileSize: parseInt(process.env.LOCAL_MAX_FILE_SIZE) || 10485760,
      allowedExtensions: process.env.LOCAL_ALLOWED_EXTENSIONS
        ? process.env.LOCAL_ALLOWED_EXTENSIONS.split(',')
        : ['jpg', 'jpeg', 'png', 'gif', 'webp', 'pdf', 'doc', 'docx', 'csv', 'json']
    },
    s3: {
      bucket: process.env.S3_BUCKET || '',
      region: process.env.S3_REGION || 'us-east-1',
      accessKeyId: process.env.S3_ACCESS_KEY_ID || '',
      secretAccessKey: process.env.S3_SECRET_ACCESS_KEY || '',
      endpoint: process.env.S3_ENDPOINT || null,
      acl: process.env.S3_ACL || 'private',
      pathPrefix: process.env.S3_PATH_PREFIX || 'uploads/'
    },
    gcs: {
      bucket: process.env.GCS_BUCKET || '',
      projectId: process.env.GCS_PROJECT_ID || '',
      keyFilename: process.env.GCS_KEY_FILE || null,
      acl: process.env.GCS_ACL || 'private',
      pathPrefix: process.env.GCS_PATH_PREFIX || 'uploads/'
    },
    azure: {
      connectionString: process.env.AZURE_STORAGE_CONNECTION_STRING || '',
      container: process.env.AZURE_STORAGE_CONTAINER || 'uploads',
      blobPrefix: process.env.AZURE_BLOB_PREFIX || 'uploads/'
    }
  },

  fileLimits: {
    maxFileSize: parseInt(process.env.MAX_FILE_SIZE) || 10485760,
    maxFiles: parseInt(process.env.MAX_FILES) || 10,
    maxTotalSize: parseInt(process.env.MAX_TOTAL_SIZE) || 104857600
  },

  imageProcessing: {
    enabled: process.env.IMAGE_PROCESSING_ENABLED !== 'false',
    maxWidth: parseInt(process.env.IMAGE_MAX_WIDTH) || 1920,
    maxHeight: parseInt(process.env.IMAGE_MAX_HEIGHT) || 1080,
    quality: parseInt(process.env.IMAGE_QUALITY) || 85,
    generateThumbnails: process.env.GENERATE_THUMBNAILS === 'true',
    thumbnailWidth: parseInt(process.env.THUMBNAIL_WIDTH) || 200,
    thumbnailHeight: parseInt(process.env.THUMBNAIL_HEIGHT) || 200
  },

  virusScanning: {
    enabled: process.env.VIRUS_SCAN_ENABLED === 'true',
    provider: process.env.VIRUS_SCAN_PROVIDER || 'clamav',
    clamav: {
      host: process.env.CLAMAV_HOST || 'localhost',
      port: parseInt(process.env.CLAMAV_PORT) || 3310
    }
  },

  cdn: {
    enabled: process.env.CDN_ENABLED === 'true',
    url: process.env.CDN_URL || '',
    invalidation: {
      enabled: process.env.CDN_INVALIDATION_ENABLED === 'true',
      path: process.env.CDN_INVALIDATION_PATH || '/api/cdn/invalidate'
    }
  },

  cleanup: {
    enabled: process.env.FILE_CLEANUP_ENABLED === 'true',
    orphanedFilesDays: parseInt(process.env.ORPHANED_FILES_DAYS) || 7,
    tempFilesHours: parseInt(process.env.TEMP_FILES_HOURS) || 24,
    cronSchedule: process.env.FILE_CLEANUP_CRON || '0 2 * * *'
  }
};
