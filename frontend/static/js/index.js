(function(globalContext) {
    'use strict';

    var CaptchaModules = {
        version: '2.0.0',
        initialized: false,
        modules: {}
    };

    /**
     * 核心模块
     */
    CaptchaModules.Core = {
        Crypto: null,
        EnvironmentDetector: null
    };

    /**
     * 工具函数
     */
    CaptchaModules.Utils = null;

    /**
     * 常量定义
     */
    CaptchaModules.Constants = null;

    /**
     * UI组件
     */
    CaptchaModules.UI = null;

    /**
     * 初始化所有模块
     * 按照依赖顺序加载各个模块
     */
    function initialize() {
        if (CaptchaModules.initialized) {
            console.warn('CaptchaModules 已经初始化，无需重复初始化');
            return CaptchaModules;
        }

        CaptchaModules.Constants = globalContext.CaptchaConstants;
        CaptchaModules.Utils = globalContext.CaptchaUtils;
        CaptchaModules.UI = globalContext.UIModule;
        CaptchaModules.Core.Crypto = globalContext.CryptoModule;
        CaptchaModules.Core.EnvironmentDetector = globalContext.EnvironmentDetectorCore;

        CaptchaModules.initialized = true;

        if (CaptchaModules.Utils && CaptchaModules.Utils.getTimestamp) {
            console.log('CaptchaModules 初始化完成', {
                version: CaptchaModules.version,
                timestamp: CaptchaModules.Utils.getTimestamp()
            });
        }

        return CaptchaModules;
    }

    /**
     * 创建验证码实例
     * @param {Object} options - 配置选项
     * @returns {Object} 验证码实例
     */
    function createCaptchaInstance(options) {
        if (!CaptchaModules.initialized) {
            initialize();
        }

        var instance = {
            options: Object.assign({
                apiBase: '/api/v1',
                captchaType: 'slider'
            }, options),

            crypto: CaptchaModules.Core.Crypto,
            detector: null,
            ui: CaptchaModules.UI,
            utils: CaptchaModules.Utils
        };

        if (CaptchaModules.Core.EnvironmentDetector) {
            instance.detector = new CaptchaModules.Core.EnvironmentDetector({
                apiBase: instance.options.apiBase
            });
        }

        instance.verify = async function(data) {
            if (!instance.crypto) {
                throw new Error('加密模块未加载');
            }

            var salt = instance.utils.generateRandomString(16);
            var encrypted = await instance.crypto.encryptData(data, salt);
            var signature = instance.crypto.generateSignature(
                Date.now(),
                salt,
                encrypted
            );

            return {
                encrypted: encrypted,
                salt: salt,
                signature: signature,
                timestamp: Date.now()
            };
        };

        instance.detect = async function() {
            if (!instance.detector) {
                console.warn('环境检测模块未加载');
                return null;
            }

            return await instance.detector.runAll();
        };

        return instance;
    }

    /**
     * 加载模块脚本
     * @param {string} modulePath - 模块路径
     * @param {Function} callback - 加载完成后的回调
     */
    function loadModule(modulePath, callback) {
        var script = document.createElement('script');
        script.src = modulePath;
        script.async = true;

        script.onload = function() {
            if (callback) {
                callback(null, modulePath);
            }
        };

        script.onerror = function(error) {
            console.error('模块加载失败:', modulePath, error);
            if (callback) {
                callback(error, modulePath);
            }
        };

        document.head.appendChild(script);
    }

    /**
     * 按顺序加载所有模块
     * @param {Function} callback - 加载完成后的回调
     */
    function loadAllModules(callback) {
        var basePath = '';

        var moduleList = [
            basePath + 'constants/constants.js',
            basePath + 'utils/utils.js',
            basePath + 'core/crypto-module.js',
            basePath + 'core/environment-detector-core.js',
            basePath + 'components/ui-components.js'
        ];

        var loadedCount = 0;
        var hasError = false;

        function checkComplete(error) {
            if (hasError) return;

            if (error) {
                hasError = true;
                if (callback) callback(error);
                return;
            }

            loadedCount++;
            if (loadedCount >= moduleList.length) {
                initialize();
                if (callback) callback(null);
            }
        }

        moduleList.forEach(function(path) {
            loadModule(path, checkComplete);
        });
    }

    /**
     * 导出公共API
     */
    CaptchaModules.init = initialize;
    CaptchaModules.loadAll = loadAllModules;
    CaptchaModules.loadModule = loadModule;
    CaptchaModules.createInstance = createCaptchaInstance;
    CaptchaModules.VERSION = '2.0.0';

    /**
     * 快速初始化方法
     * 自动加载所有模块并初始化
     */
    CaptchaModules.quickInit = function(callback) {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', function() {
                loadAllModules(callback);
            });
        } else {
            loadAllModules(callback);
        }
    };

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CaptchaModules;
    } else {
        globalContext.CaptchaModules = CaptchaModules;
    }

})(typeof window !== 'undefined' ? window : this);
