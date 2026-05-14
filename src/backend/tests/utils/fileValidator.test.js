describe('File Validator', () => {
  const ALLOWED_EXTENSIONS = ['.jpg', '.jpeg', '.png', '.gif', '.pdf', '.doc', '.docx', '.xls', '.xlsx'];
  const MAX_FILE_SIZE = 10 * 1024 * 1024;
  const MAX_IMAGE_SIZE = 5 * 1024 * 1024;
  const IMAGE_EXTENSIONS = ['.jpg', '.jpeg', '.png', '.gif'];

  const isValidExtension = (filename, allowed = ALLOWED_EXTENSIONS) => {
    const ext = filename.toLowerCase().substring(filename.lastIndexOf('.'));
    return allowed.includes(ext);
  };

  const isValidFileSize = (size, isImage = false) => {
    const maxSize = isImage ? MAX_IMAGE_SIZE : MAX_FILE_SIZE;
    return size > 0 && size <= maxSize;
  };

  const validateFile = (file, options = {}) => {
    const errors = [];

    if (!file.name) {
      errors.push('File name is required');
    }

    if (!isValidExtension(file.name, options.allowed || ALLOWED_EXTENSIONS)) {
      errors.push('Invalid file extension');
    }

    if (!isValidFileSize(file.size, options.isImage)) {
      errors.push('File size exceeds maximum allowed');
    }

    return { valid: errors.length === 0, errors };
  };

  describe('Extension Validation', () => {
    test('should accept valid image extensions', () => {
      expect(isValidExtension('photo.jpg')).toBe(true);
      expect(isValidExtension('photo.JPEG')).toBe(true);
      expect(isValidExtension('image.png')).toBe(true);
      expect(isValidExtension('graphic.gif')).toBe(true);
    });

    test('should accept valid document extensions', () => {
      expect(isValidExtension('document.pdf')).toBe(true);
      expect(isValidExtension('file.doc')).toBe(true);
      expect(isValidExtension('data.xlsx')).toBe(true);
    });

    test('should reject invalid extensions', () => {
      expect(isValidExtension('file.exe')).toBe(false);
      expect(isValidExtension('script.js')).toBe(false);
      expect(isValidExtension('file.sh')).toBe(false);
    });

    test('should reject files without extension', () => {
      expect(isValidExtension('filewithout Extension')).toBe(false);
    });
  });

  describe('File Size Validation', () => {
    test('should accept valid file sizes', () => {
      expect(isValidFileSize(1024)).toBe(true);
      expect(isValidFileSize(MAX_FILE_SIZE)).toBe(true);
      expect(isValidFileSize(1)).toBe(true);
    });

    test('should reject oversized files', () => {
      expect(isValidFileSize(MAX_FILE_SIZE + 1)).toBe(false);
      expect(isValidFileSize(100 * 1024 * 1024)).toBe(false);
    });

    test('should reject empty files', () => {
      expect(isValidFileSize(0)).toBe(false);
    });

    test('should handle image size limits', () => {
      expect(isValidFileSize(MAX_IMAGE_SIZE, true)).toBe(true);
      expect(isValidFileSize(MAX_IMAGE_SIZE + 1, true)).toBe(false);
    });
  });

  describe('Complete File Validation', () => {
    test('should validate a valid image file', () => {
      const file = { name: 'photo.jpg', size: 1024 * 1024 };
      const result = validateFile(file, { isImage: true });
      expect(result.valid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    test('should validate a valid document', () => {
      const file = { name: 'report.pdf', size: 2048 * 1024 };
      const result = validateFile(file);
      expect(result.valid).toBe(true);
    });

    test('should reject invalid extension', () => {
      const file = { name: 'virus.exe', size: 1024 };
      const result = validateFile(file);
      expect(result.valid).toBe(false);
      expect(result.errors).toContain('Invalid file extension');
    });

    test('should reject oversized file', () => {
      const file = { name: 'large.pdf', size: 50 * 1024 * 1024 };
      const result = validateFile(file);
      expect(result.valid).toBe(false);
      expect(result.errors).toContain('File size exceeds maximum allowed');
    });

    test('should reject file without name', () => {
      const file = { name: '', size: 1024 };
      const result = validateFile(file);
      expect(result.valid).toBe(false);
      expect(result.errors).toContain('File name is required');
    });

    test('should reject file without size property', () => {
      const file = { name: 'file.pdf' };
      const result = validateFile(file);
      expect(result.valid).toBe(false);
    });
  });

  describe('Custom Allowed Extensions', () => {
    test('should respect custom allowed extensions', () => {
      const customAllowed = ['.jpg', '.png'];
      expect(isValidExtension('photo.jpg', customAllowed)).toBe(true);
      expect(isValidExtension('photo.png', customAllowed)).toBe(true);
      expect(isValidExtension('photo.gif', customAllowed)).toBe(false);
    });
  });
});
