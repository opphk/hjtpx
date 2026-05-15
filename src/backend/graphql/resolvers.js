const { GraphQLScalarType, Kind } = require('graphql');

const userService = require('../services/userService');
const Notification = require('../models/Notification');

const parseLiteralValue = (ast) => {
  switch (ast.kind) {
    case Kind.STRING:
      return ast.value;
    case Kind.INT:
      return parseInt(ast.value, 10);
    case Kind.FLOAT:
      return parseFloat(ast.value);
    case Kind.BOOLEAN:
      return ast.value;
    case Kind.NULL:
      return null;
    case Kind.OBJECT: {
      const obj = {};
      ast.fields.forEach(field => {
        obj[field.name.value] = parseLiteralValue(field.value);
      });
      return obj;
    }
    case Kind.LIST:
      return ast.values.map(parseLiteralValue);
    default:
      return null;
  }
};

const JSONScalar = new GraphQLScalarType({
  name: 'JSON',
  description: 'The JSON scalar type represents JSON values as specified by ECMA-404',
  serialize(value) {
    return value;
  },
  parseValue(value) {
    return value;
  },
  parseLiteral(ast) {
    return parseLiteralValue(ast);
  }
});

const resolvers = {
  JSON: JSONScalar,

  Query: {
    users: async () => {
      return await userService.getAllUsers();
    },

    user: async (_, { id }) => {
      return await userService.getUserById(id);
    },

    me: async (_, __, context) => {
      if (!context.user) {
        throw new Error('Not authenticated');
      }
      return await userService.getUserById(context.user.id);
    },

    notifications: async (_, args, context) => {
      if (!context.user) {
        throw new Error('Not authenticated');
      }
      const result = await Notification.getUserNotifications(context.user.id, args);
      return {
        notifications: result.notifications.map(n => ({
          ...n.toObject(),
          id: n._id.toString()
        })),
        pagination: result.pagination
      };
    },

    notification: async (_, { id }, context) => {
      if (!context.user) {
        throw new Error('Not authenticated');
      }
      const notification = await Notification.findOne({ _id: id, userId: context.user.id });
      if (!notification) {
        return null;
      }
      return {
        ...notification.toObject(),
        id: notification._id.toString()
      };
    },

    unreadNotificationsCount: async (_, __, context) => {
      if (!context.user) {
        throw new Error('Not authenticated');
      }
      return await Notification.getUnreadCount(context.user.id);
    }
  },

  Mutation: {
    createUser: async (_, args, context) => {
      if (!context.user || context.user.role !== 'admin') {
        throw new Error('Not authorized');
      }
      return await userService.createUser(args);
    },

    updateUser: async (_, { id, ...args }, context) => {
      if (!context.user) {
        throw new Error('Not authenticated');
      }
      if (context.user.role !== 'admin' && context.user.id !== id) {
        throw new Error('Not authorized');
      }
      if (context.user.role !== 'admin') {
        delete args.role;
      }
      return await userService.updateUser(id, args);
    },

    deleteUser: async (_, { id }, context) => {
      if (!context.user || context.user.role !== 'admin') {
        throw new Error('Not authorized');
      }
      await userService.deleteUser(id);
      return true;
    },

    createNotification: async (_, args, context) => {
      if (!context.user) {
        throw new Error('Not authenticated');
      }
      const notification = await Notification.createNotification({
        ...args,
        userId: args.userId || context.user.id
      });
      return {
        ...notification.toObject(),
        id: notification._id.toString()
      };
    },

    markNotificationAsRead: async (_, { id }, context) => {
      if (!context.user) {
        throw new Error('Not authenticated');
      }
      const result = await Notification.markAsRead(id, context.user.id);
      if (result.modifiedCount === 0) {
        return null;
      }
      const notification = await Notification.findById(id);
      return {
        ...notification.toObject(),
        id: notification._id.toString()
      };
    },

    markAllNotificationsAsRead: async (_, __, context) => {
      if (!context.user) {
        throw new Error('Not authenticated');
      }
      await Notification.markAllAsRead(context.user.id);
      return true;
    }
  },

  User: {
    notifications: async (user) => {
      const result = await Notification.getUserNotifications(user.id, { limit: 10 });
      return result.notifications.map(n => ({
        ...n.toObject(),
        id: n._id.toString()
      }));
    },
    unreadNotificationsCount: async (user) => {
      return await Notification.getUnreadCount(user.id);
    }
  },

  Notification: {
    user: async (notification) => {
      return await userService.getUserById(notification.userId);
    }
  }
};

module.exports = resolvers;
