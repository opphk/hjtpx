jest.mock('../../../../src/config/database/db', () => ({
  query: jest.fn(),
  pool: {
    on: jest.fn(),
    query: jest.fn(),
    connect: jest.fn(),
    totalCount: 0,
    idleCount: 0,
    waitingCount: 0
  }
}));

const {
  validateInput,
  validateObject,
  escapeIdentifier,
  checkForSqlInjection,
  createSafeQueryBuilder,
  buildWhereClause,
  buildUpdateSet
} = require('../../utils/sqlInjectionProtection');

describe('SQL Injection Protection', () => {
  describe('validateInput', () => {
    test('should detect SQL keywords', () => {
      const maliciousInputs = [
        "'; DROP TABLE users; --",
        '1 OR 1=1',
        '1; DELETE FROM users WHERE 1=1',
        "admin'--",
        '1 UNION SELECT password FROM users'
      ];

      maliciousInputs.forEach(input => {
        const result = validateInput(input);
        expect(result.valid).toBe(false);
      });
    });

    test('should allow safe input', () => {
      const safeInputs = ['John Doe', 'Normal text without special chars'];

      safeInputs.forEach(input => {
        const result = validateInput(input);
        expect(result.valid).toBe(true);
      });
    });

    test('should handle null and undefined', () => {
      expect(validateInput(null).valid).toBe(true);
      expect(validateInput(undefined).valid).toBe(true);
    });

    test('should respect maxLength option', () => {
      const longInput = 'a'.repeat(500);
      const result = validateInput(longInput, { maxLength: 100 });
      expect(result.sanitized.length).toBe(100);
    });

    test('should sanitize SQL wildcards when not allowed', () => {
      const input = 'test%value';
      const result = validateInput(input, { allowSqlWildcards: false });
      expect(result.sanitized).toBe('test\\%value');
    });
  });

  describe('validateObject', () => {
    test('should validate object with malicious values', () => {
      const obj = {
        username: "admin' OR '1'='1",
        email: 'valid@email.com'
      };

      const result = validateObject(obj);
      expect(result.errors.length).toBeGreaterThan(0);
      expect(result.errors[0].field).toBe('username');
    });

    test('should preserve safe values', () => {
      const obj = {
        username: 'john'
      };

      const result = validateObject(obj);
      expect(result.errors.length).toBe(0);
    });

    test('should handle nested objects', () => {
      const obj = {
        user: {
          name: "Robert'; DROP TABLE Users;--"
        }
      };

      const result = validateObject(obj);
      expect(result.errors.length).toBeGreaterThan(0);
    });

    test('should handle arrays in objects', () => {
      const obj = {
        ids: ['1', '2', '3']
      };

      const result = validateObject(obj);
      expect(result.errors.length).toBe(0);
    });
  });

  describe('escapeIdentifier', () => {
    test('should escape valid identifiers', () => {
      expect(escapeIdentifier('username')).toBe('"username"');
      expect(escapeIdentifier('user_id')).toBe('"user_id"');
    });

    test('should reject invalid identifiers', () => {
      expect(() => escapeIdentifier('123invalid')).toThrow();
      expect(() => escapeIdentifier('user-name')).toThrow();
      expect(() => escapeIdentifier('user name')).toThrow();
      expect(() => escapeIdentifier('user;DROP')).toThrow();
    });

    test('should reject reserved keywords', () => {
      expect(() => escapeIdentifier('SELECT')).toThrow();
      expect(() => escapeIdentifier('DROP')).toThrow();
      expect(() => escapeIdentifier('DELETE')).toThrow();
    });
  });

  describe('checkForSqlInjection', () => {
    test('should detect injection attempts', () => {
      expect(checkForSqlInjection("' OR 1=1 --")).toBe(true);
      expect(checkForSqlInjection("'; DELETE FROM users")).toBe(true);
      expect(checkForSqlInjection('1 UNION SELECT *')).toBe(true);
    });

    test('should pass safe strings', () => {
      expect(checkForSqlInjection('normal text')).toBe(false);
    });

    test('should handle non-string inputs', () => {
      expect(checkForSqlInjection(123)).toBe(false);
      expect(checkForSqlInjection(null)).toBe(false);
    });
  });

  describe('buildWhereClause', () => {
    test('should build simple equality condition', () => {
      const result = buildWhereClause({ status: 'active' });
      expect(result.clause).toBe('WHERE "status" = $1');
      expect(result.params).toEqual(['active']);
    });

    test('should build IN clause for arrays', () => {
      const result = buildWhereClause({ id: [1, 2, 3] });
      expect(result.clause).toBe('WHERE "id" IN ($1, $2, $3)');
      expect(result.params).toEqual([1, 2, 3]);
    });

    test('should build IS NULL for null values', () => {
      const result = buildWhereClause({ deleted_at: null });
      expect(result.clause).toBe('WHERE "deleted_at" IS NULL');
      expect(result.params).toEqual([]);
    });

    test('should build multiple conditions', () => {
      const result = buildWhereClause({
        status: 'active',
        role: 'admin'
      });
      expect(result.clause).toBe('WHERE "status" = $1 AND "role" = $2');
      expect(result.params).toEqual(['active', 'admin']);
    });
  });

  describe('buildUpdateSet', () => {
    test('should build update set clause', () => {
      const result = buildUpdateSet({ name: 'John', email: 'john@example.com' });
      expect(result.clause).toBe('SET "name" = $1, "email" = $2');
      expect(result.params).toEqual(['John', 'john@example.com']);
    });

    test('should exclude undefined values', () => {
      const result = buildUpdateSet({ name: 'John', email: undefined });
      expect(result.clause).toBe('SET "name" = $1');
      expect(result.params).toEqual(['John']);
    });
  });

  describe('createSafeQueryBuilder', () => {
    test('should build SELECT query', () => {
      const builder = createSafeQueryBuilder('users');
      const result = builder.select(['id', 'name'], { status: 'active' });

      expect(result.query).toContain('SELECT "id", "name"');
      expect(result.query).toContain('FROM "users"');
      expect(result.query).toContain('WHERE "status" = $1');
      expect(result.params).toEqual(['active']);
    });

    test('should build INSERT query', () => {
      const builder = createSafeQueryBuilder('users');
      const result = builder.insert({ name: 'John', email: 'john@example.com' });

      expect(result.query).toContain('INSERT INTO "users"');
      expect(result.query).toContain('("name", "email")');
      expect(result.query).toContain('VALUES ($1, $2)');
      expect(result.params).toEqual(['John', 'john@example.com']);
    });

    test('should build UPDATE query', () => {
      const builder = createSafeQueryBuilder('users');
      const result = builder.update({ name: 'Jane' }, { id: 1 });

      expect(result.query).toContain('UPDATE "users"');
      expect(result.query).toContain('SET "name" = $1');
      expect(result.query).toContain('WHERE "id" = $1');
      expect(result.params).toEqual(['Jane', 1]);
    });

    test('should build DELETE query', () => {
      const builder = createSafeQueryBuilder('users');
      const result = builder.delete({ id: 1 });

      expect(result.query).toContain('DELETE FROM "users"');
      expect(result.query).toContain('WHERE "id" = $1');
      expect(result.params).toEqual([1]);
    });

    test('should reject invalid table names', () => {
      expect(() => createSafeQueryBuilder('123invalid')).toThrow();
      expect(() => createSafeQueryBuilder('DROP TABLE')).toThrow();
    });
  });
});
