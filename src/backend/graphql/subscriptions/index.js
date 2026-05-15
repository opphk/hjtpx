const { PubSub } = require('graphql-subscriptions');
const { withFilter } = require('graphql-subscriptions');

const pubsub = new PubSub();

const NOTIFICATION_CREATED = 'NOTIFICATION_CREATED';
const NOTIFICATION_UPDATED = 'NOTIFICATION_UPDATED';
const NOTIFICATION_DELETED = 'NOTIFICATION_DELETED';
const USER_UPDATED = 'USER_UPDATED';

const notificationCreated = {
  subscribe: withFilter(
    (_, { userId }, context) => {
      if (userId) {
        return pubsub.asyncIterator(`${NOTIFICATION_CREATED}_${userId}`);
      }
      return pubsub.asyncIterator(NOTIFICATION_CREATED);
    },
    (payload, variables) => {
      if (variables.userId) {
        return payload.notificationCreated.userId === variables.userId;
      }
      return true;
    }
  )
};

const notificationUpdated = {
  subscribe: withFilter(
    (_, { userId }, context) => {
      return pubsub.asyncIterator(`${NOTIFICATION_UPDATED}_${userId}`);
    },
    (payload, variables) => {
      return payload.notificationUpdated.userId === variables.userId;
    }
  )
};

const notificationDeleted = {
  subscribe: withFilter(
    (_, { userId }, context) => {
      return pubsub.asyncIterator(`${NOTIFICATION_DELETED}_${userId}`);
    },
    (payload, variables) => {
      return payload.notificationDeleted.userId === variables.userId;
    }
  )
};

const userUpdated = {
  subscribe: () => pubsub.asyncIterator(USER_UPDATED)
};

const publishNotificationCreated = async (notification, userId = null) => {
  const payload = {
    notificationCreated: {
      id: notification.id?.toString() || notification._id?.toString(),
      userId: notification.userId?.toString(),
      type: notification.type,
      title: notification.title,
      message: notification.message,
      priority: notification.priority,
      status: notification.status,
      channels: notification.channels || ['in_app'],
      createdAt: notification.createdAt?.toISOString?.() || new Date().toISOString()
    }
  };
  
  if (userId) {
    await pubsub.publish(`${NOTIFICATION_CREATED}_${userId}`, payload);
  }
  await pubsub.publish(NOTIFICATION_CREATED, payload);
};

const publishNotificationUpdated = async (notification, userId) => {
  await pubsub.publish(`${NOTIFICATION_UPDATED}_${userId}`, {
    notificationUpdated: {
      id: notification.id?.toString() || notification._id?.toString(),
      userId: notification.userId?.toString(),
      type: notification.type,
      title: notification.title,
      message: notification.message,
      priority: notification.priority,
      status: notification.status,
      updatedAt: notification.updatedAt?.toISOString?.() || new Date().toISOString()
    }
  });
};

const publishNotificationDeleted = async (notificationId, userId) => {
  await pubsub.publish(`${NOTIFICATION_DELETED}_${userId}`, {
    notificationDeleted: {
      id: notificationId.toString(),
      userId: userId.toString()
    }
  });
};

const publishUserUpdated = async (user) => {
  await pubsub.publish(USER_UPDATED, {
    userUpdated: {
      id: user.id?.toString() || user._id?.toString(),
      email: user.email,
      name: user.name,
      role: user.role,
      updatedAt: new Date().toISOString()
    }
  });
};

const subscriptionResolver = {
  notificationCreated,
  notificationUpdated,
  notificationDeleted,
  userUpdated
};

module.exports = {
  subscriptionResolver,
  publishNotificationCreated,
  publishNotificationUpdated,
  publishNotificationDeleted,
  publishUserUpdated,
  pubsub
};
