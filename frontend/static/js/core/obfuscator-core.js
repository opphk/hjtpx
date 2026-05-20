(function(globalContext) {
    'use strict';

    const AdvancedObfuscator = (function() {
        const VERSION = '5.0.0';
        const CHAR_POOL = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_$';
        const HEX_POOL = '0123456789abcdef';
        const STRING_ENCODING_POOL = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
        
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
            enableRC4Encryption: true,
            enableAESEncryption: false,
            stringEncryptionKey: 'hjtpx-obfuscate-key-2024',
            obfuscationLevel: 3,
            shuffleStrings: true,
            addOpaquePredicates: true,
            controlFlowIterations: 3,
            addDebugTrap: true,
            enableSelfDefending: true,
            enableAntiDebug: true,
            maxStringLength: 100,
            splitStrings: true,
            stringChunkSize: 50
        };

        let variableMap = new Map();
        let stringMap = new Map();
        let functionMap = new Map();
        let usedNames = new Set();
        let identifierCounter = 0;
        let stringCounter = 0;
        let encryptionKey = null;
        let rc4State = null;

        function setConfig(cfg) {
            Object.assign(config, cfg);
            if (cfg.stringEncryptionKey) {
                encryptionKey = cfg.stringEncryptionKey;
                rc4State = initRC4State(encryptionKey);
            }
        }

        function initRC4State(key) {
            const state = new Array(256);
            for (let i = 0; i < 256; i++) {
                state[i] = i;
            }
            
            let j = 0;
            const keyBytes = stringToBytes(key);
            for (let i = 0; i < 256; i++) {
                j = (j + state[i] + keyBytes[i % keyBytes.length]) % 256;
                [state[i], state[j]] = [state[j], state[i]];
            }
            
            return { state, i: 0, j: 0 };
        }

        function rc4Encrypt(data) {
            if (!rc4State) {
                rc4State = initRC4State(config.stringEncryptionKey);
            }
            
            const state = rc4State.state.slice();
            let i = 0, j = 0;
            const result = [];
            
            const dataBytes = stringToBytes(data);
            for (let n = 0; n < dataBytes.length; n++) {
                i = (i + 1) % 256;
                j = (j + state[i]) % 256;
                [state[i], state[j]] = [state[j], state[i]];
                const k = state[(state[i] + state[j]) % 256];
                result.push(dataBytes[n] ^ k);
            }
            
            return result;
        }

        function rc4Decrypt(encryptedBytes) {
            return rc4Encrypt(encryptedBytes);
        }

        function stringToBytes(str) {
            const bytes = [];
            for (let i = 0; i < str.length; i++) {
                bytes.push(str.charCodeAt(i) & 0xFF);
            }
            return bytes;
        }

        function bytesToString(bytes) {
            return String.fromCharCode.apply(null, bytes);
        }

        function bytesToBase64(bytes) {
            let result = '';
            const chunkSize = 3;
            for (let i = 0; i < bytes.length; i += chunkSize) {
                const chunk = bytes.slice(i, i + chunkSize);
                const padding = chunk.length < chunkSize ? chunkSize - chunk.length : 0;
                
                const b1 = chunk[0] || 0;
                const b2 = chunk[1] || 0;
                const b3 = chunk[2] || 0;
                
                result += STRING_ENCODING_POOL[b1 >> 2];
                result += STRING_ENCODING_POOL[((b1 & 0x03) << 4) | (b2 >> 4)];
                result += padding > 1 ? '=' : STRING_ENCODING_POOL[((b2 & 0x0F) << 2) | (b3 >> 6)];
                result += padding > 0 ? '=' : STRING_ENCODING_POOL[b3 & 0x3F];
            }
            return result;
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
            
            if (config.enableRC4Encryption) {
                const encrypted = rc4Encrypt(str);
                const hexString = encrypted.map(b => b.toString(16).padStart(2, '0')).join('');
                return btoa(hexString);
            } else {
                let encrypted = '';
                for (let i = 0; i < str.length; i++) {
                    const charCode = str.charCodeAt(i);
                    const keyChar = key.charCodeAt(i % key.length);
                    const encryptedCode = charCode ^ keyChar;
                    encrypted += String.fromCharCode(encryptedCode);
                }
                return btoa(encrypted);
            }
        }

        function splitString(str) {
            if (!config.splitStrings || str.length <= config.stringChunkSize) {
                return [str];
            }
            
            const chunks = [];
            for (let i = 0; i < str.length; i += config.stringChunkSize) {
                chunks.push(str.slice(i, i + config.stringChunkSize));
            }
            return chunks;
        }

        function obfuscateStrings(code) {
            if (!config.enableStringEncryption) return code;

            const stringPattern = /(["'`])([^\\]|\\.)*?\1/g;
            let result = code;

            result = result.replace(stringPattern, (match) => {
                const quote = match[0];
                const content = match.slice(1, -1);
                
                if (content.length < 3 || content.length > config.maxStringLength) return match;
                if (content.includes('__decrypt') || content.includes('window.')) return match;
                
                stringCounter++;
                const chunks = splitString(content);
                const encryptedChunks = chunks.map(chunk => encryptString(chunk));
                stringMap.set(content, encryptedChunks);
                
                if (chunks.length === 1) {
                    return quote + 'window.__decrypt' + quote + encryptedChunks[0] + quote + ')';
                } else {
                    const combinedExpr = encryptedChunks.map((enc, idx) => 
                        'window.__decrypt' + quote + enc + quote + ')'
                    ).join('+');
                    return '((' + combinedExpr + '))';
                }
            });

            return result;
        }

        function flattenControlFlow(code) {
            if (!config.enableControlFlowFlattening) return code;

            let result = code;
            
            for (let i = 0; i < config.controlFlowIterations; i++) {
                result = flattenIfStatements(result);
                result = flattenLoops(result);
                result = flattenSwitchStatements(result);
                result = addStateMachine(result);
            }

            return result;
        }

        function flattenSwitchStatements(code) {
            const switchPattern = /\bswitch\s*\(([^)]+)\)\s*\{([^}]+)\}/g;
            
            return code.replace(switchPattern, (match, expr, body) => {
                const stateVar = generateObfuscatedName();
                const cases = [];
                let caseIndex = 0;
                
                const casePattern = /\bcase\s+(\d+)\s*:/g;
                let caseMatch;
                while ((caseMatch = casePattern.exec(body)) !== null) {
                    cases.push({ value: parseInt(caseMatch[1]), index: caseIndex++ });
                }
                
                if (cases.length < 2) return match;
                
                const switchImpl = `(function(){var ${stateVar}=${expr};${body}})()`;
                return switchImpl;
            });
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
            const numStates = 3 + Math.floor(Math.random() * 3);
            const stateVar = generateObfuscatedName();
            const transitions = [];
            
            for (let i = 0; i < numStates; i++) {
                const nextState = (i + 1) % numStates;
                transitions.push(`case ${i}: ${stateVar}=${nextState};break;`);
            }
            
            const machineCode = `(function(){var ${stateVar}=0;switch(${stateVar}){${transitions.join('')}case ${numStates}:}})();`;
            
            if (config.enableDeadCodeInjection) {
                return machineCode + injectDeadCode(code);
            }
            return machineCode + code;
        }

        function injectDeadCode(code) {
            if (!config.enableDeadCodeInjection) return code;

            const deadCode = generateDeadCode();
            
            if (config.addDebugTrap) {
                const trapCode = generateDebugTrap();
                return '(function(){' + deadCode + trapCode + '})();' + code;
            }
            
            return '(function(){' + deadCode + '})();' + code;
        }

        function generateDebugTrap() {
            const trapVar1 = generateObfuscatedName();
            const trapVar2 = generateObfuscatedName();
            const trapVar3 = generateObfuscatedName();
            
            return `
                var ${trapVar1}=function(){
                    var ${trapVar2}=Date.now();
                    var ${trapVar3}=Math.random();
                    if(${trapVar3}<0.001){
                        debugger;
                    }
                };
                ${trapVar1}();
            `.replace(/\s+/g, ' ').trim();
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
            
            if (config.enableRC4Encryption) {
                return `
                    window.__decrypt=function(_0xE){
                        var _0xK=atob('${key}');
                        var _0xS=[];
                        for(var _0xI=0;_0xI<256;_0xI++){_0xS[_0xI]=_0xI;}
                        var _0xJ=0;
                        for(var _0xI=0;_0xI<256;_0xI++){
                            _0xJ=(_0xJ+_0xS[_0xI]+_0xK.charCodeAt(_0xI%_0xK.length))%256;
                            var _0xT=_0xS[_0xI];_0xS[_0xI]=_0xS[_0xJ];_0xS[_0xJ]=_0xT;
                        }
                        var _0xH=atob(_0xE);
                        var _0xR='';
                        var _0xI=0,_0xJ=0;
                        for(var _0xP=0;_0xP<_0xH.length;_0xP++){
                            _0xI=(_0xI+1)%256;
                            _0xJ=(_0xJ+_0xS[_0xI])%256;
                            var _0xT=_0xS[_0xI];_0xS[_0xI]=_0xS[_0xJ];_0xS[_0xJ]=_0xT;
                            var _0xK2=_0xS[(_0xS[_0xI]+_0xS[_0xJ])%256];
                            _0xR+=String.fromCharCode(_0xH.charCodeAt(_0xP)^_0xK2);
                        }
                        return _0xR;
                    };
                `.replace(/\s+/g, ' ').trim();
            } else {
                return `window.__decrypt=function(_0xE){var _0xK=atob('${key}');var _0xC=atob(_0xE);var _0xO='';for(var _0xI=0;_0xI<_0xC.length;_0xI++){_0xO+=String.fromCharCode(_0xC.charCodeAt(_0xI)^_0xK.charCodeAt(_0xI%_0xK.length));}return _0xO;};`;
            }
        }

        function compressCode(code) {
            code = code.replace(/\s+/g, ' ');
            code = code.replace(/\s*([{};,:])\s*/g, '$1');
            code = code.replace(/\{\s*/g, '{');
            code = code.replace(/\s*\}/g, '}');
            code = code.replace(/\n\s*\n/g, '\n');
            
            return code.trim();
        }

        function addSelfDefendingCode(code) {
            if (!config.enableSelfDefending) return code;

            const checksumVar = generateObfuscatedName();
            const hashVar = generateObfuscatedName();
            
            const selfDefendingCode = `
                (function(){
                    var ${checksumVar}=0;
                    var ${hashVar}=function(_0xC){
                        var _0xH=0;
                        for(var _0xI=0;_0xI<_0xC.length;_0xI++){
                            _0xH=((_0xH<<5)-_0xH)+_0xC.charCodeAt(_0xI);
                            _0xH=_0xH&_0xH;
                        }
                        return Math.abs(_0xH);
                    };
                    var _0xS=document.getElementsByTagName('script');
                    if(_0xS.length>0){
                        var _0xSC=_0xS[_0xS.length-1].textContent;
                        var _0xCK=${hashVar}(_0xSC);
                        if(_0xCK!==${checksumVar}){
                            throw new Error('Code integrity check failed');
                        }
                    }
                })();
            `.replace(/\s+/g, ' ').trim();
            
            return selfDefendingCode + code;
        }

        function addAntiDebugCode(code) {
            if (!config.enableAntiDebug) return code;

            const antiDebugCode = `
                (function(){
                    var _0xT=Date.now();
                    var _0xF=function(){
                        var _0xN=Date.now();
                        if(_0xN-_0xT>100){
                            throw new Error('Execution time anomaly detected');
                        }
                        _0xT=_0xN;
                    };
                    setInterval(_0xF,500);
                    var _0xCK=function(){
                        var _0xS=performance.now();
                        debugger;
                        var _0xE=performance.now()-_0xS;
                        if(_0xE>50){
                            throw new Error('Debugger detected');
                        }
                    };
                    setTimeout(_0xCK,1000);
                })();
            `.replace(/\s+/g, ' ').trim();
            
            return antiDebugCode + code;
        }

        function obfuscate(code) {
            variableMap.clear();
            stringMap.clear();
            functionMap.clear();
            usedNames.clear();
            identifierCounter = 0;
            stringCounter = 0;
            encryptionKey = config.stringEncryptionKey;
            rc4State = initRC4State(encryptionKey);

            let result = code;

            if (config.enableDeadCodeInjection) {
                result = injectDeadCode(result);
            }

            if (config.enableAntiDebug) {
                result = addAntiDebugCode(result);
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

            if (config.enableSelfDefending) {
                result = addSelfDefendingCode(result);
            }

            result = compressCode(result);

            return result;
        }

        function getStats() {
            return {
                variablesObfuscated: variableMap.size,
                stringsEncrypted: stringMap.size,
                functionsWrapped: functionMap.size,
                version: VERSION,
                rc4Enabled: config.enableRC4Encryption,
                controlFlowIterations: config.controlFlowIterations,
                selfDefendingEnabled: config.enableSelfDefending,
                antiDebugEnabled: config.enableAntiDebug
            };
        }

        return {
            VERSION: VERSION,
            setConfig: setConfig,
            obfuscate: obfuscate,
            getStats: getStats,
            encryptString: encryptString,
            generateObfuscatedName: generateObfuscatedName,
            rc4Encrypt: rc4Encrypt,
            rc4Decrypt: rc4Decrypt
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = AdvancedObfuscator;
    } else {
        globalContext.AdvancedObfuscator = AdvancedObfuscator;
    }

})(typeof window !== 'undefined' ? window : (typeof global !== 'undefined' ? global : this));