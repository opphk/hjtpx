jest.mock('../../../config/database/db');

describe('Query Builder', () => {
  const buildSelectQuery = (table, columns = ['*'], conditions = {}) => {
    let query = `SELECT ${columns.join(', ')} FROM ${table}`;
    const params = [];
    
    if (Object.keys(conditions).length > 0) {
      const whereClauses = [];
      let paramIndex = 1;
      
      for (const [key, value] of Object.entries(conditions)) {
        if (value === null) {
          whereClauses.push(`${key} IS NULL`);
        } else if (Array.isArray(value)) {
          whereClauses.push(`${key} IN (${value.map(() => `$${paramIndex++}`).join(', ')})`);
          params.push(...value);
        } else {
          whereClauses.push(`${key} = $${paramIndex++}`);
          params.push(value);
        }
      }
      
      query += ` WHERE ${whereClauses.join(' AND ')}`;
    }
    
    return { query, params };
  };

  const buildInsertQuery = (table, data) => {
    const columns = Object.keys(data);
    const values = Object.values(data);
    const paramIndex = values.map((_, i) => `$${i + 1}`);
    
    return {
      query: `INSERT INTO ${table} (${columns.join(', ')}) VALUES (${paramIndex.join(', ')})`,
      params: values
    };
  };

  const buildUpdateQuery = (table, data, conditions = {}) => {
    const setClauses = [];
    const params = [];
    let paramIndex = 1;
    
    for (const [key, value] of Object.entries(data)) {
      if (value === null) {
        setClauses.push(`${key} = NULL`);
      } else {
        setClauses.push(`${key} = $${paramIndex++}`);
        params.push(value);
      }
    }
    
    let query = `UPDATE ${table} SET ${setClauses.join(', ')}`;
    
    if (Object.keys(conditions).length > 0) {
      const whereClauses = [];
      for (const [key, value] of Object.entries(conditions)) {
        whereClauses.push(`${key} = $${paramIndex++}`);
        params.push(value);
      }
      query += ` WHERE ${whereClauses.join(' AND ')}`;
    }
    
    return { query, params };
  };

  const buildDeleteQuery = (table, conditions = {}) => {
    let query = `DELETE FROM ${table}`;
    const params = [];
    
    if (Object.keys(conditions).length > 0) {
      const whereClauses = [];
      let paramIndex = 1;
      
      for (const [key, value] of Object.entries(conditions)) {
        whereClauses.push(`${key} = $${paramIndex++}`);
        params.push(value);
      }
      query += ` WHERE ${whereClauses.join(' AND ')}`;
    }
    
    return { query, params };
  };

  describe('SELECT queries', () => {
    test('should build simple SELECT query', () => {
      const { query, params } = buildSelectQuery('users');
      expect(query).toBe('SELECT * FROM users');
      expect(params).toEqual([]);
    });

    test('should build SELECT with specific columns', () => {
      const { query, params } = buildSelectQuery('users', ['id', 'email', 'name']);
      expect(query).toBe('SELECT id, email, name FROM users');
      expect(params).toEqual([]);
    });

    test('should build SELECT with equality condition', () => {
      const { query, params } = buildSelectQuery('users', ['*'], { status: 'active' });
      expect(query).toBe('SELECT * FROM users WHERE status = $1');
      expect(params).toEqual(['active']);
    });

    test('should build SELECT with multiple conditions', () => {
      const { query, params } = buildSelectQuery('users', ['*'], { 
        status: 'active', 
        role: 'admin' 
      });
      expect(query).toBe('SELECT * FROM users WHERE status = $1 AND role = $2');
      expect(params).toEqual(['active', 'admin']);
    });

    test('should handle NULL conditions', () => {
      const { query, params } = buildSelectQuery('users', ['*'], { deleted_at: null });
      expect(query).toBe('SELECT * FROM users WHERE deleted_at IS NULL');
      expect(params).toEqual([]);
    });

    test('should handle IN clause for arrays', () => {
      const { query, params } = buildSelectQuery('users', ['*'], { id: [1, 2, 3] });
      expect(query).toBe('SELECT * FROM users WHERE id IN ($1, $2, $3)');
      expect(params).toEqual([1, 2, 3]);
    });
  });

  describe('INSERT queries', () => {
    test('should build INSERT query', () => {
      const { query, params } = buildInsertQuery('users', { 
        email: 'test@example.com', 
        name: 'Test User' 
      });
      expect(query).toBe('INSERT INTO users (email, name) VALUES ($1, $2)');
      expect(params).toEqual(['test@example.com', 'Test User']);
    });

    test('should handle single column', () => {
      const { query, params } = buildInsertQuery('users', { email: 'test@example.com' });
      expect(query).toBe('INSERT INTO users (email) VALUES ($1)');
      expect(params).toEqual(['test@example.com']);
    });
  });

  describe('UPDATE queries', () => {
    test('should build UPDATE query', () => {
      const { query, params } = buildUpdateQuery('users', { name: 'New Name' }, { id: 1 });
      expect(query).toBe('UPDATE users SET name = $1 WHERE id = $2');
      expect(params).toEqual(['New Name', 1]);
    });

    test('should handle NULL values', () => {
      const { query, params } = buildUpdateQuery('users', { deleted_at: null }, { id: 1 });
      expect(query).toBe('UPDATE users SET deleted_at = NULL WHERE id = $1');
      expect(params).toEqual([1]);
    });

    test('should handle multiple columns', () => {
      const { query, params } = buildUpdateQuery('users', { 
        name: 'New Name', 
        email: 'new@example.com' 
      }, { id: 1 });
      expect(query).toBe('UPDATE users SET name = $1, email = $2 WHERE id = $3');
      expect(params).toEqual(['New Name', 'new@example.com', 1]);
    });
  });

  describe('DELETE queries', () => {
    test('should build DELETE query with condition', () => {
      const { query, params } = buildDeleteQuery('users', { id: 1 });
      expect(query).toBe('DELETE FROM users WHERE id = $1');
      expect(params).toEqual([1]);
    });

    test('should build DELETE query without condition', () => {
      const { query, params } = buildDeleteQuery('users', {});
      expect(query).toBe('DELETE FROM users');
      expect(params).toEqual([]);
    });

    test('should handle multiple conditions', () => {
      const { query, params } = buildDeleteQuery('users', { id: 1, status: 'inactive' });
      expect(query).toBe('DELETE FROM users WHERE id = $1 AND status = $2');
      expect(params).toEqual([1, 'inactive']);
    });
  });
});

