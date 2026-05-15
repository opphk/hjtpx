jest.mock('../../../config/database/db', () => ({
  query: jest.fn(),
  getClient: jest.fn()
}));

const db = require('../../config/database/db');
const queryOptimizer = require('../../utils/queryOptimizer');

describe('queryOptimizer', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    queryOptimizer.clearCache();
  });

  describe('cacheKey', () => {
    test('should generate consistent cache keys', () => {
      const key1 = queryOptimizer.cacheKey('SELECT * FROM users', [1, 'test']);
      const key2 = queryOptimizer.cacheKey('SELECT * FROM users', [1, 'test']);
      const key3 = queryOptimizer.cacheKey('SELECT * FROM users', [2, 'test']);

      expect(key1).toBe(key2);
      expect(key1).not.toBe(key3);
    });

    test('should handle null params', () => {
      const key = queryOptimizer.cacheKey('SELECT * FROM users', null);
      expect(key).toContain('SELECT * FROM users');
    });
  });

  describe('cachedQuery', () => {
    test('should return cached result if within TTL', async () => {
      const mockData = [{ id: 1, name: 'User 1' }];
      db.query.mockResolvedValueOnce({ rows: mockData });

      const result1 = await queryOptimizer.cachedQuery('SELECT * FROM users', [], 60);
      const result2 = await queryOptimizer.cachedQuery('SELECT * FROM users', [], 60);

      expect(result1).toEqual(mockData);
      expect(result2).toEqual(mockData);
      expect(db.query).toHaveBeenCalledTimes(1);
    });

    test('should fetch from database if cache expired', async () => {
      const mockData1 = [{ id: 1, name: 'User 1' }];
      const mockData2 = [{ id: 2, name: 'User 2' }];

      db.query
        .mockResolvedValueOnce({ rows: mockData1 })
        .mockResolvedValueOnce({ rows: mockData2 });

      const result1 = await queryOptimizer.cachedQuery('SELECT * FROM users', [], 0);
      await new Promise(resolve => setTimeout(resolve, 10));
      const result2 = await queryOptimizer.cachedQuery('SELECT * FROM users', [], 0);

      expect(result1).toEqual(mockData1);
      expect(result2).toEqual(mockData2);
      expect(db.query).toHaveBeenCalledTimes(2);
    });

    test('should log warning for slow queries', async () => {
      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation();

      const originalThreshold = process.env.SLOW_QUERY_THRESHOLD;
      process.env.SLOW_QUERY_THRESHOLD = '0';

      const optimizer = Object.create(queryOptimizer);
      optimizer.slowQueryThreshold = 0;
      optimizer.queryCache = new Map();

      db.query.mockResolvedValueOnce({ rows: [{ id: 1 }] });

      await optimizer.cachedQuery('SELECT * FROM slow_query', [], 60);

      consoleSpy.mockRestore();
      process.env.SLOW_QUERY_THRESHOLD = originalThreshold;
    });
  });

  describe('batchQuery', () => {
    test('should execute multiple queries in a transaction', async () => {
      const mockClient = {
        query: jest.fn()
          .mockResolvedValueOnce()
          .mockResolvedValueOnce({ rows: [{ id: 1 }] })
          .mockResolvedValueOnce({ rows: [{ id: 2 }] })
          .mockResolvedValueOnce(),
        release: jest.fn()
      };

      db.getClient.mockResolvedValueOnce(mockClient);

      const queries = [
        { query: 'INSERT INTO users VALUES ($1)', params: ['user1'] },
        { query: 'INSERT INTO users VALUES ($1)', params: ['user2'] }
      ];

      const results = await queryOptimizer.batchQuery(queries);

      expect(mockClient.query).toHaveBeenCalledWith('BEGIN');
      expect(mockClient.query).toHaveBeenCalledWith('COMMIT');
      expect(mockClient.release).toHaveBeenCalled();
    });

    test('should rollback on error', async () => {
      const mockClient = {
        query: jest.fn()
          .mockResolvedValueOnce()
          .mockRejectedValueOnce(new Error('DB Error'))
          .mockResolvedValueOnce(),
        release: jest.fn()
      };

      db.getClient.mockResolvedValueOnce(mockClient);

      const queries = [
        { query: 'INSERT INTO users VALUES ($1)', params: ['user1'] }
      ];

      await expect(queryOptimizer.batchQuery(queries)).rejects.toThrow('DB Error');
      expect(mockClient.query).toHaveBeenCalledWith('ROLLBACK');
      expect(mockClient.release).toHaveBeenCalled();
    });
  });

  describe('batchInsert', () => {
    test('should insert rows in batches', async () => {
      const rows = [
        { name: 'User 1', email: 'user1@example.com' },
        { name: 'User 2', email: 'user2@example.com' }
      ];

      db.query.mockResolvedValue({ rows: [{ id: 1, ...rows[0] }, { id: 2, ...rows[1] }] });

      const result = await queryOptimizer.batchInsert('users', rows, 100);

      expect(result).toHaveLength(2);
      expect(db.query).toHaveBeenCalledTimes(1);
    });

    test('should handle empty rows array', async () => {
      const result = await queryOptimizer.batchInsert('users', [], 100);
      expect(result).toEqual([]);
      expect(db.query).not.toHaveBeenCalled();
    });

    test('should split large batches', async () => {
      const rows = [
        { name: 'User 1', email: 'user1@example.com' },
        { name: 'User 2', email: 'user2@example.com' },
        { name: 'User 3', email: 'user3@example.com' }
      ];

      db.query.mockResolvedValue({ rows: [] });

      const result = await queryOptimizer.batchInsert('users', rows, 2);

      expect(result).toHaveLength(0);
      expect(db.query).toHaveBeenCalledTimes(2);
    });
  });

  describe('batchUpdate', () => {
    test('should update rows in batches', async () => {
      const updates = [
        { id: 1, name: 'Updated User 1' },
        { id: 2, name: 'Updated User 2' }
      ];

      const mockClient = {
        query: jest.fn()
          .mockResolvedValueOnce()
          .mockResolvedValueOnce({ rows: [{ id: 1 }] })
          .mockResolvedValueOnce({ rows: [{ id: 2 }] })
          .mockResolvedValueOnce(),
        release: jest.fn()
      };

      db.getClient.mockResolvedValueOnce(mockClient);

      const result = await queryOptimizer.batchUpdate('users', updates);

      expect(result).toHaveLength(2);
      expect(mockClient.query).toHaveBeenCalledWith('BEGIN');
      expect(mockClient.query).toHaveBeenCalledWith('COMMIT');
    });

    test('should handle empty updates array', async () => {
      const result = await queryOptimizer.batchUpdate('users', []);
      expect(result).toEqual([]);
    });
  });

  describe('paginatedQuery', () => {
    test('should return paginated results', async () => {
      const mockData = [
        { id: 1, name: 'User 1' },
        { id: 2, name: 'User 2' }
      ];

      db.query
        .mockResolvedValueOnce({ rows: [{ total: 10 }] })
        .mockResolvedValueOnce({ rows: mockData });

      const result = await queryOptimizer.paginatedQuery('SELECT * FROM users', [], 1, 2);

      expect(result.data).toEqual(mockData);
      expect(result.pagination).toEqual({
        page: 1,
        pageSize: 2,
        total: 10,
        totalPages: 5
      });
    });

    test('should handle empty results', async () => {
      db.query
        .mockResolvedValueOnce({ rows: [{ total: 0 }] })
        .mockResolvedValueOnce({ rows: [] });

      const result = await queryOptimizer.paginatedQuery('SELECT * FROM users', [], 1, 10);

      expect(result.pagination.total).toBe(0);
      expect(result.pagination.totalPages).toBe(0);
    });
  });

  describe('explainQuery', () => {
    test('should return query execution plan', async () => {
      const mockPlan = [{ 'QUERY PLAN': ['Seq Scan on users'] }];
      db.query.mockResolvedValueOnce({ rows: mockPlan });

      const result = await queryOptimizer.explainQuery('SELECT * FROM users', []);

      expect(result).toEqual(mockPlan);
      expect(db.query).toHaveBeenCalledWith('EXPLAIN ANALYZE SELECT * FROM users', []);
    });
  });

  describe('clearCache', () => {
    test('should clear the query cache', async () => {
      db.query.mockResolvedValue({ rows: [{ id: 1 }] });

      await queryOptimizer.cachedQuery('SELECT * FROM users', [], 60);
      expect(db.query).toHaveBeenCalledTimes(1);

      queryOptimizer.clearCache();

      await queryOptimizer.cachedQuery('SELECT * FROM users', [], 60);
      expect(db.query).toHaveBeenCalledTimes(2);
    });
  });

  describe('getCacheStats', () => {
    test('should return cache statistics', async () => {
      db.query.mockResolvedValue({ rows: [{ id: 1 }] });

      await queryOptimizer.cachedQuery('SELECT * FROM users', [], 60);
      const stats = queryOptimizer.getCacheStats();

      expect(stats.size).toBe(1);
      expect(stats.maxSize).toBe(100);
      expect(stats.queries).toHaveLength(1);
    });

    test('should respect max cache size', async () => {
      const optimizer = Object.create(queryOptimizer);
      optimizer.queryCache = new Map();
      optimizer.maxCacheSize = 2;

      db.query.mockResolvedValue({ rows: [] });

      await optimizer.cachedQuery('SELECT 1', [], 60);
      await optimizer.cachedQuery('SELECT 2', [], 60);
      await optimizer.cachedQuery('SELECT 3', [], 60);

      expect(optimizer.queryCache.size).toBe(2);
    });
  });
});
