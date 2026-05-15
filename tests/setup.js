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
