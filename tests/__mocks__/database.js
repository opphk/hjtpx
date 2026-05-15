const mockQuery = jest.fn();
const mockGetClient = jest.fn();
const mockTransaction = jest.fn();
const mockHealthCheck = jest.fn();
const mockGetPoolStats = jest.fn();
const mockClose = jest.fn();

const mockClient = {
  query: mockQuery,
  release: jest.fn()
};

mockGetClient.mockResolvedValue(mockClient);
mockQuery.mockResolvedValue({ rows: [], rowCount: 0 });
mockHealthCheck.mockResolvedValue({
  status: 'healthy',
  timestamp: new Date(),
  responseTime: 10,
  dbSize: 1000000
});
mockGetPoolStats.mockResolvedValue({
  totalCount: 0,
  idleCount: 0,
  waitingCount: 0,
  checkedOutCount: 0,
  queryStats: {
    totalQueries: 0,
    slowQueries: 0,
    errors: 0,
    avgQueryTime: 0,
    hitRate: '100%',
    connectionLeaks: 0
  }
});
mockClose.mockResolvedValue(undefined);

const pool = {
  query: mockQuery,
  connect: mockGetClient,
  totalCount: 0,
  idleCount: 0,
  waitingCount: 0,
  end: mockClose,
  on: jest.fn()
};

module.exports = {
  query: mockQuery,
  getClient: mockGetClient,
  transaction: mockTransaction,
  healthCheck: mockHealthCheck,
  getPoolStats: mockGetPoolStats,
  close: mockClose,
  pool,
  events: {
    on: jest.fn(),
    emit: jest.fn()
  }
};
