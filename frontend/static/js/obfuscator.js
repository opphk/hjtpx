(function(globalContext) {
    'use strict';

    var Obfuscator = (function() {
        var version = '2.0.0';

        var defaultConfig = {
            enableVariableObfuscation: true,
            enableStringEncryption: true,
            enableCodeCompression: true,
            enableControlFlowFlattening: true,
            enableDeadCodeInjection: false,
            enableFunctionWrapping: true,
            enableAntiDebug: true,
            enableSelfDestruct: false,
            enableMemoryProtection: false,
            removeComments: true,
            preserveConsole: true,
            compressWhitespace: true,
            seed: 12345,
            targetObfuscationRate: 0.7
        };

        var variableMap = {};
        var functionMap = {};
        var usedNames = {};
        var stringCount = 0;
        var functionCount = 0;

        var reservedWords = [
            'if', 'else', 'for', 'while', 'do', 'switch', 'case', 'default',
            'break', 'continue', 'return', 'try', 'catch', 'finally', 'throw',
            'new', 'delete', 'typeof', 'instanceof', 'void', 'this', 'super',
            'class', 'extends', 'static', 'get', 'set', 'import', 'export',
            'from', 'as', 'const', 'let', 'var', 'function', 'async', 'await',
            'yield', 'true', 'false', 'null', 'undefined', 'NaN', 'Infinity',
            'arguments', 'eval', 'constructor', 'prototype', 'console'
        ];

        function generateObfuscatedName() {
            var prefixes = ['_0x', '_0x1', '_0x2', '_0x3', '_0x4', '_0x5'];
            var chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_$';

            functionCount++;
            if (functionCount < prefixes.length) {
                if (!usedNames[prefixes[functionCount]]) {
                    usedNames[prefixes[functionCount]] = true;
                    return prefixes[functionCount];
                }
            }

            var length = 3 + (functionCount % 4);
            var name = '_0x';
            for (var i = 0; i < length; i++) {
                var idx = Math.floor(Math.random() * chars.length);
                name += chars[idx];
            }

            if (!usedNames[name]) {
                usedNames[name] = true;
                return name;
            }

            return generateObfuscatedName();
        }

        function isReservedWord(word) {
            return reservedWords.indexOf(word) !== -1;
        }

        function isAlreadyObfuscated(name) {
            return variableMap.hasOwnProperty(name);
        }

        function removeComments(code) {
            code = code.replace(/\/\*[\s\S]*?\*\//g, '');
            code = code.replace(/\/\/[^\n]*/g, '');
            return code;
        }

        function obfuscateVariables(code) {
            var identifierPattern = /([a-zA-Z_$][a-zA-Z0-9_$]*)\s*=/g;
            var result = code;
            var matches = [];

            var match;
            while ((match = identifierPattern.exec(code)) !== null) {
                var name = match[1];
                if (!isReservedWord(name) && !isAlreadyObfuscated(name)) {
                    var newName = generateObfuscatedName();
                    variableMap[name] = newName;
                }
                matches.push(match);
            }

            for (var original in variableMap) {
                if (variableMap.hasOwnProperty(original)) {
                    var obfuscated = variableMap[original];
                    var regex = new RegExp('\\b' + escapeRegExp(original) + '\\b', 'g');
                    result = result.replace(regex, obfuscated);
                }
            }

            return result;
        }

        function escapeRegExp(string) {
            return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        }

        function shouldEncryptString(s) {
            if (s.length < 3) {
                return false;
            }

            var keywords = ['function', 'var ', 'let ', 'const ', 'if ', 'else',
                          'for ', 'while', 'return ', 'true', 'false', 'null',
                          'undefined', 'console', 'window', 'document',
                          'localStorage', 'sessionStorage', 'fetch'];

            for (var i = 0; i < keywords.length; i++) {
                if (s.indexOf(keywords[i]) !== -1) {
                    return false;
                }
            }

            return true;
        }

        function encryptString(s, key) {
            var encrypted = '';
            for (var i = 0; i < s.length; i++) {
                var xorChar = key.charCodeAt(i % key.length);
                encrypted += String.fromCharCode(s.charCodeAt(i) ^ xorChar);
            }
            return btoa(unescape(encodeURIComponent(encrypted)));
        }

        function encryptStrings(code, key) {
            var result = '';
            var i = 0;
            var codeBytes = code.split('');

            while (i < codeBytes.length) {
                var char = codeBytes[i];
                if (char === '"' || char === '\'' || char === '`') {
                    var quote = char;
                    var start = i;
                    i++;

                    var strContent = '';
                    while (i < codeBytes.length) {
                        if (codeBytes[i] === '\\' && i + 1 < codeBytes.length) {
                            strContent += codeBytes[i];
                            i++;
                            strContent += codeBytes[i];
                            i++;
                        } else if (codeBytes[i] === quote) {
                            i++;
                            break;
                        } else {
                            strContent += codeBytes[i];
                            i++;
                        }
                    }

                    if (shouldEncryptString(strContent)) {
                        var encrypted = encryptString(strContent, key);
                        stringCount++;
                        result += quote + '__dec' + stringCount + "('" + encrypted + "')" + quote;
                    } else {
                        result += quote + strContent + quote;
                    }
                } else {
                    result += char;
                    i++;
                }
            }

            return result;
        }

        function generateDecoderFunction(key) {
            var decoderCode = '(function(_0xK){';
            decoderCode += 'window.__dec=function(_0xS){';
            decoderCode += 'var _0xD=atob(_0xS);';
            decoderCode += 'var _0xR="";';
            decoderCode += 'for(var _0xI=0;_0xI<_0xD.length;_0xI++){';
            decoderCode += '_0xR+=String.fromCharCode(_0xD.charCodeAt(_0xI)^_0xK.charCodeAt(_0xI%_0xK.length));';
            decoderCode += '}';
            decoderCode += 'return _0xR;';
            decoderCode += '};';
            decoderCode += '})("' + btoa(key) + '");';

            for (var i = 1; i <= stringCount; i++) {
                decoderCode = decoderCode.replace('__dec' + i, '__dec' + i);
            }

            return decoderCode;
        }

        function wrapCode(code, key) {
            var decoders = generateDecoderFunction(key);
            return decoders + '\n' + code;
        }

        function flattenControlFlow(code) {
            var ifPattern = /if\s*\(([^)]+)\)\s*\{([^}]+)\}/g;
            code = code.replace(ifPattern, function(match, condition, body) {
                return '(function(){var _0xF1=!!(' + condition + ');if(_0xF1){' + body + '}})()';
            });

            var forPattern = /for\s*\(([^)]+)\)\s*\{([^}]+)\}/g;
            code = code.replace(forPattern, function(match, init, body) {
                return '(function(){var _0xF2=0,_0xF3=' + init + ';for(;_0xF2;){' + body + ';_0xF2=_0xF3;}})(0,function(){return 1;})';
            });

            return code;
        }

        function injectDeadCode() {
            var deadCode = '(function(){';
            deadCode += 'var _0xD1=Math.random();';
            deadCode += 'var _0xD2=_0xD1>0?_0xD1:0;';
            deadCode += 'if(_0xD2<0){console.log("' + generateRandomString(8) + '");}';
            deadCode += '})();';
            return deadCode;
        }

        function generateRandomString(length) {
            var chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
            var result = '';
            for (var i = 0; i < length; i++) {
                result += chars[Math.floor(Math.random() * chars.length)];
            }
            return result;
        }

        function compressCode(code) {
            code = code.replace(/\s+/g, ' ');
            code = code.replace(/\s*([{};,:])\s*/g, '$1');
            code = code.replace(/;\s*}/g, ';}');
            code = code.replace(/{\s*/g, '{');
            code = code.replace(/\s*}/g, '}');
            code = code.replace(/\n\s*\n/g, '\n');
            return code.trim();
        }

        function injectAntiDebug() {
            var antiDebug = ';';
            antiDebug += '(function(){';
            antiDebug += 'var _0xAD={';
            antiDebug += 'check:function(){';
            antiDebug += 'if(window.outerHeight-window.innerHeight>100||window.outerWidth-window.innerWidth>100){';
            antiDebug += '_0xAD.trigger();';
            antiDebug += '}';
            antiDebug += 'var _0xT=function(){};';
            antiDebug += '_0xT.toString=function(){';
            antiDebug += 'var _0xD=new Date();';
            antiDebug += 'var _0xE=_0xD.getTime();';
            antiDebug += 'debugger;';
            antiDebug += 'var _0xF=new Date();';
            antiDebug += 'if(_0xF.getTime()-_0xE>100){';
            antiDebug += '_0xAD.trigger();';
            antiDebug += '}';
            antiDebug += '};';
            antiDebug += 'setInterval(function(){console.log(_0xT);},1000);';
            antiDebug += '},';
            antiDebug += 'trigger:function(){';
            antiDebug += 'document.documentElement.style.display="none";';
            antiDebug += 'document.body.innerHTML=\'<div style="padding:50px;text-align:center;"><h1>访问受限</h1></div>\';';
            antiDebug += 'throw new Error("Debug detected");';
            antiDebug += '},';
            antiDebug += 'start:function(){';
            antiDebug += 'document.addEventListener("keydown",function(e){';
            antiDebug += 'if(e.keyCode===123){';
            antiDebug += '_0xAD.trigger();';
            antiDebug += '}';
            antiDebug += '});';
            antiDebug += 'document.addEventListener("contextmenu",function(e){';
            antiDebug += 'e.preventDefault();';
            antiDebug += '});';
            antiDebug += 'setInterval(function(){';
            antiDebug += 'var _0xW=window.outerWidth-window.innerWidth>100;';
            antiDebug += 'var _0xH=window.outerHeight-window.innerHeight>100;';
            antiDebug += 'if(_0xW||_0xH){';
            antiDebug += '_0xAD.trigger();';
            antiDebug += '}';
            antiDebug += '},1000);';
            antiDebug += '}';
            antiDebug += '};';
            antiDebug += 'if(document.readyState==="complete"){';
            antiDebug += '_0xAD.start();';
            antiDebug += '}else{';
            antiDebug += 'window.addEventListener("load",function(){_0xAD.start();});';
            antiDebug += '}';
            antiDebug += '_0xAD.check();';
            antiDebug += '})();';

            return antiDebug;
        }

        function obfuscate(code, config) {
            if (!code || code.length === 0) {
                throw new Error('Code cannot be empty');
            }

            config = config || defaultConfig;
            variableMap = {};
            functionMap = {};
            usedNames = {};
            stringCount = 0;
            functionCount = 0;

            var key = 'hjtpx-obfuscate-key-2024';
            var result = code;

            if (config.removeComments) {
                result = removeComments(result);
            }

            if (config.enableVariableObfuscation) {
                result = obfuscateVariables(result);
            }

            if (config.enableStringEncryption) {
                result = encryptStrings(result, key);
            }

            if (config.enableFunctionWrapping) {
                result = wrapCode(result, key);
            }

            if (config.enableControlFlowFlattening) {
                result = flattenControlFlow(result);
            }

            if (config.enableAntiDebug) {
                result = injectAntiDebug() + result;
            }

            if (config.enableDeadCodeInjection) {
                result = injectDeadCode() + result;
            }

            if (config.enableCodeCompression) {
                result = compressCode(result);
            }

            return result;
        }

        function getStats() {
            return {
                variablesObfuscated: Object.keys(variableMap).length,
                stringsEncrypted: stringCount,
                functionsWrapped: functionCount
            };
        }

        function validateObfuscatedCode(code) {
            var openBraces = (code.match(/\{/g) || []).length;
            var closeBraces = (code.match(/\}/g) || []).length;
            if (openBraces !== closeBraces) {
                return { valid: false, message: 'Unbalanced braces' };
            }

            var openParens = (code.match(/\(/g) || []).length;
            var closeParens = (code.match(/\)/g) || []).length;
            if (openParens !== closeParens) {
                return { valid: false, message: 'Unbalanced parentheses' };
            }

            if (code.indexOf('TODO') !== -1 || code.indexOf('FIXME') !== -1) {
                return { valid: false, message: 'Code contains TODO or FIXME' };
            }

            return { valid: true, message: 'Valid' };
        }

        function calculateEntropy(code) {
            var charFreq = {};
            for (var i = 0; i < code.length; i++) {
                var char = code[i];
                charFreq[char] = (charFreq[char] || 0) + 1;
            }

            var entropy = 0;
            var length = code.length;
            for (var c in charFreq) {
                var p = charFreq[c] / length;
                if (p > 0) {
                    entropy -= p * Math.log2(p);
                }
            }

            return Math.round(entropy * 100) / 100;
        }

        function estimateQuality(original, obfuscated) {
            var entropyOriginal = calculateEntropy(original);
            var entropyObfuscated = calculateEntropy(obfuscated);

            var entropyImprovement = entropyObfuscated - entropyOriginal;
            var sizeRatio = obfuscated.length / original.length;
            var readabilityScore = Math.max(0, 100 - entropyObfuscated * 5);
            var unreadabilityPercent = Math.min(100, readabilityScore);

            return {
                entropyOriginal: entropyOriginal,
                entropyObfuscated: entropyObfuscated,
                entropyImprovement: Math.round(entropyImprovement * 100) / 100,
                sizeRatio: Math.round(sizeRatio * 100) / 100,
                readabilityScore: Math.round(readabilityScore * 100) / 100,
                unreadabilityPercent: Math.round(unreadabilityPercent * 100) / 100
            };
        }

        return {
            version: version,
            obfuscate: obfuscate,
            getStats: getStats,
            validate: validateObfuscatedCode,
            calculateEntropy: calculateEntropy,
            estimateQuality: estimateQuality,
            defaultConfig: defaultConfig
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = Obfuscator;
    } else {
        globalContext.Obfuscator = Obfuscator;
    }

})(typeof window !== 'undefined' ? window : this);

(function() {
    if (typeof CryptoUtils !== 'undefined') {
        CryptoUtils.obfuscate = function(code, config) {
            return Obfuscator.obfuscate(code, config);
        };

        CryptoUtils.obfuscator = Obfuscator;
    }
})();
