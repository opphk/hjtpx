module.exports = {
  RetryStrategy: require('./retryStrategy').RetryStrategy,
  RetryManager: require('./retryStrategy').RetryManager,
  retryManager: require('./retryStrategy').retryManager,
  deadLetterQueue: require('./deadLetterQueue')
};
