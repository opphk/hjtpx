const DataLoader = require('dataloader');
const userService = require('../../services/userService');
const Notification = require('../../models/Notification');

const createUserLoader = () => {
  return new DataLoader(async (ids) => {
    const uniqueIds = [...new Set(ids)];
    const users = await Promise.all(
      uniqueIds.map(id => userService.getUserById(id))
    );
    
    const userMap = new Map();
    users.forEach((user, index) => {
      if (user) {
        userMap.set(user.id.toString(), user);
      }
    });
    
    return ids.map(id => userMap.get(id.toString()) || null);
  });
};

const createUsersLoader = () => {
  return new DataLoader(async (keys) => {
    const results = await Promise.all(
      keys.map(async ({ limit = 10, offset = 0 }) => {
        return await userService.getAllUsers({ limit, offset });
      })
    );
    return results;
  });
};

const createNotificationLoader = () => {
  return new DataLoader(async (ids) => {
    const uniqueIds = [...new Set(ids)];
    const notifications = await Notification.find({
      _id: { $in: uniqueIds }
    }).lean();
    
    const notificationMap = new Map();
    notifications.forEach(notification => {
      notificationMap.set(notification._id.toString(), {
        ...notification,
        id: notification._id.toString()
      });
    });
    
    return ids.map(id => notificationMap.get(id.toString()) || null);
  });
};

const createUserNotificationsLoader = () => {
  return new DataLoader(async (userIds) => {
    const uniqueUserIds = [...new Set(userIds)];
    const notifications = await Notification.find({
      userId: { $in: uniqueUserIds }
    }).sort({ createdAt: -1 }).limit(10).lean();
    
    const notificationMap = new Map();
    notifications.forEach(notification => {
      const userIdStr = notification.userId.toString();
      if (!notificationMap.has(userIdStr)) {
        notificationMap.set(userIdStr, []);
      }
      notificationMap.get(userIdStr).push({
        ...notification,
        id: notification._id.toString()
      });
    });
    
    return userIds.map(userId => 
      notificationMap.get(userId.toString()) || []
    );
  });
};

const createUnreadCountLoader = () => {
  return new DataLoader(async (userIds) => {
    const uniqueUserIds = [...new Set(userIds)];
    const counts = await Notification.aggregate([
      { $match: { userId: { $in: uniqueUserIds }, status: 'unread' } },
      { $group: { _id: '$userId', count: { $sum: 1 } } }
    ]);
    
    const countMap = new Map();
    counts.forEach(item => {
      countMap.set(item._id.toString(), item.count);
    });
    
    return userIds.map(userId => countMap.get(userId.toString()) || 0);
  });
};

const createLoaders = () => ({
  user: createUserLoader(),
  users: createUsersLoader(),
  notification: createNotificationLoader(),
  userNotifications: createUserNotificationsLoader(),
  unreadCount: createUnreadCountLoader()
});

module.exports = {
  createLoaders,
  createUserLoader,
  createUsersLoader,
  createNotificationLoader,
  createUserNotificationsLoader,
  createUnreadCountLoader
};
