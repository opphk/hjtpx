# File Storage Guide

## Overview

This document describes the file storage and management system implemented in the HJTPX project.

## Storage Providers

The system supports multiple storage providers:

- **Local**: Default storage on server filesystem
- **Amazon S3**: Cloud storage on AWS S3
- **Google Cloud Storage**: Cloud storage on GCS
- **Azure Blob Storage**: Cloud storage on Azure

### Configuration

Set the storage provider in environment variables:

```bash
STORAGE_PROVIDER=local  # or s3, gcs, azure
```

## File Upload

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/files/upload` | POST | Upload files |
| `/api/v1/files` | GET | List user files |
| `/api/v1/files/:id` | GET | Get file details |
| `/api/v1/files/:id/download` | GET | Download file |
| `/api/v1/files/:id` | DELETE | Delete file |
| `/api/v1/files/stats` | GET | Get storage statistics |

### Upload Request

```bash
curl -X POST http://localhost:3000/api/v1/files/upload \
  -H "Authorization: Bearer <token>" \
  -F "files=@document.pdf" \
  -F "folder=documents"
```

### Response

```json
{
  "success": true,
  "data": {
    "files": [
      {
        "id": "uuid",
        "originalName": "document.pdf",
        "storedName": "uuid.pdf",
        "url": "/uploads/documents/uuid.pdf",
        "mimeType": "application/pdf",
        "size": 102400,
        "checksum": "md5hash"
      }
    ],
    "count": 1
  }
}
```

## File Validation

### Allowed File Types

| Type | Extensions |
|------|------------|
| image | jpg, jpeg, png, gif, webp, svg |
| document | pdf, doc, docx, txt, rtf |
| csv | csv |
| json | json |

### Size Limits

| Type | Max Size |
|------|----------|
| Default | 10 MB |
| Image | 5 MB |
| Document | 10 MB |
| CSV | 10 MB |
| Archive | 50 MB |

### Validation Rules

The `fileValidator` utility validates:
- File extension
- MIME type (using file-type)
- File size
- Filename sanitization

```javascript
const { validateFile } = require('./utils/fileValidator');

const result = validateFile('document.pdf', buffer, ['document']);
console.log(result);
// { valid: true, errors: [], details: {...} }
```

## File Management

### Folder Organization

Files are organized into folders:
- `general`: Default folder
- `documents`: User documents
- `images`: User images
- `imports`: Import files
- `exports`: Export files

### File Operations

**Copy File**
```bash
POST /api/v1/files/:id/copy
{
  "targetFolder": "backup"
}
```

**Move File**
```bash
POST /api/v1/files/:id/move
{
  "targetFolder": "archive"
}
```

**Delete File**
```bash
DELETE /api/v1/files/:id
```

**Delete Folder**
```bash
DELETE /api/v1/files/folder/:folderName
```

## Storage Statistics

Get storage usage statistics:

```bash
GET /api/v1/files/stats
```

Response:
```json
{
  "success": true,
  "data": {
    "totalFiles": 150,
    "totalSize": 52428800,
    "byFolder": {
      "documents": { "count": 50, "size": 10485760 },
      "images": { "count": 100, "size": 41943040 }
    },
    "byType": {
      "application/pdf": { "count": 20, "size": 5242880 }
    }
  }
}
```

## Cloud Storage Setup

### Amazon S3

```bash
S3_BUCKET=my-bucket
S3_REGION=us-east-1
S3_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
S3_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

### Google Cloud Storage

```bash
GCS_BUCKET=my-bucket
GCS_PROJECT_ID=my-project
GCS_KEY_FILE=/path/to/key.json
```

### Azure Blob Storage

```bash
AZURE_STORAGE_CONNECTION_STRING=DefaultEndpointsProtocol=https;AccountName=...
AZURE_STORAGE_CONTAINER=uploads
```

## Image Processing

Enable image processing for automatic resizing:

```bash
IMAGE_PROCESSING_ENABLED=true
IMAGE_MAX_WIDTH=1920
IMAGE_MAX_HEIGHT=1080
IMAGE_QUALITY=85
GENERATE_THUMBNAILS=true
THUMBNAIL_WIDTH=200
THUMBNAIL_HEIGHT=200
```

## CDN Integration

Configure CDN for faster file delivery:

```bash
CDN_ENABLED=true
CDN_URL=https://cdn.example.com
CDN_INVALIDATION_ENABLED=true
```

## Cleanup

Configure automatic file cleanup:

```bash
FILE_CLEANUP_ENABLED=true
ORPHANED_FILES_DAYS=7
TEMP_FILES_HOURS=24
FILE_CLEANUP_CRON=0 2 * * *
```

## Security

### Best Practices

1. **File Validation**: Always validate files on upload
2. **Size Limits**: Enforce file size limits
3. **Type Restrictions**: Only allow necessary file types
4. **Storage Isolation**: Separate user file storage
5. **Access Control**: Implement proper authorization
6. **Virus Scanning**: Enable virus scanning for uploads
7. **Checksum Verification**: Verify file integrity

### Security Checklist

- [ ] Validate file extensions
- [ ] Check MIME types
- [ ] Limit file sizes
- [ ] Sanitize filenames
- [ ] Implement rate limiting
- [ ] Use secure upload tokens
- [ ] Enable audit logging
- [ ] Regular security audits

## Troubleshooting

### Common Issues

1. **Upload Fails**
   - Check file size limits
   - Verify allowed extensions
   - Check disk space

2. **File Not Found**
   - Verify file path
   - Check file permissions
   - Ensure file exists

3. **Storage Quota Exceeded**
   - Check storage limits
   - Clean up old files
   - Implement cleanup job

## Performance Optimization

### Recommendations

1. Use CDN for static assets
2. Implement lazy loading
3. Use compression for large files
4. Enable browser caching
5. Implement file chunking for large uploads
6. Use async processing for thumbnails
