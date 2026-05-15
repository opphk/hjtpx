const mockCreateApolloServer = jest.fn().mockResolvedValue({
  start: jest.fn().mockResolvedValue(true),
  applyMiddleware: jest.fn().mockReturnValue(true)
});

module.exports = {
  createApolloServer: mockCreateApolloServer
};
