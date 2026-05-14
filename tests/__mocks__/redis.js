const mockGet = jest.fn();
const mockSet = jest.fn();
const mockDel = jest.fn();
const mockExists = jest.fn();
const mockKeys = jest.fn();

const client = {
  get: mockGet,
  set: mockSet,
  del: mockDel,
  exists: mockExists,
  keys: mockKeys
};

module.exports = {
  client,
  get: mockGet,
  set: mockSet,
  del: mockDel,
  exists: mockExists,
  keys: mockKeys
};
