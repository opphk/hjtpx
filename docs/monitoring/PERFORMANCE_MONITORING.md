# 性能监控指南

## 概述

本文档详细说明了HJTPX项目的性能监控体系架构、配置方法和使用指南。该监控体系整合了Sentry APM、Prometheus和自定义监控中间件，为项目提供全方位的性能观测能力。

## 1. 监控架构

### 1.1 整体架构

HJTPX的性能监控体系采用多层次、多维度的设计思路，主要包括以下几个核心组件：

**后端监控层**：基于Node.js的Express应用，通过自定义中间件和Sentry APM实现全面的后端性能追踪。该层负责监控API响应时间、数据库查询性能、Redis操作延迟、系统资源使用情况等关键指标。

**前端监控层**：基于React应用，集成了Sentry的Browser SDK和Performance API，实现页面加载性能、用户交互行为、API调用性能等前端指标的采集。

**告警系统**：使用Prometheus和Alertmanager构建的告警体系，支持多级别的告警规则配置，包括警告（Warning）、严重（Critical）和信息（Info）三个级别。

**数据可视化**：通过Prometheus收集和存储指标数据，配合Grafana等工具实现数据的可视化展示，帮助团队快速了解系统运行状态。

### 1.2 技术选型理由

选择Sentry作为APM工具的主要考虑因素包括：原生支持Node.js和React生态系统，提供开箱即用的Express和MongoDB集成，具有强大的错误追踪和性能分析能力，支持分布式追踪和采样策略配置。

Prometheus的选择则基于其强大的指标采集能力和灵活的告警规则配置，与Kubernetes生态系统的深度集成，以及丰富的客户端库支持。

## 2. 后端性能监控配置

### 2.1 Sentry APM配置详解

Sentry APM是后端监控的核心组件，通过`src/backend/config/sentry.js`文件进行配置。该配置提供了环境自适应的采样率设置、智能的错误过滤机制以及性能追踪优化。

```javascript
const Sentry = require('@sentry/node');
const { nodeProfilingIntegration } = require('@sentry/profiling-node');

function initSentry(app) {
  if (!process.env.SENTRY_DSN) {
    console.log('⚠️ SENTRY_DSN 未配置，Sentry 将不会启动');
    return;
  }

  const environment = process.env.SENTRY_ENVIRONMENT || process.env.NODE_ENV || 'development';
  const tracesSampleRate = parseFloat(process.env.SENTRY_TRACES_SAMPLE_RATE);
  const profilesSampleRate = parseFloat(process.env.SENTRY_PROFILES_SAMPLE_RATE);

  Sentry.init({
    dsn: process.env.SENTRY_DSN,
    environment,
    release: process.env.SENTRY_RELEASE || 'hjtpx@1.0.0',
    
    integrations: [
      new Sentry.Integrations.Http({ tracing: true }),
      new Sentry.Integrations.Express({ app }),
      new Sentry.Integrations.Mongo(),
      new Sentry.Integrations.Postgres(),
      nodeProfilingIntegration(),
    ],
    
    tracesSampleRate: !isNaN(tracesSampleRate) ? tracesSampleRate : (environment === 'production' ? 0.1 : 1.0),
    profilesSampleRate: !isNaN(profilesSampleRate) ? profilesSampleRate : (environment === 'production' ? 0.05 : 1.0),
    
    normalizeDepth: 5,
    sendDefaultPii: false,
    
    ignoreErrors: [
      'Network Error',
      'Failed to fetch',
      'Network request failed',
      'ECONNREFUSED',
      'ETIMEDOUT',
      'socket hang up'
    ],
    
    beforeSend(event, hint) {
      const error = hint.originalException;
      if (error && error.message) {
        if (error.message.includes('Network Error') || 
            error.message.includes('timeout') ||
            error.message.includes('ECONNREFUSED') ||
            error.message.includes('ETIMEDOUT')) {
          event.tags = event.tags || {};
          event.tags.network_error = 'true';
          event.level = 'warning';
        }
      }
      return event;
    },
    
    beforeSendTransaction(event) {
      if (!event.transaction) return null;
      
      const ignoredPaths = ['/health', '/metrics', '/favicon.ico'];
      if (ignoredPaths.some(path => event.transaction.includes(path))) {
        return null;
      }
      
      if (event.spans && event.spans.length > 100) {
        event.spans = event.spans.slice(-100);
      }
      
      return event;
    }
  });
}
```

### 2.2 环境变量配置

后端Sentry配置需要设置以下环境变量，这些变量应在部署配置中根据环境进行适当调整：

