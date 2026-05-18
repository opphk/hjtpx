/**
 * v15.0 高级环境检测系统
 * 包含 WebGL 指纹深度检测、Canvas 指纹时序分析、浏览器插件检测、虚拟机多维度检测
 */
class AdvancedEnvironmentDetector {
    constructor(options = {}) {
        this.options = Object.assign({
            apiBase: '/api/v1',
            enableWebGLAnalysis: true,
            enableCanvasTiming: true,
            enablePluginDetection: true,
            enableVMDetection: true,
            enableTorCheck: true,
            maxTimingSamples: 10,
            riskThreshold: 50
        }, options);
        
        this.results = {};
        this.riskScore = 0;
        this.detectionId = 'adv_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
        this.weights = {
            webgl_anomaly: 15,
            webgl_rendering: 12,
            webgl2_support: 8,
            canvas_timing: 14,
            canvas_entropy: 10,
            canvas_render_consistency: 12,
            plugin_count: 6,
            plugin_types: 8,
            rare_plugins: 10,
            vm_cpu: 12,
            vm_memory: 10,
            vm_process: 14,
            vm_gpu: 15,
            vm_bios: 8,
            vm_registry: 12,
            tor_detected: 20,
            tor_exit_node: 18,
            dark_web: 16
        };
        
        this.softwareRenderers = [
            'swiftshader', 'llvmpipe', 'mesa', 'software', 'emulated',
            'virtual', 'vmware', 'virtualbox', 'parallels', 'qemu', 'kvm'
        ];
        
        this.vmProcessNames = [
            'vboxservice', 'vboxTray', 'vmtoolsd', 'vmusrvc', 'VMwareUser',
            'parallels', 'qemu', 'kvm', 'xenhpet', 'winxp', 'virtio'
        ];
        
        this.vmBiosPatterns = [
            'virtualbox', 'vmware', 'parallels', 'qemu', 'kvm', 'hyperv',
            ' seawolf', 'bochs', 'bunny'
        ];
        
        this.torExitNodes = [];
        this.torDetectorEndpoints = [
            'https://check.torproject.org/api/ip',
            'https://ipinfo.io/json'
        ];
    }
    
    /**
     * WebGL 指纹深度检测 - 渲染异常分析
     */
    async detectWebGLAnomaly() {
        let score = 0;
        const detections = [];
        
        try {
            const canvas = document.createElement('canvas');
            canvas.width = 512;
            canvas.height = 512;
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            
            if (!gl) {
                score += 50;
                detections.push('no_webgl_context');
                return { detected: true, score: Math.min(score, 100), detections };
            }
            
            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            if (!debugInfo) {
                score += 25;
                detections.push('webgl_debug_blocked');
            } else {
                const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
                const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                
                if (!vendor || !renderer) {
                    score += 20;
                    detections.push('webgl_no_vendor_renderer');
                }
                
                const rendererLower = renderer.toLowerCase();
                for (const sw of this.softwareRenderers) {
                    if (rendererLower.includes(sw)) {
                        score += 40;
                        detections.push(`software_renderer:${sw}`);
                    }
                }
                
                if (rendererLower.includes('unknown') || rendererLower.includes('generic')) {
                    score += 15;
                    detections.push('webgl_anonymized_renderer');
                }
            }
            
            const maxTextureSize = gl.getParameter(gl.MAX_TEXTURE_SIZE);
            if (maxTextureSize < 4096) {
                score += 15;
                detections.push(`low_max_texture:${maxTextureSize}`);
            }
            
            const maxRenderbufferSize = gl.getParameter(gl.MAX_RENDERBUFFER_SIZE);
            if (maxRenderbufferSize < 4096) {
                score += 15;
                detections.push(`low_renderbuffer:${maxRenderbufferSize}`);
            }
            
            const maxVertexAttribs = gl.getParameter(gl.MAX_VERTEX_ATTRIBS);
            if (maxVertexAttribs < 8) {
                score += 10;
                detections.push(`few_vertex_attribs:${maxVertexAttribs}`);
            }
            
            const aliasedLineWidthRange = gl.getParameter(gl.ALIASED_LINE_WIDTH_RANGE);
            if (aliasedLineWidthRange && aliasedLineWidthRange[1] <= 1) {
                score += 20;
                detections.push('aliased_rendering_only');
            }
            
            const shaderPrecision = gl.getShaderPrecisionFormat(gl.FRAGMENT_SHADER, gl.HIGH_FLOAT);
            if (shaderPrecision && shaderPrecision.precision < 16) {
                score += 25;
                detections.push('low_shader_precision');
            }
            
            const supportedExts = gl.getSupportedExtensions() || [];
            if (supportedExts.length < 15) {
                score += 20;
                detections.push(`few_extensions:${supportedExts.length}`);
            }
            
            const ext = gl.getExtension('EXT_texture_filter_anisotropic');
            if (!ext) {
                score += 10;
                detections.push('no_anisotropic_filter');
            }
            
            const rendererResult = await this.performRenderingTest(gl, canvas);
            if (rendererResult.anomalyDetected) {
                score += rendererResult.anomalyScore;
                detections.push(...rendererResult.anomalyDetails);
            }
            
        } catch (e) {
            score += 40;
            detections.push('webgl_detection_error');
        }
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }
    
