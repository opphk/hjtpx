require('dotenv').config();

jest.setTimeout(30000);

const localStorageMock = (() => {
  let store = {};
  return {
    getItem: jest.fn((key) => store[key] || null),
    setItem: jest.fn((key, value) => {
      store[key] = String(value);
    }),
    removeItem: jest.fn((key) => {
      delete store[key];
    }),
    clear: jest.fn(() => {
      store = {};
    }),
    get length() {
      return Object.keys(store).length;
    },
    key: jest.fn((index) => Object.keys(store)[index] || null),
  };
})();

global.localStorage = localStorageMock;
global.window = {
  location: {
    href: '',
    pathname: '/',
    reload: jest.fn(),
  },
  history: {
    pushState: jest.fn(),
    replaceState: jest.fn(),
  },
};

if (typeof TextEncoder === 'undefined') {
  global.TextEncoder = class TextEncoder {
    encode(str) {
      const buf = Buffer.from(str);
      return new Uint8Array(buf);
    }
  };
}

if (typeof TextDecoder === 'undefined') {
  global.TextDecoder = class TextDecoder {
    decode(arr) {
      if (arr instanceof Uint8Array) {
        return Buffer.from(arr).toString();
      }
      return String(arr);
    }
  };
}

global.fetch = jest.fn(() =>
  Promise.resolve({
    ok: true,
    json: () => Promise.resolve({ success: true }),
    text: () => Promise.resolve(''),
  })
);

jest.mock('pg', () => {
  return {
    Pool: jest.fn().mockImplementation(() => ({
      query: jest.fn(),
      connect: jest.fn(),
      end: jest.fn(),
    })),
  };
});

jest.mock('mongoose', () => {
  return {
    connect: jest.fn().mockResolvedValue(true),
    Schema: jest.fn(() => ({
      Types: {
        ObjectId: jest.fn()
      }
    })),
    model: jest.fn().mockReturnValue({
      find: jest.fn(),
      findById: jest.fn(),
      create: jest.fn(),
      findByIdAndUpdate: jest.fn(),
      findByIdAndDelete: jest.fn(),
    }),
    Types: {
      ObjectId: jest.fn().mockReturnValue('mock-object-id')
    }
  };
});

jest.mock('redis', () => ({
  createClient: jest.fn().mockReturnValue({
    on: jest.fn(),
    connect: jest.fn().mockResolvedValue(true),
    get: jest.fn(),
    set: jest.fn(),
    del: jest.fn(),
    expire: jest.fn(),
    disconnect: jest.fn(),
  }),
}));

jest.mock('ioredis', () => jest.fn().mockImplementation(() => ({
  on: jest.fn(),
  connect: jest.fn().mockResolvedValue(true),
  get: jest.fn(),
  set: jest.fn(),
  del: jest.fn(),
  expire: jest.fn(),
  disconnect: jest.fn(),
})));

jest.mock('apollo-server-express', () => ({
  ApolloServer: jest.fn().mockImplementation(() => ({
    start: jest.fn().mockResolvedValue(true),
    applyMiddleware: jest.fn(),
  })),
}));

jest.mock('@sentry/node', () => ({
  init: jest.fn(),
  Handlers: {
    requestHandler: jest.fn(),
    tracingHandler: jest.fn(),
    errorHandler: jest.fn(),
  },
}));

jest.mock('@sentry/tracing', () => ({}));

beforeAll(() => {
  if (!process.env.JWT_SECRET) {
    process.env.JWT_SECRET = 'test-secret-key-for-testing';
  }
  if (!process.env.NODE_ENV) {
    process.env.NODE_ENV = 'test';
  }
});

afterAll(() => {
  jest.clearAllMocks();
});