describe('Validation Rules', () => {
  const RULES = {
    email: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
    password: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)[a-zA-Z\d]{8,}$/,
    phone: /^\+?[1-9]\d{1,14}$/,
    url: /^https?:\/\/.+/,
    uuid: /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
  };

  const validate = (type, value) => {
    const rule = RULES[type];
    if (!rule) return { valid: false, error: 'Unknown validation type' };
    if (!rule.test(value)) return { valid: false, error: `Invalid ${type}` };
    return { valid: true };
  };

  describe('email validation', () => {
    test('should accept valid email', () => {
      const result = validate('email', 'test@example.com');
      expect(result.valid).toBe(true);
    });

    test('should reject invalid email', () => {
      const result = validate('email', 'invalid-email');
      expect(result.valid).toBe(false);
    });

    test('should reject email without @', () => {
      const result = validate('email', 'testexample.com');
      expect(result.valid).toBe(false);
    });

    test('should reject email without domain', () => {
      const result = validate('email', 'test@');
      expect(result.valid).toBe(false);
    });
  });

  describe('password validation', () => {
    test('should accept valid password', () => {
      const result = validate('password', 'Password123');
      expect(result.valid).toBe(true);
    });

    test('should reject short password', () => {
      const result = validate('password', 'Pass1');
      expect(result.valid).toBe(false);
    });

    test('should reject password without uppercase', () => {
      const result = validate('password', 'password123');
      expect(result.valid).toBe(false);
    });

    test('should reject password without lowercase', () => {
      const result = validate('password', 'PASSWORD123');
      expect(result.valid).toBe(false);
    });

    test('should reject password without number', () => {
      const result = validate('password', 'PasswordABC');
      expect(result.valid).toBe(false);
    });
  });

  describe('phone validation', () => {
    test('should accept valid phone number', () => {
      expect(validate('phone', '+1234567890').valid).toBe(true);
    });

    test('should reject invalid phone', () => {
      expect(validate('phone', 'abc').valid).toBe(false);
    });
  });

  describe('url validation', () => {
    test('should accept http url', () => {
      expect(validate('url', 'http://example.com').valid).toBe(true);
    });

    test('should accept https url', () => {
      expect(validate('url', 'https://example.com').valid).toBe(true);
    });

    test('should reject invalid url', () => {
      expect(validate('url', 'not-a-url').valid).toBe(false);
    });
  });

  describe('uuid validation', () => {
    test('should accept valid uuid', () => {
      expect(validate('uuid', '123e4567-e89b-12d3-a456-426614174000').valid).toBe(true);
    });

    test('should reject invalid uuid', () => {
      expect(validate('uuid', 'not-a-uuid').valid).toBe(false);
    });
  });
});