    /**
     * WebGL2 渲染器分析
     */
    async detectWebGL2Rendering() {
        let score = 0;
        const detections = [];
        
        try {
            const canvas = document.createElement('canvas');
            const gl2 = canvas.getContext('webgl2');
            
            if (!gl2) {
                return { detected: false, score: 0, detections: ['webgl2_not_supported'] };
            }
            
            const debugInfo = gl2.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const renderer = gl2.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                const rendererLower = renderer.toLowerCase();
                
                for (const sw of this.softwareRenderers) {
                    if (rendererLower.includes(sw)) {
                        score += 35;
                        detections.push(`webgl2_software:${sw}`);
                    }
                }
            }
            
            const max3DTextureSize = gl2.getParameter(gl2.MAX_3D_TEXTURE_SIZE);
            if (max3DTextureSize < 512) {
                score += 15;
                detections.push(`limited_3d_texture:${max3DTextureSize}`);
            }
            
            const maxSamples = gl2.getParameter(gl2.MAX_SAMPLES);
            if (maxSamples < 4) {
                score += 10;
                detections.push(`low_sample_count:${maxSamples}`);
            }
            
            const vertexOutputComponents = gl2.getParameter(gl2.MAX_VERTEX_OUTPUT_COMPONENTS);
            if (vertexOutputComponents < 64) {
                score += 10;
                detections.push(`low_vertex_output:${vertexOutputComponents}`);
            }
            
        } catch (e) {
            score += 20;
            detections.push('webgl2_analysis_error');
        }
        
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }
    
    /**
     * 执行渲染测试以检测异常
     */
    async performRenderingTest(gl, canvas) {
        const result = {
            anomalyDetected: false,
            anomalyScore: 0,
            anomalyDetails: []
        };
        
        try {
            gl.clearColor(0.0, 1.0, 0.0, 1.0);
            gl.clear(gl.COLOR_BUFFER_BIT);
            
            const pixels = new Uint8Array(4);
            gl.readPixels(canvas.width / 2, canvas.height / 2, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, pixels);
            
            if (pixels[0] !== 0 || pixels[1] !== 255 || pixels[2] !== 0 || pixels[3] !== 255) {
                result.anomalyDetected = true;
                result.anomalyScore += 25;
                result.anomalyDetails.push('render_color_mismatch');
            }
            
            gl.clearColor(1.0, 0.0, 0.0, 1.0);
            gl.clear(gl.COLOR_BUFFER_BIT);
            gl.readPixels(canvas.width / 2, canvas.height / 2, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, pixels);
            
            if (pixels[0] !== 255 || pixels[1] !== 0 || pixels[2] !== 0 || pixels[3] !== 255) {
                result.anomalyDetected = true;
                result.anomalyScore += 25;
                result.anomalyDetails.push('render_buffer_issue');
            }
            
            const buffer = gl.createBuffer();
            gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
            gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([1,2,3,4,5,6,7,8,9,10]), gl.STATIC_DRAW);
            
            const ext = gl.getExtension('WEBGL_debug_renderer_info');
            if (ext) {
                const renderer = gl.getParameter(ext.UNMASKED_RENDERER_WEBGL) || '';
                const rendererLower = renderer.toLowerCase();
                
                if (this.softwareRenderers.some(sw => rendererLower.includes(sw))) {
                    result.anomalyDetected = true;
                    result.anomalyScore += 30;
                    result.anomalyDetails.push('software_rendering_confirmed');
                }
            }
            
            gl.deleteBuffer(buffer);
            
        } catch (e) {
            result.anomalyDetected = true;
            result.anomalyScore += 35;
            result.anomalyDetails.push('render_test_exception');
        }
        
        return result;
    }
    
    /**
     * Canvas 指纹时序分析
     */
    async detectCanvasTiming() {
        let score = 0;
        const detections = [];
        
        try {
            const timingResults = [];
            const sampleCount = Math.min(this.options.maxTimingSamples, 10);
            
            for (let i = 0; i < sampleCount; i++) {
                const startTime = performance.now();
                const canvas = document.createElement('canvas');
                canvas.width = 280;
                canvas.height = 80;
                const ctx = canvas.getContext('2d');
                
                if (!ctx) {
                    score += 45;
                    detections.push('no_canvas_2d_context');
                    return { detected: true, score: Math.min(score, 100), detections };
                }
                
                ctx.textBaseline = 'alphabetic';
                ctx.fillStyle = '#f60';
                ctx.fillRect(125, 1, 62, 20);
                ctx.fillStyle = '#069';
                ctx.font = '11pt Arial';
                ctx.fillText('Cwm fjordbank glyphs vext quiz, 😀', 2, 15);
                
                ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
                ctx.font = '18pt Arial';
                ctx.fillText('Cwm fjordbank glyphs vext quiz, 😀', 4, 45);
                
                ctx.globalCompositeOperation = 'multiply';
                ctx.fillStyle = 'rgb(255,0,255)';
                ctx.beginPath();
                ctx.arc(50, 50, 50, 0, Math.PI * 2, true);
                ctx.closePath();
                ctx.fill();
                
                ctx.fillStyle = 'rgb(0,255,255)';
                ctx.beginPath();
                ctx.arc(100, 50, 50, 0, Math.PI * 2 / 3, true);
                ctx.closePath();
                ctx.fill();
                
                ctx.fillStyle = 'rgb(255,255,0)';
                ctx.beginPath();
                ctx.arc(75, 50, 50, 0, Math.PI * 2 / 3, false);
                ctx.closePath();
                ctx.fill();
                
                const dataURL = canvas.toDataURL();
                const elapsed = performance.now() - startTime;
                
                timingResults.push({
                    time: elapsed,
                    dataURL: dataURL,
                    sample: i
                });
            }
            
            const times = timingResults.map(r => r.time);
            const avgTime = times.reduce((a, b) => a + b, 0) / times.length;
            const variance = times.reduce((sum, t) => sum + Math.pow(t - avgTime, 2), 0) / times.length;
            const stdDev = Math.sqrt(variance);
            
            if (stdDev > avgTime * 0.5) {
                score += 30;
                detections.push('high_timing_variance');
            }
            
            if (avgTime < 1) {
                score += 25;
                detections.push('abnormally_fast_timing');
            }
            
            if (avgTime > 100) {
                score += 20;
                detections.push('abnormally_slow_timing');
            }
            
            const firstDataURL = timingResults[0].dataURL;
            const lastDataURL = timingResults[timingResults.length - 1].dataURL;
            
            if (firstDataURL !== lastDataURL) {
                score += 15;
                detections.push('canvas_inconsistent_output');
            }
            
            const firstImageData = this.getCanvasImageData(timingResults[0].dataURL);
            const secondImageData = this.getCanvasImageData(timingResults[1].dataURL);
            
            if (firstImageData && secondImageData) {
                const similarity = this.calculateImageSimilarity(firstImageData, secondImageData);
                if (similarity < 0.9) {
                    score += 25;
                    detections.push(`low_canvas_consistency:${similarity.toFixed(2)}`);
                }
            }
            
            const allDataURLs = timingResults.map(r => r.dataURL);
            const uniqueHashes = new Set(allDataURLs.map(url => this.hashString(url)));
            if (uniqueHashes.size < allDataURLs.length * 0.5) {
                score += 20;
                detections.push('canvas_low_entropy');
            }
            
        } catch (e) {
            score += 35;
            detections.push('canvas_timing_error');
        }
        
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }
    
    /**
     * Canvas 渲染一致性检测
     */
    async detectCanvasRenderConsistency() {
        let score = 0;
        const detections = [];
        
        try {
            const canvas1 = document.createElement('canvas');
            canvas1.width = 200;
            canvas1.height = 100;
            const ctx1 = canvas1.getContext('2d');
            
            const canvas2 = document.createElement('canvas');
            canvas2.width = 200;
            canvas2.height = 100;
            const ctx2 = canvas2.getContext('2d');
            
            if (!ctx1 || !ctx2) {
                score += 40;
                detections.push('no_consistency_context');
                return { detected: true, score: Math.min(score, 100), detections };
            }
            
            const pattern = () => {
                ctx1.fillStyle = '#ff0000';
                ctx1.fillRect(0, 0, 100, 50);
                ctx1.fillStyle = '#0000ff';
                ctx1.fillRect(100, 0, 100, 50);
                ctx1.fillStyle = '#00ff00';
                ctx1.fillRect(0, 50, 100, 50);
                ctx1.fillStyle = '#ffff00';
                ctx1.fillRect(100, 50, 100, 50);
            };
            
            pattern.apply(ctx1);
            pattern.apply(ctx2);
            
            const data1 = ctx1.getImageData(0, 0, 200, 100).data;
            const data2 = ctx2.getImageData(0, 0, 200, 100).data;
            
            let matchingPixels = 0;
            let totalPixels = data1.length / 4;
            
            for (let i = 0; i < data1.length; i += 4) {
                if (data1[i] === data2[i] && data1[i+1] === data2[i+1] && 
                    data1[i+2] === data2[i+2] && data1[i+3] === data2[i+3]) {
                    matchingPixels++;
                }
            }
            
            const consistency = matchingPixels / totalPixels;
            
            if (consistency < 0.95) {
                score += 30;
                detections.push(`low_render_consistency:${consistency.toFixed(2)}`);
            }
            
            const imageData = ctx1.getImageData(0, 0, 10, 10);
            const pixelSum = Array.from(imageData.data.slice(0, 40)).reduce((a, b) => a + b, 0);
            
            if (pixelSum === 0) {
                score += 35;
                detections.push('canvas_empty_readback');
            }
            
            const testCanvas = document.createElement('canvas');
            testCanvas.width = 100;
            testCanvas.height = 100;
            const testCtx = testCanvas.getContext('2d');
            
            testCtx.fillStyle = '#ff0000';
            testCtx.fillRect(0, 0, 50, 50);
            testCtx.fillStyle = '#0000ff';
            testCtx.fillRect(50, 0, 50, 50);
            
            const testData = testCtx.getImageData(0, 0, 100, 100).data;
            let nonBlackPixels = 0;
            
            for (let i = 0; i < testData.length; i += 4) {
                if (testData[i] !== 0 || testData[i+1] !== 0 || testData[i+2] !== 0) {
                    nonBlackPixels++;
                }
            }
            
            if (nonBlackPixels === 0) {
                score += 45;
                detections.push('canvas_fully_blocked');
            }
            
        } catch (e) {
            score += 30;
            detections.push('canvas_consistency_error');
        }
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }
    
    /**
     * 浏览器插件检测模块
     */
    async detectPlugins() {
        let score = 0;
        const detections = [];
        
        try {
            const plugins = navigator.plugins;
            
            if (!plugins || plugins.length === 0) {
                score += 35;
                detections.push('no_plugins_installed');
                return { detected: true, score: Math.min(score, 100), detections };
            }
            
            const pluginCount = plugins.length;
            if (pluginCount < 2) {
                score += 20;
                detections.push(`very_few_plugins:${pluginCount}`);
            }
            
            if (pluginCount > 20) {
                score += 15;
                detections.push(`excessive_plugins:${pluginCount}`);
            }
            
            const commonPlugins = ['pdf', 'flash', 'silverlight', 'java', 'shockwave'];
            const foundCommon = commonPlugins.filter(cp => 
                Array.from(plugins).some(p => p.name.toLowerCase().includes(cp))
            );
            
            if (foundCommon.length === 0) {
                score += 20;
                detections.push('no_common_plugins');
            }
            
            const pluginNames = Array.from(plugins).map(p => p.name.toLowerCase());
            
            const suspiciousPlugins = ['honey', 'privacy', 'adblock', 'ublock', 'noscript', 'ghostery'];
            const foundSuspicious = suspiciousPlugins.filter(sp => 
                pluginNames.some(pn => pn.includes(sp))
            );
            
            if (foundSuspicious.length > 0) {
                score += 10;
                detections.push(`privacy_plugins:${foundSuspicious.length}`);
            }
            
            const automationPlugins = ['selenium', 'webdriver', 'automate', 'bot'];
            const foundAutomation = automationPlugins.filter(ap => 
                pluginNames.some(pn => pn.includes(ap))
            );
            
            if (foundAutomation.length > 0) {
                score += 40;
                detections.push(`automation_plugins:${foundAutomation.join(',')}`);
            }
            
            const rarePlugins = pluginNames.filter(pn => {
                return !pn.includes('pdf') && !pn.includes('flash') && 
                       !pn.includes('media') && !pn.includes('viewer');
            });
            
            if (rarePlugins.length > 5) {
                score += 15;
                detections.push(`rare_plugins:${rarePlugins.length}`);
            }
            
            const pluginDescriptions = Array.from(plugins)
                .filter(p => p.description)
                .map(p => p.description.toLowerCase());
            
            const suspiciousDescriptions = ['automation', 'testing', 'bot', 'crawler', 'scraper'];
            const foundSuspiciousDesc = suspiciousDescriptions.filter(sd => 
                pluginDescriptions.some(pd => pd.includes(sd))
            );
            
            if (foundSuspiciousDesc.length > 0) {
                score += 35;
                detections.push('suspicious_plugin_description');
            }
            
            const mimeTypes = navigator.mimeTypes || [];
            if (mimeTypes.length === 0 && pluginCount > 0) {
                score += 25;
                detections.push('no_mime_types');
            }
            
        } catch (e) {
            score += 30;
            detections.push('plugin_detection_error');
        }
        
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }
    
    /**
     * 虚拟机多维度检测 - CPU
     */
    async detectVMCPU() {
        let score = 0;
        const detections = [];
        
        try {
            const cpuCount = navigator.hardwareConcurrency;
            
            if (cpuCount === undefined || cpuCount === null) {
                score += 30;
                detections.push('cpu_count_unavailable');
            } else if (cpuCount === 1) {
                score += 35;
                detections.push('single_core_cpu_vm');
            } else if (cpuCount === 2) {
                score += 20;
                detections.push('dual_core_cpu_vm');
            } else if (cpuCount > 64) {
                score += 15;
                detections.push(`unrealistic_cpu_count:${cpuCount}`);
            } else if (cpuCount >= 4 && cpuCount <= 8) {
                score += 10;
                detections.push(`typical_vm_cpu:${cpuCount}`);
            }
            
            const cpuBrand = this.getCPUBrand();
            if (cpuBrand) {
                const brandLower = cpuBrand.toLowerCase();
                const vmCpuPatterns = ['virtual', 'vmware', 'virtualbox', 'qemu', 'kvm', 'hyperv'];
                for (const pattern of vmCpuPatterns) {
                    if (brandLower.includes(pattern)) {
                        score += 45;
                        detections.push(`vm_cpu_brand:${pattern}`);
                    }
                }
            }
            
            if (navigator.deviceMemory !== undefined) {
                const mem = navigator.deviceMemory;
                if (mem <= 0.5) {
                    score += 30;
                    detections.push('minimal_device_memory');
                } else if (mem > 64) {
                    score += 10;
                    detections.push(`unrealistic_memory:${mem}gb`);
                }
            }
            
        } catch (e) {
            score += 25;
            detections.push('vm_cpu_detection_error');
        }
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }
    
    /**
     * 虚拟机多维度检测 - GPU
     */
    async detectVMGPU() {
        let score = 0;
        const detections = [];
        
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            
            if (gl) {
                const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                if (debugInfo) {
                    const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) || '';
                    const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) || '';
                    
                    const vmGpuPatterns = {
                        'vmware': ['vmware', 'svga', 'virtual'],
                        'virtualbox': ['virtualbox', 'vbox'],
                        'parallels': ['parallels'],
                        'qemu': ['qemu', 'virtio'],
                        'hyperv': ['hyper-v', 'microsoft basic']
                    };
                    
                    const rendererLower = renderer.toLowerCase();
                    const vendorLower = vendor.toLowerCase();
                    
                    for (const [vmType, patterns] of Object.entries(vmGpuPatterns)) {
                        for (const pattern of patterns) {
                            if (rendererLower.includes(pattern) || vendorLower.includes(pattern)) {
                                score += 50;
                                detections.push(`vm_gpu_detected:${vmType}`);
                            }
                        }
                    }
                    
                    if (this.softwareRenderers.some(sw => rendererLower.includes(sw))) {
                        score += 40;
                        detections.push('software_rendering_gpu');
                    }
                    
                    const capabilities = {
                        maxTextureSize: gl.getParameter(gl.MAX_TEXTURE_SIZE),
                        maxRenderbufferSize: gl.getParameter(gl.MAX_RENDERBUFFER_SIZE),
                        maxVertexAttribs: gl.getParameter(gl.MAX_VERTEX_ATTRIBS)
                    };
                    
                    if (capabilities.maxTextureSize < 4096 || capabilities.maxRenderbufferSize < 4096) {
                        score += 25;
                        detections.push('limited_gpu_capabilities');
                    }
                }
            }
            
            if (window.WebGL2RenderingContext) {
                const gl2 = canvas.getContext('webgl2');
                if (gl2) {
                    const max3DTextureSize = gl2.getParameter(gl2.MAX_3D_TEXTURE_SIZE);
                    if (max3DTextureSize < 512) {
                        score += 20;
                        detections.push(`limited_3d_texture_support:${max3DTextureSize}`);
                    }
                }
            }
            
        } catch (e) {
            score += 25;
            detections.push('vm_gpu_detection_error');
        }
        
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }
    
    /**
     * 虚拟机多维度检测 - 进程和系统
     */
    async detectVMProcess() {
        let score = 0;
        const detections = [];
        
        try {
            const ua = navigator.userAgent.toLowerCase();
            
            const processPatterns = [
                { pattern: 'vmware', name: 'vmware_detected' },
                { pattern: 'virtualbox', name: 'virtualbox_detected' },
                { pattern: 'parallels', name: 'parallels_detected' },
                { pattern: 'xen', name: 'xen_detected' },
                { pattern: 'qemu', name: 'qemu_detected' },
                { pattern: 'kvm', name: 'kvm_detected' },
                { pattern: 'hyperv', name: 'hyperv_detected' },
                { pattern: 'openvz', name: 'openvz_detected' },
                { pattern: 'docker', name: 'container_detected' }
            ];
            
            for (const {pattern, name} of processPatterns) {
                if (ua.includes(pattern)) {
                    score += 40;
                    detections.push(name);
                }
            }
            
            const platform = navigator.platform?.toLowerCase() || '';
            for (const {pattern, name} of processPatterns) {
                if (platform.includes(pattern)) {
                    score += 35;
                    detections.push(`${name}_platform`);
                }
            }
            
            try {
                const testDiv = document.createElement('div');
                testDiv.style.cssText = 'position:absolute;left:-9999px;top:-9999px;';
                document.body.appendChild(testDiv);
                
                const computedStyle = window.getComputedStyle(testDiv);
                if (computedStyle.display === 'none') {
                    score += 5;
                    detections.push('hidden_element_accessible');
                }
                
                document.body.removeChild(testDiv);
            } catch (e) {}
            
            const biosPatterns = [
                { pattern: 'virtualbox', name: 'vbox_bios' },
                { pattern: 'vmware', name: 'vmware_bios' },
                { pattern: 'parallels', name: 'parallels_bios' },
                { pattern: 'qemu', name: 'qemu_bios' },
                { pattern: 'bochs', name: 'bochs_bios' }
            ];
            
            for (const {pattern, name} of biosPatterns) {
                if (ua.includes(pattern)) {
                    score += 30;
                    detections.push(name);
                }
            }
            
            if (screen && screen.width && screen.height) {
                const { width, height } = screen;
                
                const commonVmResolutions = [
                    [800, 600], [1024, 768], [1280, 720], [1366, 768],
                    [1440, 900], [1600, 900], [1920, 1080]
                ];
                
                const isCommonVmRes = commonVmResolutions.some(
                    ([w, h]) => w === width && h === height
                );
                
                if (isCommonVmRes) {
                    score += 15;
                    detections.push(`common_vm_resolution:${width}x${height}`);
                }
                
                if (width === 0 || height === 0) {
                    score += 40;
                    detections.push('zero_screen_resolution');
                }
            }
            
            if (window.outerWidth !== undefined && window.outerHeight !== undefined) {
                if (window.outerWidth === 0 || window.outerHeight === 0) {
                    score += 35;
                    detections.push('zero_window_size');
                }
                
                if (window.outerWidth < 100 || window.outerHeight < 100) {
                    score += 25;
                    detections.push('abnormally_small_window');
                }
            }
            
        } catch (e) {
            score += 25;
            detections.push('vm_process_detection_error');
        }
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }
    
    /**
     * 虚拟机多维度检测 - 内存
     */
    async detectVMMemory() {
        let score = 0;
        const detections = [];
        
        try {
            if (navigator.deviceMemory !== undefined) {
                const mem = navigator.deviceMemory;
                
                if (mem <= 0.25) {
                    score += 35;
                    detections.push(`minimal_memory:${mem}gb`);
                } else if (mem <= 0.5) {
                    score += 25;
                    detections.push(`low_memory:${mem}gb`);
                } else if (mem >= 32) {
                    score += 10;
                    detections.push(`high_memory:${mem}gb`);
                }
                
                if (mem === 2 || mem === 4) {
                    score += 15;
                    detections.push(`typical_vm_memory:${mem}gb`);
                }
            }
            
            try {
                if (navigator.storage && navigator.storage.estimate) {
                    const estimate = await navigator.storage.estimate();
                    
                    if (estimate.quota === 0) {
                        score += 30;
                        detections.push('zero_storage_quota');
                    }
                    
                    if (estimate.quota && estimate.quota < 100000000) {
                        score += 25;
                        detections.push(`low_storage_quota:${estimate.quota}`);
                    }
                    
                    if (estimate.usage === 0 && estimate.quota > 0) {
                        score += 15;
                        detections.push('no_storage_used');
                    }
                }
            } catch (e) {
                score += 15;
                detections.push('storage_api_unavailable');
            }
            
            try {
                const canvas = document.createElement('canvas');
                canvas.width = 100;
                canvas.height = 100;
                const ctx = canvas.getContext('2d');
                
                if (ctx) {
                    ctx.fillStyle = '#ff0000';
                    ctx.fillRect(0, 0, 50, 50);
                    ctx.fillStyle = '#0000ff';
                    ctx.fillRect(50, 0, 50, 50);
                    
                    const data = ctx.getImageData(0, 0, 100, 100).data;
                    let nonZero = 0;
                    
                    for (let i = 0; i < data.length; i += 4) {
                        if (data[i] !== 0 || data[i+1] !== 0 || data[i+2] !== 0) {
                            nonZero++;
                        }
                    }
                    
                    if (nonZero === 0) {
                        score += 40;
                        detections.push('canvas_memory_issue');
                    }
                }
            } catch (e) {}
            
        } catch (e) {
            score += 25;
            detections.push('vm_memory_detection_error');
        }
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }
    
    /**
     * Tor 网络检测
     */
    async detectTorNetwork() {
        let score = 0;
        const detections = [];
        
        try {
            const ua = navigator.userAgent || '';
            if (/tor|onion/i.test(ua)) {
                score += 60;
                detections.push('tor_user_agent');
            }
            
            try {
                const response = await fetch('https://check.torproject.org/api/ip', {
                    method: 'GET',
                    headers: { 'Accept': 'application/json' }
                }).catch(() => null);
                
                if (response && response.ok) {
                    const data = await response.json();
                    
                    if (data.IsTor !== undefined) {
                        if (data.IsTor === true) {
                            score += 70;
                            detections.push('tor_network_confirmed');
                        }
                    }
                    
                    if (data.IP) {
                        this.lastKnownIP = data.IP;
                    }
                }
            } catch (e) {
                score += 15;
                detections.push('tor_api_check_failed');
            }
            
            try {
                const pc = new RTCPeerConnection({
                    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
                });
                pc.createDataChannel('');
                
                const offer = await pc.createOffer();
                await pc.setLocalDescription(offer);
                const sdp = pc.localDescription.sdp;
                
                if (/tls|inject_host_overwrite/i.test(sdp)) {
                    score += 50;
                    detections.push('tor_sdp_signature');
                }
                
                if (/candidate.*tcp|tcptype/i.test(sdp)) {
                    score += 35;
                    detections.push('tor_tcp_candidate');
                }
                
                pc.close();
            } catch (e) {}
            
            if (navigator.connection) {
                const conn = navigator.connection;
                if (conn.rtt && conn.rtt > 500) {
                    score += 30;
                    detections.push('high_round_trip_time');
                }
            }
            
        } catch (e) {
            score += 25;
            detections.push('tor_detection_error');
        }
        
        return { detected: score > 45, score: Math.min(score, 100), detections };
    }
    
    /**
     * Tor 出口节点检测
     */
    async detectTorExitNode() {
        let score = 0;
        const detections = [];
        
        try {
            const exitNodePatterns = [
                'tor', 'exit', 'onion', 'anonymizer', 'torservers',
                'tornode', 'torproject', 'tor-exit'
            ];
            
            const ua = navigator.userAgent || '';
            const platform = navigator.platform || '';
            
            for (const pattern of exitNodePatterns) {
                if (ua.toLowerCase().includes(pattern) || platform.toLowerCase().includes(pattern)) {
                    score += 50;
                    detections.push(`exit_node_pattern:${pattern}`);
                }
            }
            
            try {
                const response = await fetch('https://ipinfo.io/json', {
                    method: 'GET',
                    headers: { 'Accept': 'application/json' }
                }).catch(() => null);
                
                if (response && response.ok) {
                    const data = await response.json();
                    
                    if (data.hosting === true || data.hosting === 'true') {
                        score += 35;
                        detections.push('hosting_detected');
                    }
                    
                    if (data.proxy === true || data.proxy === 'true') {
                        score += 40;
                        detections.push('proxy_detected');
                    }
                    
                    const org = (data.org || '').toLowerCase();
                    const isp = (data.isp || '').toLowerCase();
                    const asn = (data.asn || '').toLowerCase();
                    
                    const torIndicators = ['tor', 'onion', 'exitnode', 'anonymizer'];
                    for (const indicator of torIndicators) {
                        if (org.includes(indicator) || isp.includes(indicator) || asn.includes(indicator)) {
                            score += 55;
                            detections.push(`tor_isp_indicator:${indicator}`);
                        }
                    }
                    
                    if (data.country) {
                        this.lastKnownCountry = data.country;
                    }
                }
            } catch (e) {}
            
            try {
                const latencyStart = performance.now();
                await fetch('/api/v1/health', { 
                    method: 'HEAD',
                    mode: 'no-cors',
                    cache: 'no-cache'
                }).catch(() => null);
                const latency = performance.now() - latencyStart;
                
                if (latency > 2000) {
                    score += 25;
                    detections.push(`high_latency:${Math.round(latency)}ms`);
                }
            } catch (e) {}
            
        } catch (e) {
            score += 20;
            detections.push('tor_exit_detection_error');
        }
        
        return { detected: score > 40, score: Math.min(score, 100), detections };
    }
    
    /**
     * 暗网出口节点检测
     */
    async detectDarkWebIndicators() {
        let score = 0;
        const detections = [];
        
        try {
            const ua = navigator.userAgent || '';
            const darkWebPatterns = [
                'dark web', 'tor browser', 'onion', '暗网', 'ダークウェブ'
            ];
            
            for (const pattern of darkWebPatterns) {
                if (ua.toLowerCase().includes(pattern.toLowerCase())) {
                    score += 55;
                    detections.push(`dark_web_indicator:${pattern}`);
                }
            }
            
            try {
                const pc = new RTCPeerConnection({
                    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
                });
                pc.createDataChannel('');
                
                const offer = await pc.createOffer();
                await pc.setLocalDescription(offer);
                const sdp = pc.localDescription.sdp;
                
                const lines = sdp.split('\n');
                let relayCount = 0;
                
                for (const line of lines) {
                    if (line.includes('candidate') && /relay|prflx/i.test(line)) {
                        relayCount++;
                    }
                }
                
                if (relayCount > 2) {
                    score += 40;
                    detections.push(`many_relay_candidates:${relayCount}`);
                }
                
                pc.close();
            } catch (e) {}
            
            const networkTypes = ['vpn', 'tor', 'proxy', 'datacenter'];
            if (navigator.connection && navigator.connection.type) {
                const connType = navigator.connection.type.toLowerCase();
                if (networkTypes.some(nt => connType.includes(nt))) {
                    score += 35;
                    detections.push(`suspicious_network_type:${navigator.connection.type}`);
                }
            }
            
        } catch (e) {
            score += 20;
            detections.push('dark_web_detection_error');
        }
        
        return { detected: score > 45, score: Math.min(score, 100), detections };
    }
    
    /**
     * 运行所有检测
     */
    async runAll() {
        const startTime = performance.now();
        
        const detections = [];
        
        if (this.options.enableWebGLAnalysis) {
            const webglAnomaly = await this.detectWebGLAnomaly();
            this.results.webgl_anomaly = webglAnomaly;
            detections.push(...webglAnomaly.detections);
            
            const webgl2Rendering = await this.detectWebGL2Rendering();
            this.results.webgl_rendering = webgl2Rendering;
            detections.push(...webgl2Rendering.detections);
        }
        
        if (this.options.enableCanvasTiming) {
            const canvasTiming = await this.detectCanvasTiming();
            this.results.canvas_timing = canvasTiming;
            detections.push(...canvasTiming.detections);
            
            const canvasConsistency = await this.detectCanvasRenderConsistency();
            this.results.canvas_entropy = canvasConsistency;
            detections.push(...canvasConsistency.detections);
        }
        
        if (this.options.enablePluginDetection) {
            const plugins = await this.detectPlugins();
            this.results.plugin_detection = plugins;
            detections.push(...plugins.detections);
        }
        
        if (this.options.enableVMDetection) {
            const vmCPU = await this.detectVMCPU();
            this.results.vm_cpu = vmCPU;
            detections.push(...vmCPU.detections);
            
            const vmGPU = await this.detectVMGPU();
            this.results.vm_gpu = vmGPU;
            detections.push(...vmGPU.detections);
            
            const vmProcess = await this.detectVMProcess();
            this.results.vm_process = vmProcess;
            detections.push(...vmProcess.detections);
            
            const vmMemory = await this.detectVMMemory();
            this.results.vm_memory = vmMemory;
            detections.push(...vmMemory.detections);
        }
        
        if (this.options.enableTorCheck) {
            const torNetwork = await this.detectTorNetwork();
            this.results.tor_detected = torNetwork;
            detections.push(...torNetwork.detections);
            
            const torExit = await this.detectTorExitNode();
            this.results.tor_exit_node = torExit;
            detections.push(...torExit.detections);
            
            const darkWeb = await this.detectDarkWebIndicators();
            this.results.dark_web = darkWeb;
            detections.push(...darkWeb.detections);
        }
        
        this.riskScore = this.calculateOverallRiskScore();
        const duration = performance.now() - startTime;
        
        return {
            detection_id: this.detectionId,
            results: this.results,
            risk_score: this.riskScore,
            risk_level: this.getRiskLevel(),
            all_detections: detections,
            timestamp: Date.now(),
            duration_ms: Math.round(duration),
            summary: this.generateSummary()
        };
    }
    
    /**
     * 计算综合风险评分
     */
    calculateOverallRiskScore() {
        let weightedScore = 0;
        let totalWeight = 0;
        
        for (const [key, result] of Object.entries(this.results)) {
            if (result && typeof result.score === 'number') {
                const weight = this.weights[key] || 5;
                weightedScore += result.score * weight;
                totalWeight += weight;
            }
        }
        
        if (totalWeight === 0) return 0;
        
        let baseScore = weightedScore / totalWeight;
        
        const highRiskDetections = Object.values(this.results)
            .filter(r => r && r.score > 50);
        
        if (highRiskDetections.length >= 3) {
            baseScore = Math.min(baseScore * 1.5 + 25, 100);
        } else if (highRiskDetections.length >= 2) {
            baseScore = Math.min(baseScore * 1.3 + 15, 100);
        } else if (highRiskDetections.length >= 1) {
            baseScore = Math.min(baseScore * 1.15 + 8, 100);
        }
        
        const torRelated = ['tor_detected', 'tor_exit_node', 'dark_web'];
        const torDetected = torRelated.filter(k => 
            this.results[k] && this.results[k].score > 30
        );
        if (torDetected.length > 0) {
            baseScore = Math.min(baseScore * 1.3 + 15, 100);
        }
        
        const vmRelated = ['vm_cpu', 'vm_gpu', 'vm_process', 'vm_memory'];
        const vmDetected = vmRelated.filter(k => 
            this.results[k] && this.results[k].score > 30
        );
        if (vmDetected.length >= 2) {
            baseScore = Math.min(baseScore * 1.4 + 18, 100);
        } else if (vmDetected.length >= 1) {
            baseScore = Math.min(baseScore * 1.2 + 10, 100);
        }
        
        return Math.round(Math.min(Math.max(baseScore, 0), 100));
    }
    
    /**
     * 获取风险等级
     */
    getRiskLevel() {
        if (this.riskScore >= 70) return 'high';
        if (this.riskScore >= 40) return 'medium';
        return 'low';
    }
    
    /**
     * 生成检测摘要
     */
    generateSummary() {
        const summary = {
            total_checks: Object.keys(this.results).length,
            high_risk_checks: 0,
            medium_risk_checks: 0,
            low_risk_checks: 0,
            categories: {
                webgl: { score: 0, detections: [] },
                canvas: { score: 0, detections: [] },
                plugins: { score: 0, detections: [] },
                vm: { score: 0, detections: [] },
                tor: { score: 0, detections: [] }
            }
        };
        
        const categoryMap = {
            webgl_anomaly: 'webgl',
            webgl_rendering: 'webgl',
            canvas_timing: 'canvas',
            canvas_entropy: 'canvas',
            plugin_detection: 'plugins',
            vm_cpu: 'vm',
            vm_gpu: 'vm',
            vm_process: 'vm',
            vm_memory: 'vm',
            tor_detected: 'tor',
            tor_exit_node: 'tor',
            dark_web: 'tor'
        };
        
        for (const [key, result] of Object.entries(this.results)) {
            if (result && typeof result.score === 'number') {
                const category = categoryMap[key] || 'other';
                
                summary.categories[category].score = Math.max(
                    summary.categories[category].score,
                    result.score
                );
                summary.categories[category].detections.push(...result.detections);
                
                if (result.score > 50) summary.high_risk_checks++;
                else if (result.score > 25) summary.medium_risk_checks++;
                else summary.low_risk_checks++;
            }
        }
        
        return summary;
    }
    
    /**
     * 获取 CPU 品牌信息
     */
    getCPUBrand() {
        try {
            if (navigator.hardwareConcurrency && navigator.deviceMemory) {
                return 'Virtual CPU';
            }
        } catch (e) {}
        return null;
    }
    
    /**
     * 获取 Canvas 图像数据
     */
    getCanvasImageData(dataURL) {
        try {
            return dataURL.split(',')[1] || '';
        } catch (e) {
            return null;
        }
    }
    
    /**
     * 计算图像相似度
     */
    calculateImageSimilarity(data1, data2) {
        if (!data1 || !data2) return 0;
        
        const hash1 = this.hashString(data1);
        const hash2 = this.hashString(data2);
        
        const maxLen = Math.max(hash1.length, hash2.length);
        const minLen = Math.min(hash1.length, hash2.length);
        
        let matches = 0;
        for (let i = 0; i < minLen; i++) {
            if (hash1[i] === hash2[i]) matches++;
        }
        
        return matches / maxLen;
    }
    
    /**
     * 字符串哈希
     */
    hashString(str) {
        let hash = 0;
        for (let i = 0; i < Math.min(str.length, 1000); i++) {
            const char = str.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash;
        }
        return Math.abs(hash).toString(16);
    }
    
    /**
     * 发送到后端进行验证
     */
    async sendToBackend() {
        const detectionResult = await this.runAll();
        
        try {
            const response = await fetch(`${this.options.apiBase}/environment/detect`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    detection_id: detectionResult.detection_id,
                    risk_score: detectionResult.risk_score,
                    risk_level: detectionResult.risk_level,
                    all_detections: detectionResult.all_detections,
                    timestamp: detectionResult.timestamp,
                    client_results: detectionResult.results,
                    summary: detectionResult.summary
                })
            });
            
            if (response.ok) {
                return await response.json();
            }
            
            return { success: false, error: 'Server error' };
        } catch (e) {
            return { success: false, error: e.message };
        }
    }
    
    /**
     * 检查 Tor 网络（独立接口）
     */
    async checkTorNetwork() {
        try {
            const response = await fetch(`${this.options.apiBase}/environment/tor-check`, {
                method: 'GET',
                headers: {
                    'Accept': 'application/json'
                }
            });
            
            if (response.ok) {
                return await response.json();
            }
            
            return { success: false, error: 'Server error' };
        } catch (e) {
            return { success: false, error: e.message };
        }
    }
    
    /**
     * 转换为 JSON
     */
    toJSON() {
        return {
            detection_id: this.detectionId,
            risk_score: this.riskScore,
            risk_level: this.getRiskLevel(),
            results: this.results,
            summary: this.generateSummary()
        };
    }
}

/**
 * 导出检测器
 */
if (typeof window !== 'undefined') {
    window.AdvancedEnvironmentDetector = AdvancedEnvironmentDetector;
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = AdvancedEnvironmentDetector;
}
