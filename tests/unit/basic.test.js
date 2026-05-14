describe('Basic Tests', () => {
  test('should pass basic test', () => {
    expect(1 + 1).toBe(2);
  });

  test('should handle API health check', async () => {
    const response = await fetch('http://localhost:3000/api/health');
    const data = await response.json();
    expect(data.status).toBe('healthy');
  });
});
