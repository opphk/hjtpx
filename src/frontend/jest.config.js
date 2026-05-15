module.exports = {
  testEnvironment: 'jsdom',
  setupFilesAfterEnv: ['./test-setup.js'],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '\\.(css|less|scss|sass)$': 'identity-obj-proxy'
  },
  transform: {
    '^.+\\.(js|jsx|ts|tsx)$': 'babel-jest'
  },
  testMatch: [
    '**/__tests__/**/*.[jt]s?(x)',
    '**/?(*.)+(spec|test).[jt]s?(x)'
  ],
  collectCoverageFrom: [
    'src/**/*.{js,jsx}',
    '!src/**/*.test.{js,jsx}',
    '!src/**/__tests__/**',
    '!src/**/*.stories.{js,jsx}'
  ],
  coverageThreshold: {
    global: {
      branches: 70,
      functions: 80,
      lines: 75,
      statements: 75
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
    develop: {
      branches: 75,
      functions: 85,
      lines: 80,
      statements: 80
    }
  },
  verbose: true
};
