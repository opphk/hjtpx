const mongoose = require('mongoose');

const notificationSchema = new mongoose.Schema(
  {
    userId: {
      type: mongoose.Schema.Types.ObjectId,
      ref: 'User',
      required: true,
      index: true
    },
    type: {
      type: String,
      required: true,
      enum: [
        'info',
        'success',
        'warning',
        'error',
        'system',
        'message',
        'reminder',
        'alert'
      ],
      default: 'info'
    },
    title: {
      type: String,
      required: true,
      trim: true,
      maxlength: 200
    },
    message: {
      type: String,
      required: true,
      trim: true,
      maxlength: 1000
    },
    data: {
      type: mongoose.Schema.Types.Mixed,
      default: {}
    },
    priority: {
      type: String,
      enum: ['low', 'normal', 'high', 'urgent'],
      default: 'normal'
    },
    status: {
      type: String,
      enum: ['unread', 'read', 'archived'],
      default: 'unread',
      index: true
    },
    readAt: {
      type: Date,
      default: null
    },
    expiresAt: {
      type: Date,
      default: null,
      index: true
    },
    actionUrl: {
      type: String,
      default: null
    },
    actionLabel: {
      type: String,
      maxlength: 50
    },
    channels: {
      type: [String],
      enum: ['in_app', 'email', 'sms', 'push'],
      default: ['in_app']
    },
    metadata: {
      source: String,
      correlationId: String,
      tags: [String]
    }
  },
  {
    timestamps: true,
    toJSON: {
      transform: function (doc, ret) {
        delete ret.__v;
        return ret;
      }
    }
  }
);

notificationSchema.index({ userId: 1, status: 1, createdAt: -1 });
notificationSchema.index({ userId: 1, type: 1 });
notificationSchema.index({ expiresAt: 1 }, { expireAfterSeconds: 0 });

notificationSchema.statics.createNotification = async function (data) {
  const notification = new this(data);
  return notification.save();
};

notificationSchema.statics.createBulkNotifications = async function (notifications) {
  return this.insertMany(notifications, { ordered: false });
};

notificationSchema.statics.getUserNotifications = async function (
  userId,
  options = {}
) {
  const {
    status = null,
    type = null,
    page = 1,
    limit = 20,
    sortBy = 'createdAt',
    order = 'desc'
  } = options;

  const query = { userId };
  if (status) query.status = status;
  if (type) query.type = type;

  const skip = (page - 1) * limit;
  const sort = { [sortBy]: order === 'asc' ? 1 : -1 };

  const [notifications, total] = await Promise.all([
    this.find(query).sort(sort).skip(skip).limit(limit),
    this.countDocuments(query)
  ]);

  return {
    notifications,
    pagination: {
      page,
      limit,
      total,
      pages: Math.ceil(total / limit)
    }
  };
};

notificationSchema.statics.getUnreadCount = async function (userId) {
  return this.countDocuments({ userId, status: 'unread' });
};

notificationSchema.statics.markAsRead = async function (notificationId, userId) {
  return this.updateOne(
    { _id: notificationId, userId },
    { status: 'read', readAt: new Date() }
  );
};

notificationSchema.statics.markAllAsRead = async function (userId) {
  return this.updateMany(
    { userId, status: 'unread' },
    { status: 'read', readAt: new Date() }
  );
};

notificationSchema.statics.archiveOld = async function (daysOld = 30) {
  const cutoffDate = new Date();
  cutoffDate.setDate(cutoffDate.getDate() - daysOld);

  return this.updateMany(
    {
      status: 'read',
      createdAt: { $lt: cutoffDate }
    },
    { status: 'archived' }
  );
};

const Notification = mongoose.model('Notification', notificationSchema);

module.exports = Notification;
