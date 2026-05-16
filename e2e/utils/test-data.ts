export const testUsers = {
  admin: {
    username: 'admin',
    password: 'admin123',
    email: 'admin@example.com'
  },
  test: {
    username: 'testuser',
    password: 'TestPass123!',
    email: 'testuser@example.com'
  }
};

export const testApplications = [
  {
    name: 'Test App 1',
    description: 'Test application 1'
  },
  {
    name: 'Test App 2',
    description: 'Test application 2'
  }
];

export const captchaTypes = ['slider', 'click', 'rotate', 'gesture'] as const;

export function generateRandomString(length: number = 8): string {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

export function generateRandomEmail(): string {
  return `${generateRandomString(8)}@example.com`;
}
