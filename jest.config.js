module.exports = {
  testEnvironment: 'node',
  roots: ['<rootDir>/src', '<rootDir>/captchax', '<rootDir>/tests'],
  testMatch: ['**/tests/**/*.test.js', '**/__tests__/**/*.test.js'],
  collectCoverageFrom: [
    'src/**/*.js',
    'src/**/*.jsx',
    'captchax/**/*.js',
    '!src/**/*.test.js',
    '!captchax/**/*.test.js',
    'src/**/__tests__/**',
    '!src/**/node_modules/**',
    '!captchax/**/node_modules/**'
  ],
  transform: {
    '^.+\\.(js|jsx)$': 'babel-jest'
  },
  transformIgnorePatterns: [
    '/node_modules/(?!(uuid|mongoose|mongodb|bson|@aws|exceljs|pdfkit|csv-parse|token-types|strtok3|file-type|@borewit)/)'
  ],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '\\.(css|less|scss|sass)$': '<rootDir>/src/backend/tests/__mocks__/styleMock.js',
    '^config/database/db$': '<rootDir>/src/backend/tests/__mocks__/database.js',
    '^src/config/database/db$': '<rootDir>/src/backend/tests/__mocks__/database.js',
    '^backend/config/database/db$': '<rootDir>/src/backend/tests/__mocks__/database.js',
    '^src/backend/config/database/db$': '<rootDir>/src/backend/tests/__mocks__/database.js',
    '^../../config/database/db$': '<rootDir>/src/backend/tests/__mocks__/database.js',
    '^../config/database/db$': '<rootDir>/src/backend/tests/__mocks__/database.js',
    '^../../../config/database/db$': '<rootDir>/src/backend/tests/__mocks__/database.js',
    '^config/redis/client$': '<rootDir>/src/backend/tests/__mocks__/redisClient.js',
    '^src/config/redis/client$': '<rootDir>/src/backend/tests/__mocks__/redisClient.js',
    '^backend/config/redis/client$': '<rootDir>/src/backend/tests/__mocks__/redisClient.js',
    '^../../../config/redis/client$': '<rootDir>/src/backend/tests/__mocks__/redisClient.js',
    '^../../config/redis/client$': '<rootDir>/src/backend/tests/__mocks__/redisClient.js',
    '^../config/redis/client$': '<rootDir>/src/backend/tests/__mocks__/redisClient.js',
    '^../src/backend/middleware/versionControl$': '<rootDir>/src/backend/middleware/versionControl.js',
    '^../src/backend/middleware/apiVersionNegotiation$': '<rootDir>/src/backend/middleware/apiVersionNegotiation.js',
    '^../src/backend/middleware/deprecationWarning$': '<rootDir>/src/backend/middleware/deprecationWarning.js'
  },
  setupFilesAfterEnv: ['<rootDir>/src/backend/tests/setup.js'],
  clearMocks: true,
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'lcov', 'clover', 'json-summary', 'json'],
  coverageThreshold: {
    global: {
      branches: 75,
      functions: 85,
      lines: 80,
      statements: 80
    },
    './src/utils/': {
      branches: 75,
      functions: 85,
      lines: 80,
      statements: 80
    },
    './src/models/': {
      branches: 75,
      functions: 85,
      lines: 80,
      statements: 80
    },
    './src/services/': {
      branches: 75,
      functions: 85,
      lines: 80,
      statements: 80
    }
  },
  coverageAlertThreshold: {
    global: {
      branches: 70,
      functions: 80,
      lines: 75,
      statements: 75
    }
  },
  coverageBranchRequirements: {
    main: {
      branches: 75,
      functions: 85,
      lines: 80,
      statements: 80
    },
    develop: {
      branches: 75,
      functions: 85,
      lines: 80,
      statements: 80
    },
    feature: {
      branches: 75,
      functions: 85,
      lines: 80,
      statements: 80
    },
    hotfix: {
      branches: 75,
      functions: 85,
      lines: 80,
      statements: 80
    }
  },
  testPathIgnorePatterns: ['/node_modules/', '/frontend/node_modules/'],
  verbose: true,
  modulePathIgnorePatterns: ['<rootDir>/src/config/database/db.js'],
  testTimeout: 30000,
  forceExit: true,
  detectOpenHandles: true
};