**SENTRY_DSN**：Sentry的数据源名称，是连接Sentry服务的关键标识。该值可以在Sentry项目设置中获取，格式为`https://<key>@sentry.io/<project>`。

**SENTRY_ENVIRONMENT**：指定应用程序的运行环境，如development、staging或production。不同的环境可以使用不同的采样率和告警策略。

**SENTRY_RELEASE**：应用程序的版本号，用于关联特定版本的错误和性能数据。推荐使用语义化版本号，如`hjtpx@1.0.0`。

**SENTRY_TRACES_SAMPLE_RATE**：性能追踪的采样率，取值范围为0到1。生产环境建议设置为0.1（10%），开发环境可以设置为1.0（100%）。

**SENTRY_PROFILES_SAMPLE_RATE**：性能分析的采样率，用于CPU和内存分析。生产环境建议设置为0.05（5%）。

### 2.3 性能监控中间件

`src/backend/middleware/enhancedPerformanceMonitor.js`提供了增强的性能监控功能，包括请求耗时追踪、数据库查询监控和Redis操作监控。

该中间件的核心功能包括：自动检测慢请求并记录警告、集成Sentry的Breadcrumb功能提供详细的性能上下文、支持可配置的慢请求阈值设置。

```javascript
const performanceMiddleware = (req, res, next) => {
  const startTime = process.hrtime.bigint();
  const requestId = req.requestId || `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  
  req.performanceStartTime = startTime;
  req.requestId = requestId;
  res.setHeader('X-Request-ID', requestId);
  
  // 性能数据记录逻辑
  res.on('finish', () => {
    const duration = Number(process.hrtime.bigint() - startTime) / 1e6;
    if (duration > SLOW_REQUEST_THRESHOLD) {
      console.warn(`慢请求: ${req.method} ${req.path} ${duration.toFixed(2)}ms ${res.statusCode}`);
    }
  });
  
  next();
};
```

### 2.4 数据库查询监控

DatabaseQueryMonitor类提供了针对数据库操作的专项监控能力。它能够追踪每个查询的执行时间，并在检测到慢查询时自动记录警告信息。

该监控器通过计算查询开始和结束时间戳的差值来获取精确的执行时间，并将结果记录到Prometheus指标中。慢查询的判定阈值可以通过环境变量SLOW_QUERY_THRESHOLD进行配置，默认值为100毫秒。

### 2.5 Redis操作监控

RedisOperationMonitor类专门用于监控Redis操作的性能表现。它追踪所有Redis操作的执行时间，并在操作耗时超过阈值时发出警告。

Redis操作的慢操作阈值由环境变量SLOW_REDIS_THRESHOLD控制，默认值为50毫秒。通过这个监控器，可以及时发现Redis性能问题并进行优化。

## 3. 前端性能监控配置

### 3.1 Sentry前端集成

前端性能监控通过`src/frontend/src/utils/sentry.js`进行配置。需要在React应用的入口文件中初始化Sentry，通常在main.jsx或App.jsx中进行。

```javascript
import * as Sentry from "@sentry/react";
import { browserTracingIntegration, replayIntegration } from "@sentry/react";

const environment = import.meta.env.MODE || 'development';

