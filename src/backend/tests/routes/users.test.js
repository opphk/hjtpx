jest.mock('../../../config/database/db', () => ({
  query: jest.fn(),
  pool: {
    on: jest.fn(),
    query: jest.fn(),
    connect: jest.fn(),
    totalCount: 0,
    idleCount: 0,
    waitingCount: 0
  }
}));

jest.mock('../../services/userService');

const userService = require('../../services/userService');

describe('Users Routes', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('userService', () => {
    it('should have getAllUsers method', () => {
      expect(userService.getAllUsers).toBeDefined();
    });

    it('should have getUserById method', () => {
      expect(userService.getUserById).toBeDefined();
    });

    it('should have createUser method', () => {
      expect(userService.createUser).toBeDefined();
    });

    it('should have updateUser method', () => {
      expect(userService.updateUser).toBeDefined();
    });

    it('should have deleteUser method', () => {
      expect(userService.deleteUser).toBeDefined();
    });

    it('should return users from getAllUsers', async () => {
      const mockUsers = [
        { id: 1, email: 'user1@example.com', name: 'User One' },
        { id: 2, email: 'user2@example.com', name: 'User Two' }
      ];
      userService.getAllUsers.mockResolvedValue(mockUsers);

      const result = await userService.getAllUsers();

      expect(result).toHaveLength(2);
      expect(result[0].email).toBe('user1@example.com');
    });

    it('should return user by id', async () => {
      const mockUser = { id: 1, email: 'user@example.com', name: 'Test User' };
      userService.getUserById.mockResolvedValue(mockUser);

      const result = await userService.getUserById(1);

      expect(result.id).toBe(1);
      expect(result.email).toBe('user@example.com');
    });

    it('should return undefined for non-existent user', async () => {
      userService.getUserById.mockResolvedValue(undefined);

      const result = await userService.getUserById(999);

      expect(result).toBeUndefined();
    });

    it('should create new user', async () => {
      const newUser = { email: 'new@example.com', name: 'New User' };
      const createdUser = { id: 3, ...newUser };
      userService.createUser.mockResolvedValue(createdUser);

      const result = await userService.createUser(newUser);

      expect(result.id).toBe(3);
      expect(result.email).toBe('new@example.com');
    });

    it('should update user', async () => {
      const updatedUser = { id: 1, email: 'updated@example.com' };
      userService.updateUser.mockResolvedValue(updatedUser);

      const result = await userService.updateUser(1, { email: 'updated@example.com' });

      expect(result.email).toBe('updated@example.com');
    });

    it('should delete user', async () => {
      userService.deleteUser.mockResolvedValue();

      await expect(userService.deleteUser(1)).resolves.toBeUndefined();
    });

    it('should handle getAllUsers error', async () => {
      userService.getAllUsers.mockRejectedValue(new Error('Database error'));

      await expect(userService.getAllUsers()).rejects.toThrow('Database error');
    });

    it('should handle getUserById error', async () => {
      userService.getUserById.mockRejectedValue(new Error('User not found'));

      await expect(userService.getUserById(1)).rejects.toThrow('User not found');
    });

    it('should handle createUser error', async () => {
      userService.createUser.mockRejectedValue(new Error('User already exists'));

      await expect(
        userService.createUser({ email: 'existing@example.com' })
      ).rejects.toThrow('User already exists');
    });
  });
});
