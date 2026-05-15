const mockQuery = jest.fn();
const mockOn = jest.fn();
const mockConnect = jest.fn();
const mockEnd = jest.fn().mockResolvedValue(undefined);
const mockGetClient = jest.fn();
const mockTransaction = jest.fn();
const mockHealthCheck = jest.fn();
const mockGetPoolStats = jest.fn();
const mockClose = jest.fn();

const pool = {
  query: mockQuery,
  on: mockOn,
  connect: mockConnect,
  end: mockEnd,
  totalCount: 0,
  idleCount: 0,
  waitingCount: 0
};

module.exports = {
  query: mockQuery,
  getClient: mockGetClient,
  transaction: mockTransaction,
  healthCheck: mockHealthCheck,
  getPoolStats: mockGetPoolStats,
  close: mockClose,
  end: mockEnd,
  pool: pool
};
