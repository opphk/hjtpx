(function(globalContext) {
    'use strict';

    const AdvancedObfuscator = (function() {
        const VERSION = '4.0.0';
        const CHAR_POOL = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_$';
        const HEX_POOL = '0123456789abcdef';
        
        let config = {
            enableVariableObfuscation: true,
            enableStringEncryption: true,
            enableControlFlowFlattening: true,
            enableDeadCodeInjection: true,
            enableFunctionWrapping: true,
            enableNumberObfuscation: true,
            enableBooleanObfuscation: true,
            enablePropertyAccessObfuscation: true,
            enableUnicodeObfuscation: true,
            stringEncryptionKey: 'hjtpx-obfuscate-key-2024',
            obfuscationLevel: 3,
            shuffleStrings: true,
            addOpaquePredicates: true,
            controlFlowIterations: 3
        };

        let variableMap = new Map();
        let stringMap = new Map();
        let functionMap = new Map();
        let usedNames = new Set();
        let identifierCounter = 0;
        let stringCounter = 0;

        function setConfig(cfg) {
            Object.assign(config, cfg);
        }

        function generateObfuscatedName() {
            const prefixes = ['_0x', '_$', '__'];
            let name;
            
            do {
                const prefix = prefixes[identifierCounter % prefixes.length];
                const length = 4 + (identifierCounter % 5);
                let chars = '';
                for (let i = 0; i < length; i++) {
                    chars += CHAR_POOL[Math.floor(Math.random() * CHAR_POOL.length)];
                }
                name = prefix + chars;
                identifierCounter++;
            } while (usedNames.has(name));
            
            usedNames.add(name);
            return name;
        }

        function obfuscateVariables(code) {
            if (!config.enableVariableObfuscation) return code;

            const varPattern = /\b(var|let|const)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\s*=/g;
            let result = code;

            result = result.replace(varPattern, (match, declType, name) => {
                if (isReserved(name)) return match;
                
                const newName = generateObfuscatedName();
                variableMap.set(name, newName);
                return declType + ' ' + newName + '=';
            });

            for (const [original, obfuscated] of variableMap) {
                const pattern = new RegExp('\\b' + escapeRegex(original) + '\\b', 'g');
                result = result.replace(pattern, obfuscated);
            }

            return result;
        }

        function isReserved(name) {
            const reserved = new Set([
                'if', 'else', 'for', 'while', 'do', 'switch', 'case', 'default',
                'break', 'continue', 'return', 'try', 'catch', 'finally', 'throw',
                'new', 'delete', 'typeof', 'instanceof', 'void', 'this', 'super',
                'class', 'extends', 'static', 'get', 'set', 'import', 'export',
                'from', 'as', 'const', 'let', 'var', 'function', 'async', 'await',
                'yield', 'true', 'false', 'null', 'undefined', 'NaN', 'Infinity',
                'arguments', 'eval', 'constructor', 'prototype', 'toString',
                'valueOf', 'hasOwnProperty', 'console', 'window', 'document',
                'Array', 'Object', 'String', 'Number', 'Boolean', 'Function',
                'Promise', 'Map', 'Set', 'JSON', 'Math', 'Date', 'RegExp',
                'Error', 'TypeError', 'RangeError', 'SyntaxError', 'ReferenceError'
            ]);
            return reserved.has(name);
        }

        function escapeRegex(str) {
            return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        }

        function encryptString(str) {
            if (!config.enableStringEncryption || str.length < 3) return str;

            const key = config.stringEncryptionKey;
            let encrypted = '';
            
            for (let i = 0; i < str.length; i++) {
                const charCode = str.charCodeAt(i);
                const keyChar = key.charCodeAt(i % key.length);
                const encryptedCode = charCode ^ keyChar;
                encrypted += String.fromCharCode(encryptedCode);
            }

            return btoa(encrypted);
        }

        function obfuscateStrings(code) {
            if (!config.enableStringEncryption) return code;

            const stringPattern = /(["'`])([^\\]|\\.)*?\1/g;
            let result = code;

            result = result.replace(stringPattern, (match) => {
                const quote = match[0];
                const content = match.slice(1, -1);
                
                if (content.length < 3) return match;
                
                stringCounter++;
                const encrypted = encryptString(content);
                stringMap.set(content, encrypted);
                
                return quote + 'window.__decrypt(' + quote + encrypted + quote + ')' + quote;
            });

            return result;
        }

        function flattenControlFlow(code) {
            if (!config.enableControlFlowFlattening) return code;

            let result = code;
            
            result = flattenIfStatements(result);
            result = flattenLoops(result);
            result = addStateMachine(result);

            return result;
        }

        function flattenIfStatements(code) {
            const ifPattern = /\bif\s*\(([^)]+)\)\s*\{([^}]+)\}\s*else\s*\{([^}]+)\}/g;
            
            return code.replace(ifPattern, (match, condition, ifBody, elseBody) => {
                const stateVar = generateObfuscatedName();
                const tempVar = generateObfuscatedName();
                
                return `(function(){var ${stateVar}=0,${tempVar};if(${condition}){${stateVar}=1;}else{${stateVar}=2;}switch(${stateVar}){case 1:${ifBody};break;case 2:${elseBody};break;}})()`;
            });
        }

        function flattenLoops(code) {
            const forPattern = /\bfor\s*\(([^)]+)\)\s*\{([^}]+)\}/g;
            
            return code.replace(forPattern, (match, init, body) => {
                const stateVar = generateObfuscatedName();
                const loopVar = generateObfuscatedName();
                
                return `(function(){var ${stateVar}=0,${loopVar};${init};for(;;){switch(${stateVar}){case 0:if(!(${loopVar})){${stateVar}=1;break;}${body};${stateVar}=0;continue;case 1:return;default:return;}}})()`;
            });
        }

        function addStateMachine(code) {
            const machineCode = `(function(){var _0xSM={s:0,t:[function(){this.s=1;},function(){this.s=2;},function(){this.s=0;}],r:function(){this.t[this.s].call(this);}};_0xSM.r();})();`;
            return machineCode + code;
        }

        function injectDeadCode(code) {
            if (!config.enableDeadCodeInjection) return code;

            const deadCode = generateDeadCode();
            return '(function(){' + deadCode + '})();' + code;
        }

        function generateDeadCode() {
            let code = '';
            
            for (let i = 0; i < 5; i++) {
                const varName = generateObfuscatedName();
                code += `var ${varName}=Math.random();`;
                code += `if(${varName}<0){console.log('${generateRandomString(8)}');}`;
            }

            code += 'var _0xD=function(_0xP){return _0xP*Math.random();};';
            code += 'var _0xDD=_0xD(' + Date.now() + ');';
            code += 'if(_0xDD<0){eval("' + generateRandomString(16) + '");}';

            return code;
        }

        function generateRandomString(length) {
            let str = '';
            for (let i = 0; i < length; i++) {
                str += CHAR_POOL[Math.floor(Math.random() * CHAR_POOL.length)];
            }
            return str;
        }

        function obfuscateNumbers(code) {
            if (!config.enableNumberObfuscation) return code;

            const numberPattern = /\b(\d+)\b/g;
            
            return code.replace(numberPattern, (match, num) => {
                const n = parseInt(num);
                if (n < 10) return match;
                
                const methods = [
                    () => `(${n ^ 0xFFFFFFFF}>>>0)`,
                    () => `(${n}.toString(2),${n})`,
                    () => `(${n}+0x0)`,
                    () => `(0x${n.toString(16)})`,
                    () => `(${n}*1)`
                ];
                
                return methods[Math.floor(Math.random() * methods.length)]();
            });
        }

        function obfuscateBooleans(code) {
            if (!config.enableBooleanObfuscation) return code;

            code = code.replace(/\btrue\b/g, '(!!1)');
            code = code.replace(/\bfalse\b/g, '(!!0)');

            return code;
        }

        function obfuscatePropertyAccess(code) {
            if (!config.enablePropertyAccessObfuscation) return code;

            const dotPattern = /([a-zA-Z_$][a-zA-Z0-9_$]*)\.([a-zA-Z_$][a-zA-Z0-9_$]*)/g;
            
            return code.replace(dotPattern, (match, obj, prop) => {
                if (isReserved(obj) || isReserved(prop)) return match;
                return `${obj}['${prop}']`;
            });
        }

        function obfuscateUnicode(code) {
            if (!config.enableUnicodeObfuscation) return code;

            const charPattern = /[a-zA-Z]/g;
            
            return code.replace(charPattern, (match) => {
                if (Math.random() > 0.7) {
                    return '\\u' + match.charCodeAt(0).toString(16).padStart(4, '0');
                }
                return match;
            });
        }

        function addOpaquePredicates(code) {
            if (!config.addOpaquePredicates) return code;

            const predicate = `(function(){var _0xOP=function(){var _0xP1=Math.random();var _0xP2=Math.random();return _0xP1*_0xP2>0.25&&_0xP1<0.7;};if(_0xOP()){}})();`;
            return predicate + code;
        }

        function wrapFunctions(code) {
            if (!config.enableFunctionWrapping) return code;

            const funcPattern = /\bfunction\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\s*\(/g;
            
            return code.replace(funcPattern, (match, name) => {
                if (isReserved(name)) return match;
                
                const wrapperName = generateObfuscatedName();
                functionMap.set(name, wrapperName);
                
                return `function ${wrapperName}(`;
            });
        }

        function generateDecoderFunction() {
            const key = btoa(config.stringEncryptionKey);
            
            return `window.__decrypt=function(_0xE){var _0xK=atob('${key}');var _0xC=atob(_0xE);var _0xO='';for(var _0xI=0;_0xI<_0xC.length;_0xI++){_0xO+=String.fromCharCode(_0xC.charCodeAt(_0xI)^_0xK.charCodeAt(_0xI%_0xK.length));}return _0xO;};`;
        }

        function compressCode(code) {
            code = code.replace(/\s+/g, ' ');
            code = code.replace(/\s*([{};,:])\s*/g, '$1');
            code = code.replace(/\{\s*/g, '{');
            code = code.replace(/\s*\}/g, '}');
            code = code.replace(/\n\s*\n/g, '\n');
            
            return code.trim();
        }

        function obfuscate(code) {
            variableMap.clear();
            stringMap.clear();
            functionMap.clear();
            usedNames.clear();
            identifierCounter = 0;
            stringCounter = 0;

            let result = code;

            if (config.enableDeadCodeInjection) {
                result = injectDeadCode(result);
            }

            if (config.enableVariableObfuscation) {
                result = obfuscateVariables(result);
            }

            if (config.enableStringEncryption) {
                result = obfuscateStrings(result);
            }

            if (config.enableControlFlowFlattening) {
                result = flattenControlFlow(result);
            }

            if (config.enableNumberObfuscation) {
                result = obfuscateNumbers(result);
            }

            if (config.enableBooleanObfuscation) {
                result = obfuscateBooleans(result);
            }

            if (config.enablePropertyAccessObfuscation) {
                result = obfuscatePropertyAccess(result);
            }

            if (config.enableUnicodeObfuscation) {
                result = obfuscateUnicode(result);
            }

            if (config.addOpaquePredicates) {
                result = addOpaquePredicates(result);
            }

            if (config.enableFunctionWrapping) {
                result = wrapFunctions(result);
            }

            if (config.enableStringEncryption) {
                result = generateDecoderFunction() + result;
            }

            result = compressCode(result);

            return result;
        }

        function getStats() {
            return {
                variablesObfuscated: variableMap.size,
                stringsEncrypted: stringMap.size,
                functionsWrapped: functionMap.size,
                version: VERSION
            };
        }

        return {
            VERSION: VERSION,
            setConfig: setConfig,
            obfuscate: obfuscate,
            getStats: getStats,
            encryptString: encryptString,
            generateObfuscatedName: generateObfuscatedName
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = AdvancedObfuscator;
    } else {
        globalContext.AdvancedObfuscator = AdvancedObfuscator;
    }

})(typeof window !== 'undefined' ? window : (typeof global !== 'undefined' ? global : this));