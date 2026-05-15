const bcrypt = require('bcryptjs');
const { GraphQLScalarType, Kind } = require('graphql');

const Notification = require('../../models/Notification');
const authService = require('../../services/authService');
const userService = require('../../services/userService');
const { requireAuth, requireAdmin, requireOwnerOrAdmin } = require('../middleware/auth');

const JSONScalar = new GraphQLScalarType({
  name: 'JSON',
  description: 'JSON scalar type',
  serialize(value) {
    return value;
  },
  parseValue(value) {
    return value;
  },
  parseLiteral(ast) {
    if (ast.kind === Kind.STRING) {
      try {
        return JSON.parse(ast.value);
      } catch {
        return ast.value;
      }
    }
    if (ast.kind === Kind.OBJECT) {
      const obj = {};
      ast.fields.forEach(field => {
        obj[field.name.value] = parseLiteral(field.value);
      });
      return obj;
    }
    if (ast.kind === Kind.LIST) {
      return ast.values.map(v => parseLiteral(v));
    }
    if (ast.kind === Kind.INT) {
      return parseInt(ast.value, 10);
    }
    if (ast.kind === Kind.FLOAT) {
      return parseFloat(ast.value);
    }
    if (ast.kind === Kind.BOOLEAN) {
      return ast.value;
    }
    if (ast.kind === Kind.NULL) {
      return null;
    }
    return null;
  }
});

const parseLiteral = ast => {
  if (ast.kind === Kind.STRING) return ast.value;
  if (ast.kind === Kind.INT) return parseInt(ast.value, 10);
  if (ast.kind === Kind.FLOAT) return parseFloat(ast.value);
  if (ast.kind === Kind.BOOLEAN) return ast.value;
  if (ast.kind === Kind.NULL) return null;
  if (ast.kind === Kind.OBJECT) {
    const obj = {};
    ast.fields.forEach(field => {
      obj[field.name.value] = parseLiteral(field.value);
    });
    return obj;
  }
  if (ast.kind === Kind.LIST) {
    return ast.values.map(v => parseLiteral(v));
  }
  return null;
};

const formatUser = user => {
  if (!user) return null;
  return {
    id: user.id?.toString() || user._id?.toString(),
    email: user.email,
    name: user.name,
    role: user.role,
    created_at: user.created_at?.toISOString?.() || user.created_at,
    updated_at: user.updated_at?.toISOString?.() || user.updated_at
  };
};

const formatNotification = notification => {
  if (!notification) return null;
  return {
    id: notification.id?.toString() || notification._id?.toString(),
    userId: notification.userId?.toString(),
    type: notification.type,
    title: notification.title,
    message: notification.message,
    data: notification.data,
    priority: notification.priority,
    status: notification.status,
    readAt: notification.readAt?.toISOString?.() || notification.readAt,
    expiresAt: notification.expiresAt?.toISOString?.() || notification.expiresAt,
    actionUrl: notification.actionUrl,
    actionLabel: notification.actionLabel,
    channels: notification.channels || ['in_app'],
    metadata: notification.metadata,
    createdAt: notification.createdAt?.toISOString?.() || notification.createdAt,
    updatedAt: notification.updatedAt?.toISOString?.() || notification.updatedAt
  };
};

const userResolver = {
  notifications: async (user, args, context) => {
    requireAuth(context);
    const result = await Notification.getUserNotifications(user.id, {
      status: args.status,
      limit: args.limit || 10
    });
    return result.notifications.map(formatNotification);
  },

  unreadNotificationsCount: async (user, args, context) => {
    requireAuth(context);
    return await Notification.getUnreadCount(user.id);
  }
};

const notificationResolver = {
  user: async (notification, args, context) => {
    requireAuth(context);
    const user = await userService.getUserById(notification.userId);
    return formatUser(user);
  }
};

const queryResolver = {
  users: async (_, args, context) => {
    requireAdmin(context);
    const users = await userService.getAllUsers({
      limit: args.limit || 100,
      offset: args.offset || 0
    });
    return users.map(formatUser);
  },

  user: async (_, { id }, context) => {
    requireAuth(context);
    const user = await userService.getUserById(id);
    return formatUser(user);
  },

  me: async (_, __, context) => {
    const user = requireAuth(context);
    const fullUser = await userService.getUserById(user.id);
    return formatUser(fullUser);
  },

  notifications: async (_, args, context) => {
    const user = requireAuth(context);
    const result = await Notification.getUserNotifications(user.id, args);
    return {
      notifications: result.notifications.map(formatNotification),
      pagination: result.pagination
    };
  },

  notification: async (_, { id }, context) => {
    const user = requireAuth(context);
    const notification = await Notification.findOne({
      _id: id,
      userId: user.id
    });
    return formatNotification(notification);
  },

  unreadNotificationsCount: async (_, __, context) => {
    const user = requireAuth(context);
    return await Notification.getUnreadCount(user.id);
  }
};

const mutationResolver = {
  createUser: async (_, args, context) => {
    requireAdmin(context);
    const user = await userService.createUser(args);
    return formatUser(user);
  },

  updateUser: async (_, { id, ...updates }, context) => {
    requireOwnerOrAdmin(context, id);
    if (context.user.role !== 'admin') {
      delete updates.role;
    }
    const user = await userService.updateUser(id, updates);
    return formatUser(user);
  },

  deleteUser: async (_, { id }, context) => {
    requireAdmin(context);
    await userService.deleteUser(id);
    return true;
  },

  createNotification: async (_, args, context) => {
    const user = requireAuth(context);
    const notificationData = {
      ...args,
      userId: args.userId || user.id
    };
    const notification = await Notification.createNotification(notificationData);
    return formatNotification(notification);
  },

  markNotificationAsRead: async (_, { id }, context) => {
    const user = requireAuth(context);
    const result = await Notification.markAsRead(id, user.id);
    if (result.modifiedCount === 0) {
      return null;
    }
    const notification = await Notification.findById(id);
    return formatNotification(notification);
  },

  markAllNotificationsAsRead: async (_, __, context) => {
    const user = requireAuth(context);
    await Notification.markAllAsRead(user.id);
    return true;
  },

  deleteNotification: async (_, { id }, context) => {
    const user = requireAuth(context);
    const result = await Notification.deleteOne({ _id: id, userId: user.id });
    return result.deletedCount > 0;
  },

  login: async (_, { email, password }) => {
    const user = await authService.authenticate(email, password);
    if (!user) {
      throw new Error('Invalid credentials');
    }
    const token = authService.generateToken({
      id: user.id,
      email: user.email,
      role: user.role
    });
    return {
      token,
      user: formatUser(user)
    };
  },

  register: async (_, { email, name, password }) => {
    const existingUser = await userService.getUserByEmail(email);
    if (existingUser) {
      throw new Error('User already exists');
    }

    authService.validatePassword(password);

    const hashedPassword = await bcrypt.hash(password, 10);
    const user = await userService.createUser({
      email,
      name,
      password: hashedPassword,
      role: 'user'
    });

    const token = authService.generateToken({
      id: user.id,
      email: user.email,
      role: user.role
    });

    return {
      token,
      user: formatUser(user)
    };
  }
};

const resolvers = {
  JSON: JSONScalar,

  User: userResolver,
  Notification: notificationResolver,

  Query: queryResolver,
  Mutation: mutationResolver
};

module.exports = resolvers;
