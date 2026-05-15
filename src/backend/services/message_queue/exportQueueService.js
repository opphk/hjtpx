const { producerManager } = require('./producers/streamProducer');
const { consumerManager } = require('./consumers/streamConsumer');

class ExportQueueService {
  constructor() {
    this.queueName = 'export';
  }

  async requestExport(userId, exportType, parameters = {}, options = {}) {
    return await producerManager.send(this.queueName, {
      userId,
      exportType,
      parameters,
      status: 'pending',
      createdAt: new Date().toISOString()
    }, {
      type: 'export_request',
      priority: options.priority || 0,
      correlationId: options.correlationId
    });
  }

  async exportUsers(userId, filters = {}, options = {}) {
    return await this.requestExport(userId, 'users', {
      filters,
      fields: options.fields || ['id', 'username', 'email', 'createdAt']
    }, options);
  }

  async exportAnalytics(userId, dateRange, options = {}) {
    return await this.requestExport(userId, 'analytics', {
      startDate: dateRange.startDate,
      endDate: dateRange.endDate,
      metrics: options.metrics || ['users', 'sessions', 'conversions']
    }, options);
  }

  async exportNotifications(userId, filters = {}, options = {}) {
    return await this.requestExport(userId, 'notifications', {
      filters,
      fields: options.fields || ['id', 'type', 'title', 'message', 'createdAt']
    }, options);
  }

  async exportToCSV(userId, dataType, query = {}, options = {}) {
    return await this.requestExport(userId, 'csv', {
      dataType,
      query,
      options
    }, options);
  }

  async exportToExcel(userId, dataType, query = {}, options = {}) {
    return await this.requestExport(userId, 'excel', {
      dataType,
      query,
      options
    }, options);
  }

  async exportToPDF(userId, dataType, query = {}, options = {}) {
    return await this.requestExport(userId, 'pdf', {
      dataType,
      query,
      options
    }, options);
  }

  async startConsumer(options = {}) {
    const consumer = await consumerManager.createConsumer(this.queueName, options);

    consumer.registerHandler('export_request', async (message) => {
      const exportService = require('../exportService');
      const notificationService = require('../notificationService');

      const { userId, exportType, parameters } = message.payload;

      try {
        let result;

        switch (exportType) {
          case 'users':
            result = await exportService.exportUsers(parameters.filters, parameters.fields);
            break;
          case 'analytics':
            result = await exportService.exportAnalytics(parameters);
            break;
          case 'notifications':
            result = await exportService.exportNotifications(parameters.filters, parameters.fields);
            break;
          case 'csv':
            result = await exportService.exportToCSV(parameters.dataType, parameters.query);
            break;
          case 'excel':
            result = await exportService.exportToExcel(parameters.dataType, parameters.query);
            break;
          case 'pdf':
            result = await exportService.exportToPDF(parameters.dataType, parameters.query);
            break;
          default:
            throw new Error(`Unknown export type: ${exportType}`);
        }

        await notificationService.createNotification(userId, {
          type: 'in_app',
          title: 'Export Complete',
          message: `Your ${exportType} export is ready for download.`,
          data: { exportId: result.exportId, downloadUrl: result.downloadUrl }
        });

        console.log(`[ExportQueue] Export ${exportType} completed for user ${userId}`);

      } catch (error) {
        console.error(`[ExportQueue] Export failed for user ${userId}:`, error);

        await notificationService.createNotification(userId, {
          type: 'in_app',
          title: 'Export Failed',
          message: `Failed to export ${exportType}: ${error.message}`,
          data: { error: error.message }
        });

        throw error;
      }
    });

    return consumer;
  }
}

const exportQueueService = new ExportQueueService();

module.exports = exportQueueService;
