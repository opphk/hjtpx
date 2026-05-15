import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, test, expect, vi, beforeEach } from 'vitest';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { errorHandler, handleApiError, AppError } from '../utils/errorHandler';
import { ErrorCodes, getErrorMessage, getErrorCategory } from '../utils/errorCodes';
import { AppError as AppErrorClass, ValidationError, AuthenticationError } from '../utils/errorClasses';

describe('Error Codes', () => {
  test('should export all error code categories', () => {
    expect(ErrorCodes.AUTHENTICATION_ERRORS).toBeDefined();
    expect(ErrorCodes.VALIDATION_ERRORS).toBeDefined();
    expect(ErrorCodes.SERVER_ERRORS).toBeDefined();
    expect(ErrorCodes.SECURITY_ERRORS).toBeDefined();
  });

  test('should return correct category for error code', () => {
    expect(getErrorCategory('AUTH_001')).toBe('Authentication');
    expect(getErrorCategory('VAL_001')).toBe('Validation');
    expect(getErrorCategory('SRV_001')).toBe('Server');
    expect(getErrorCategory('SEC_001')).toBe('Security');
    expect(getErrorCategory('UNKNOWN')).toBe('Unknown');
  });

  test('should return correct error messages', () => {
    expect(getErrorMessage('AUTH_001')).toBe('Invalid credentials');
    expect(getErrorMessage('AUTH_002')).toBe('Your session has expired');
    expect(getErrorMessage('SRV_004')).toBe('Too many requests, please try again later');
  });

  test('should return default message for unknown code', () => {
    expect(getErrorMessage('UNKNOWN_CODE', 'Custom message')).toBe('Custom message');
    expect(getErrorMessage('UNKNOWN_CODE')).toBe('An error occurred');
  });
});

describe('AppError Classes', () => {
  test('should create AppError with correct properties', () => {
    const error = new AppErrorClass('AUTH_001', 'Test error', { field: 'test' }, 401);

    expect(error.code).toBe('AUTH_001');
    expect(error.message).toBe('Test error');
    expect(error.details).toEqual({ field: 'test' });
    expect(error.statusCode).toBe(401);
    expect(error.category).toBe('Authentication');
    expect(error.timestamp).toBeDefined();
  });

  test('should create ValidationError with correct defaults', () => {
    const error = new ValidationError('Validation failed', { errors: ['field1', 'field2'] });

    expect(error.code).toBe('VAL_001');
    expect(error.statusCode).toBe(400);
    expect(error.details).toEqual({ errors: ['field1', 'field2'] });
  });

  test('should create AuthenticationError with correct defaults', () => {
    const error = new AuthenticationError();

    expect(error.code).toBe('AUTH_001');
    expect(error.statusCode).toBe(401);
    expect(error.message).toBe('Authentication failed');
  });

  test('AppError toJSON should return correct format', () => {
    const error = new AppErrorClass('SRV_001', 'Server error', null, 500);
    const json = error.toJSON();

    expect(json.success).toBe(false);
    expect(json.error.code).toBe('SRV_001');
    expect(json.error.message).toBe('Server error');
    expect(json.error.statusCode).toBe(500);
    expect(json.error.timestamp).toBeDefined();
  });
});

describe('ErrorHandler Service', () => {
  beforeEach(() => {
    errorHandler.clearErrorLog();
  });

  test('should add and remove listeners', () => {
    const callback = vi.fn();
    const removeListener = errorHandler.addListener(callback);

    expect(errorHandler.errorListeners.size).toBe(1);

    removeListener();
    expect(errorHandler.errorListeners.size).toBe(0);
  });

  test('should log errors', () => {
    const error = new AppErrorClass('AUTH_001', 'Test error');

    errorHandler.logError(error);

    const log = errorHandler.getErrorLog();
    expect(log.length).toBe(1);
    expect(log[0].code).toBe('AUTH_001');
  });

  test('should handle API response errors', () => {
    const mockResponse = {
      data: {
        success: false,
        error: {
          code: 'AUTH_002',
          message: 'Token expired',
          details: null
        }
      }
    };

    const error = handleApiError({ response: mockResponse });

    expect(error.code).toBe('AUTH_002');
    expect(error.message).toBe('Token expired');
  });

  test('should handle network errors', () => {
    const error = errorHandler.handleNetworkError(new Error('Network failed'));

    expect(error).toBeInstanceOf(AppErrorClass);
    expect(error.statusCode).toBe(503);
  });

  test('should handle timeout errors', () => {
    const error = errorHandler.handleTimeoutError(new Error('Timeout'));

    expect(error).toBeInstanceOf(AppErrorClass);
    expect(error.statusCode).toBe(408);
  });

  test('should get error summary', () => {
    errorHandler.logError(new AppErrorClass('AUTH_001', 'Error 1'));
    errorHandler.logError(new AppErrorClass('AUTH_002', 'Error 2'));
    errorHandler.logError(new AppErrorClass('VAL_001', 'Error 3'));

    const summary = errorHandler.getErrorSummary();

    expect(summary.total).toBe(3);
    expect(summary.byCategory.AUTH).toBe(2);
    expect(summary.byCategory.VAL).toBe(1);
  });

  test('should determine if error should retry', () => {
    expect(errorHandler.shouldRetry(new AppErrorClass('SRV_003', 'Timeout', null, 408))).toBe(true);
    expect(errorHandler.shouldRetry(new AppErrorClass('SRV_004', 'Rate limit', null, 429))).toBe(true);
    expect(errorHandler.shouldRetry(new AppErrorClass('SRV_001', 'Error', null, 500))).toBe(false);
  });

  test('should clear error log', () => {
    errorHandler.logError(new AppErrorClass('AUTH_001', 'Test'));
    expect(errorHandler.getErrorLog().length).toBeGreaterThan(0);

    errorHandler.clearErrorLog();
    expect(errorHandler.getErrorLog().length).toBe(0);
  });
});
