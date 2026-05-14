const { Pool } = require('pg');

describe('Database Connection Tests', () => {
  let pool;

  beforeAll(() => {
    pool = new Pool({
      host: process.env.DB_HOST || 'localhost',
      port: process.env.DB_PORT || 5432,
      database: process.env.DB_NAME || 'hjtpx',
      user: process.env.DB_USER || 'postgres',
      password: process.env.DB_PASSWORD || 'postgres',
    });
  });

  afterAll(async () => {
    await pool.end();
  });

  describe('Connection Tests', () => {
    test('should establish database connection', async () => {
      const client = await pool.connect();
      expect(client).toBeDefined();
      client.release();
    });

    test('should execute simple query', async () => {
      const result = await pool.query('SELECT $1 as value', ['test']);
      expect(result.rows[0].value).toBe('test');
    });

    test('should handle connection errors gracefully', async () => {
      const badPool = new Pool({
        host: 'invalid-host',
        port: 5432,
        database: 'nonexistent',
        user: 'invalid',
        password: 'invalid',
        connectionTimeoutMillis: 2000,
      });

      await expect(badPool.query('SELECT 1')).rejects.toThrow();
      await badPool.end();
    });
  });

  describe('Query Tests', () => {
    test('should insert and retrieve data', async () => {
      const testEmail = `test_${Date.now()}@example.com`;
      const insertResult = await pool.query(
        'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4) RETURNING *',
        [testEmail, 'Test User', 'hashedpassword', 'user']
      );
      expect(insertResult.rows[0].email).toBe(testEmail);

      const selectResult = await pool.query(
        'SELECT * FROM users WHERE email = $1',
        [testEmail]
      );
      expect(selectResult.rows[0].email).toBe(testEmail);

      await pool.query('DELETE FROM users WHERE email = $1', [testEmail]);
    });

    test('should update existing record', async () => {
      const testEmail = `test_update_${Date.now()}@example.com`;
      await pool.query(
        'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4)',
        [testEmail, 'Original Name', 'password', 'user']
      );

      const updateResult = await pool.query(
        'UPDATE users SET name = $1 WHERE email = $2 RETURNING *',
        ['Updated Name', testEmail]
      );
      expect(updateResult.rows[0].name).toBe('Updated Name');

      await pool.query('DELETE FROM users WHERE email = $1', [testEmail]);
    });

    test('should handle batch operations', async () => {
      const emails = Array.from({ length: 5 }, (_, i) => `batch_test_${i}_${Date.now()}@example.com`);

      for (const email of emails) {
        await pool.query(
          'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4)',
          [email, 'Batch User', 'password', 'user']
        );
      }

      const result = await pool.query(
        'SELECT COUNT(*) FROM users WHERE email LIKE $1',
        [`batch_test_%@example.com`]
      );
      expect(parseInt(result.rows[0].count)).toBeGreaterThanOrEqual(5);

      await pool.query('DELETE FROM users WHERE email LIKE $1', ['batch_test_%@example.com']);
    });
  });

  describe('Transaction Tests', () => {
    test('should commit transaction on success', async () => {
      const client = await pool.connect();
      const testEmail = `transaction_${Date.now()}@example.com`;

      try {
        await client.query('BEGIN');
        await client.query(
          'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4)',
          [testEmail, 'Transaction User', 'password', 'user']
        );
        await client.query('COMMIT');

        const result = await pool.query('SELECT * FROM users WHERE email = $1', [testEmail]);
        expect(result.rows.length).toBe(1);
      } catch (error) {
        await client.query('ROLLBACK');
        throw error;
      } finally {
        client.release();
        await pool.query('DELETE FROM users WHERE email = $1', [testEmail]);
      }
    });

    test('should rollback transaction on error', async () => {
      const client = await pool.connect();
      const testEmail = `rollback_${Date.now()}@example.com`;

      try {
        await client.query('BEGIN');
        await client.query(
          'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4)',
          [testEmail, 'Rollback User', 'password', 'user']
        );
        await client.query('INSERT INTO nonexistent_table VALUES (1)');
        await client.query('COMMIT');
      } catch (error) {
        await client.query('ROLLBACK');
      } finally {
        client.release();
      }

      const result = await pool.query('SELECT * FROM users WHERE email = $1', [testEmail]);
      expect(result.rows.length).toBe(0);
    });

    test('should handle nested transactions with savepoints', async () => {
      const client = await pool.connect();
      const email1 = `nested1_${Date.now()}@example.com`;
      const email2 = `nested2_${Date.now()}@example.com`;

      try {
        await client.query('BEGIN');
        await client.query(
          'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4)',
          [email1, 'Nested User 1', 'password', 'user']
        );

        await client.query('SAVEPOINT sp1');
        await client.query(
          'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4)',
          [email2, 'Nested User 2', 'password', 'user']
        );
        await client.query('ROLLBACK TO SAVEPOINT sp1');

        await client.query('COMMIT');

        const result = await pool.query('SELECT COUNT(*) FROM users WHERE email LIKE $1', [`nested%@example.com`]);
        expect(parseInt(result.rows[0].count)).toBe(1);
      } catch (error) {
        await client.query('ROLLBACK');
        throw error;
      } finally {
        client.release();
        await pool.query('DELETE FROM users WHERE email LIKE $1', [`nested%@example.com`]);
      }
    });
  });

  describe('Error Handling Tests', () => {
    test('should handle unique constraint violations', async () => {
      const testEmail = `unique_${Date.now()}@example.com`;

      await pool.query(
        'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4)',
        [testEmail, 'User 1', 'password', 'user']
      );

      await expect(
        pool.query(
          'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4)',
          [testEmail, 'User 2', 'password', 'user']
        )
      ).rejects.toThrow();

      await pool.query('DELETE FROM users WHERE email = $1', [testEmail]);
    });

    test('should handle invalid query syntax', async () => {
      await expect(
        pool.query('SELEC * FROM users')
      ).rejects.toThrow();
    });

    test('should handle non-existent table', async () => {
      await expect(
        pool.query('SELECT * FROM nonexistent_table')
      ).rejects.toThrow();
    });

    test('should handle null values correctly', async () => {
      const testEmail = `null_${Date.now()}@example.com`;
      await pool.query(
        'INSERT INTO users (email, name, password, role, last_login) VALUES ($1, $2, $3, $4, NULL)',
        [testEmail, 'Null User', 'password', 'user']
      );

      const result = await pool.query('SELECT * FROM users WHERE email = $1', [testEmail]);
      expect(result.rows[0].last_login).toBeNull();

      await pool.query('DELETE FROM users WHERE email = $1', [testEmail]);
    });
  });

  describe('Performance Tests', () => {
    test('should handle concurrent queries', async () => {
      const queries = Array.from({ length: 10 }, (_, i) =>
        pool.query(`SELECT ${i} as num`)
      );

      const results = await Promise.all(queries);
      expect(results.length).toBe(10);
      results.forEach((result, i) => {
        expect(result.rows[0].num).toBe(i);
      });
    });

    test('should handle large result sets', async () => {
      const result = await pool.query('SELECT * FROM users LIMIT 1000');
      expect(result.rows).toBeDefined();
      expect(Array.isArray(result.rows)).toBe(true);
    });

    test('should measure query execution time', async () => {
      const start = Date.now();
      await pool.query('SELECT * FROM users');
      const duration = Date.now() - start;
      expect(duration).toBeLessThan(5000);
    });
  });
});