export function initSentry() {
  const dsn = import.meta.env.VITE_SENTRY_DSN;
  
  if (!dsn) {
    console.log('⚠️ VITE_SENTRY_DSN 未配置，Sentry 前端监控将不会启动');
    return;
  }

  Sentry.init({
    dsn,
    environment,
    release: import.meta.env.VITE_APP_VERSION || 'hjtpx-frontend@1.0.0',
    
    integrations: [
      browserTracingIntegration({
        tracePropagationTargets: ['localhost', /^\/api\//],
      }),
      replayIntegration({
        maskAllText: true,
        maskAllInputs: true,
        blockAllMedia: true,
      }),
    ],
    
    tracesSampleRate: !isNaN(tracesSampleRate) ? tracesSampleRate : (environment === 'production' ? 0.1 : 1.0),
    replaysSessionSampleRate: !isNaN(replaysSampleRate) ? replaysSampleRate : (environment === 'production' ? 0.05 : 0.1),
    replaysOnErrorSampleRate: 1.0,
    
    ignoreErrors: [
      'Network Error',
      'Failed to fetch',
      'ResizeObserver loop',
    ],
    
    beforeSend(event) {
      if (event.request && event.request.headers) {
        delete event.request.headers['Authorization'];
        delete event.request.headers['Cookie'];
      }
      return event;
    },
  });
}
```

### 3.2 性能监控Hook

`src/frontend/src/hooks/usePerformanceMonitoring.js`提供了React Hook形式的性能监控功能，使组件能够方便地获取和上报性能数据。

```javascript
import { usePerformanceMonitoring } from '../hooks/usePerformanceMonitoring';

function MyComponent() {
  const { measurePageLoad, trackApiCall, trackInteraction } = usePerformanceMonitoring();
  
  useEffect(() => {
    // 页面加载性能测量自动进行
  }, []);
  
  const handleButtonClick = () => {
    trackInteraction('submit-button', 'click');
  };
  
  return <button onClick={handleButtonClick}>提交</button>;
}
```

### 3.3 API监控工具

`src/frontend/src/utils/apiMonitor.js`提供了自动化的API调用监控功能，它通过包装原生fetch函数来实现对所有网络请求的性能追踪。

```javascript
import { apiMonitor, setupApiMonitoring } from '../utils/apiMonitor';

// 在应用初始化时启用API监控
setupApiMonitoring();
```

该工具会自动记录每个API请求的端点、方法、状态码和响应时间，并将这些数据上报到Sentry进行分析。

### 3.4 前端环境变量

前端Sentry配置需要设置以下环境变量，通常在`.env`或`.env.production`文件中：

**VITE_SENTRY_DSN**：前端Sentry的数据源名称，应与后端分开创建项目以便于区分和管理。

**VITE_SENTRY_TRACES_SAMPLE_RATE**：前端性能追踪的采样率，生产环境建议设置为0.1。

**VITE_SENTRY_REPLAYS_SAMPLE_RATE**：会话回放功能的采样率，生产环境建议设置为0.05。

**VITE_APP_VERSION**：前端应用的版本号，用于版本间的性能对比分析。

## 4. Prometheus监控指标

### 4.1 HTTP请求指标

`src/backend/services/metricsService.js`定义了所有HTTP相关的Prometheus指标，包括请求持续时间直方图、请求总数计数器和错误计数。

http_request_duration_seconds直方图以秒为单位记录HTTP请求的处理时间，标签包括method（HTTP方法）、route（路由路径）和status_code（状态码）。直方图的bucket设置为[0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10]，覆盖了从1毫秒到10秒的请求时间范围。

http_requests_total计数器记录所有HTTP请求的数量，通过它可以计算QPS和RPS等流量指标。

http_request_errors_total计数器专门记录错误请求，其error_type标签区分server_error（5xx错误）和client_error（4xx错误）。

### 4.2 数据库查询指标

database_query_duration_seconds直方图记录数据库查询的执行时间，标签为query_type（查询类型，如SELECT、INSERT、UPDATE、DELETE）和table（操作的表名）。

database_query_errors_total计数器记录数据库查询错误，error_code标签记录具体的错误代码，便于问题诊断。

### 4.3 Redis操作指标

redis_operation_duration_seconds直方图记录Redis操作的执行时间，标签为operation（操作类型，如GET、SET、DEL）和status（操作状态，success或error）。

### 4.4 系统资源指标

系统资源指标包括内存使用量和CPU使用率两个方面。memoryUsageGauge使用type标签区分heap_used、heap_total、rss等不同类型的内存指标。

cpuUsageGauge记录进程的CPU使用百分比，用于监控应用程序的资源消耗情况。

### 4.5 业务指标

cache_hits_total和cache_misses_total计数器记录缓存的命中和未命中情况，用于分析缓存策略的有效性。

authentication_attempts_total记录认证尝试次数，result标签区分success和failure，便于发现异常的认证行为。

rate_limit_hits_total记录限流触发的次数，有助于评估限流策略的配置是否合理。

## 5. 告警规则配置

### 5.1 告警级别定义

告警系统采用三级分类体系，不同级别的告警对应不同的响应优先级和处理流程。

**Critical（严重）**：表示系统出现严重影响，需要立即响应。这类告警通常包括服务不可用、错误率异常升高、内存耗尽等可能导致系统崩溃的情况。

**Warning（警告）**：表示系统性能下降或存在潜在风险，需要在数小时内响应。包括响应时间延长、CPU使用率高、缓存命中率下降等情况。

**Info（信息）**：用于提供系统运行状态信息，不一定需要立即处理。包括流量异常、无流量等状态提示。

### 5.2 API性能告警

HighErrorRate告警在5分钟内错误率超过5%时触发，持续5分钟则升级为严重告警。该告警基于http_request_errors_total和http_requests_total指标的比值计算。

HighResponseTime告警在95百分位响应时间超过3秒时触发，CriticalResponseTime则在99百分位响应时间超过5秒时触发。这两个告警帮助团队及时发现响应时间退化问题。

SlowDatabaseQueries告警在数据库查询的95百分位执行时间超过1秒时触发，CriticalSlowDatabaseQueries则在99百分位超过3秒时触发。

SlowRedisOperations告警在Redis操作95百分位执行时间超过100毫秒时触发，提醒团队关注缓存层的性能问题。

### 5.3 系统资源告警

HighCPUUsage告警在CPU使用率超过80%且持续10分钟时触发，CriticalCPUUsage则在超过95%且持续5分钟时触发。

HighMemoryUsage告警在内存使用率超过85%且持续10分钟时触发，CriticalMemoryUsage则在超过95%且持续5分钟时触发。HighHeapMemoryUsage专门监控Node.js堆内存使用率，超过90%时触发。

### 5.4 基础设施告警

InstanceDown告警在服务实例离线超过2分钟时触发，通常意味着Kubernetes Pod或服务器出现问题。

RedisDown和PostgreSQLDown告警在相应的服务离线超过1分钟时触发，用于监控依赖服务的可用性。

DiskSpaceLow告警在磁盘空间低于10%时触发，防止因磁盘空间不足导致的服务中断。

### 5.5 业务告警

NoTraffic告警在10分钟内没有任何HTTP流量时触发，可能表示服务被意外停止或负载均衡配置问题。

UnusualTraffic告警在QPS超过1000时触发，帮助发现流量异常情况。

HighErrorRateTrend和SlowResponseTrend告警通过增量计算检测错误率和响应时间的趋势性变化，提供早期预警。

## 6. 监控最佳实践

### 6.1 采样策略优化

生产环境的采样率设置需要平衡数据完整性和成本控制。推荐的生产环境采样策略为：tracesSampleRate设置为0.1（10%），profilesSampleRate设置为0.05（5%），replaysSessionSampleRate设置为0.05（5%）。

对于关键业务路径，可以使用Sentry.setContext和Sentry.startTransaction API进行强制追踪，确保重要请求不会被采样遗漏。

### 6.2 告警阈值调优

初始部署时建议使用较为宽松的告警阈值，在积累足够的历史数据后再进行精细调整。可以通过Prometheus的查询功能分析历史数据，确定适合业务的告警阈值。

建议至少收集一周的数据后再进行阈值优化，以确保考虑到工作日和周末、业务高峰期和非高峰期等不同场景。

### 6.3 性能数据关联

利用Sentry的分布式追踪功能，将前端和后端的性能数据关联起来。通过在API请求头中添加X-Request-ID，可以在Sentry中追踪一个请求从浏览器到数据库的完整链路。

后端中间件会自动将请求ID添加到响应头中，前端监控可以读取这个ID并与后端的追踪数据进行关联。

### 6.4 慢查询优化

根据SlowDatabaseQueries告警的分析结果，对高频出现的慢查询进行优化。常见的优化手段包括：添加适当的索引、优化查询语句、使用缓存策略。

建议为所有超过500毫秒的查询建立优化清单，定期进行查询性能评审。

## 7. 故障排除

### 7.1 Sentry数据未上报

如果Sentry控制台中没有看到预期的错误或性能数据，首先检查SENTRY_DSN环境变量是否正确配置。确保DSN格式为`https://<key>@sentry.io/<project>`，且key和project ID都正确。

检查应用程序启动时是否有Sentry初始化的日志输出。如果看到"⚠️ SENTRY_DSN 未配置"的提示，说明环境变量未正确加载。

确认网络连接是否正常，Sentry需要能够访问sentry.io服务器。在内网环境中可能需要配置代理或使用Sentry的自托管版本。

### 7.2 Prometheus指标缺失

如果Prometheus中没有看到预期的指标，首先确认指标端点是否可访问。默认的指标端点为`/api/v1/metrics/prometheus`。

检查Prometheus配置中的scrape_targets是否正确指向应用程序实例。确认job_name和targets配置与docker-compose或Kubernetes服务配置一致。

使用Prometheus的TUI界面或PromQL查询工具直接查询指标，确认指标名称和标签是否正确。

### 7.3 告警未触发

如果告警规则配置正确但告警未触发，首先检查告警规则的for参数。Prometheus要求指标持续超过阈值达到for指定的时间才会触发告警。

确认Alertmanager配置正确，能够接收和转发告警通知。可以在Prometheus的alerts页面查看告警的当前状态。

检查时间同步服务是否正常运行，时间不一致可能导致告警计算出现偏差。

### 7.4 性能数据不准确

如果观察到异常的性能数据，可能是由于系统负载过高导致测量不准确。性能监控本身也会消耗一定的系统资源，建议在高负载场景下适当降低采样率。

确保使用高精度计时器（如process.hrtime.bigint()）进行时间测量，避免使用Date.now()导致的毫秒级精度损失。

## 8. 监控指标参考

### 8.1 关键性能指标

**P50响应时间**：中位数响应时间，表示50%的请求在此时间内完成。这个指标反映系统的典型响应表现。

**P95响应时间**：95百分位响应时间，表示95%的请求在此时间内完成。这是设置SLA承诺的主要参考指标。

**P99响应时间**：99百分位响应时间，反映系统在最坏情况下的表现。对于金融交易等场景，这个指标尤为重要。

**错误率**：错误请求占总请求的比例，通常以百分比表示。健康的系统错误率应低于1%。

**吞吐量**：单位时间内处理的请求数量，通常用QPS（每秒查询数）或RPS（每秒请求数）表示。

### 8.2 资源使用指标

**CPU使用率**：进程占用CPU的比例。在多核系统中，需要关注所有核心的平均使用率。

**内存使用率**：进程或系统使用的内存占总内存的比例。内存泄漏会导致使用率持续上升。

**堆内存使用率**：Node.js进程的V8堆内存使用情况。堆内存接近上限会导致频繁的垃圾回收，影响性能。

**连接数**：活跃的网络连接数量，包括HTTP连接、数据库连接和Redis连接。过高的连接数可能导致资源耗尽。

### 8.3 缓存指标

**缓存命中率**：缓存命中次数占总访问次数的比例。命中率越高说明缓存策略越有效。

**缓存未命中率**：缓存未命中次数，反映需要直接访问数据源的频率。

**平均缓存延迟**：缓存操作的平均响应时间，反映缓存层的性能表现。

## 9. 环境配置示例

### 9.1 开发环境配置

开发环境的监控配置应注重调试便利性，建议启用完整的追踪功能：

```bash
# .env.development
SENTRY_DSN=https://xxx@sentry.io/hjtpx-dev
SENTRY_ENVIRONMENT=development
SENTRY_TRACES_SAMPLE_RATE=1.0
SENTRY_PROFILES_SAMPLE_RATE=1.0
SENTRY_DEBUG=true
```

前端开发环境配置：

```bash
# .env.development
VITE_SENTRY_DSN=https://xxx@sentry.io/hjtpx-frontend-dev
VITE_SENTRY_TRACES_SAMPLE_RATE=1.0
VITE_SENTRY_REPLAYS_SAMPLE_RATE=1.0
```

### 9.2 生产环境配置

生产环境的监控配置应注重成本控制和关键数据采集：

```bash
# .env.production
SENTRY_DSN=https://xxx@sentry.io/hjtpx-prod
SENTRY_ENVIRONMENT=production
SENTRY_RELEASE=hjtpx@2.0.0
SENTRY_TRACES_SAMPLE_RATE=0.1
SENTRY_PROFILES_SAMPLE_RATE=0.05
SENTRY_DEBUG=false
```

前端生产环境配置：

```bash
# .env.production
VITE_SENTRY_DSN=https://xxx@sentry.io/hjtpx-frontend-prod
VITE_SENTRY_TRACES_SAMPLE_RATE=0.1
VITE_SENTRY_REPLAYS_SAMPLE_RATE=0.05
VITE_APP_VERSION=2.0.0
```

### 9.3 告警阈值配置

根据业务需求和系统容量，可以通过环境变量调整告警阈值：

```bash
SLOW_REQUEST_THRESHOLD=2000
SLOW_QUERY_THRESHOLD=500
SLOW_REDIS_THRESHOLD=100
```

## 10. 相关文档

本监控体系与项目的其他部分紧密相关，以下文档可能对您有帮助：

- [Sentry配置指南](../SENTRY_CONFIG.md)：关于Sentry集成的详细说明
- [监控系统部署](../monitoring/docker-compose.monitoring.yml)：Docker Compose监控栈配置
- [API文档](../api-docs/README.md)：API端点及其性能特性说明
- [性能优化指南](./PERFORMANCE_OPTIMIZATION.md)：基于监控数据的优化建议

## 更新日志

**2026-05-15**：初始版本发布，包含Sentry APM集成、Prometheus指标采集、告警规则配置和前后端性能监控功能。
