(function(globalContext) {
    'use strict';

    const CodeVirtualization = (function() {
        const VERSION = '3.0.0';
        
        const OPCODES = {
            NOP: 0x00,
            PUSH: 0x01,
            POP: 0x02,
            ADD: 0x03,
            SUB: 0x04,
            MUL: 0x05,
            DIV: 0x06,
            MOD: 0x07,
            AND: 0x08,
            OR: 0x09,
            XOR: 0x0A,
            NOT: 0x0B,
            SHL: 0x0C,
            SHR: 0x0D,
            CMP: 0x0E,
            JMP: 0x0F,
            JZ: 0x10,
            JNZ: 0x11,
            CALL: 0x12,
            RET: 0x13,
            LOAD: 0x14,
            STORE: 0x15,
            HALT: 0x16,
            XOR_ROT: 0x17,
            ADD_SUB: 0x18,
            MUL_DIV: 0x19,
            RANDOM: 0x1A,
            HASH: 0x1B,
            ENCRYPT: 0x1C,
            DECRYPT: 0x1D,
            VALIDATE: 0x1E,
            CHECKSUM: 0x1F,
            STRING_LENGTH: 0x20,
            STRING_CHAR_AT: 0x21,
            STRING_CONCAT: 0x22,
            STRING_EQUALS: 0x23,
            ARRAY_CREATE: 0x24,
            ARRAY_GET: 0x25,
            ARRAY_SET: 0x26,
            ARRAY_LENGTH: 0x27,
            OBJECT_CREATE: 0x28,
            OBJECT_GET: 0x29,
            OBJECT_SET: 0x2A,
            TIME_NOW: 0x2B,
            TIME_SLEEP: 0x2C,
            CONSOLE_LOG: 0x2D,
            ERROR_THROW: 0x2E,
            ASSERT: 0x2F
        };

        const _0xVM = {
            memory: new Uint32Array(4096),
            registers: new Uint32Array(8),
            stack: [],
            stringTable: [],
            arrayTable: [],
            objectTable: [],
            ip: 0,
            sp: 0,
            running: false,
            breakpoints: new Set(),
            instructionCount: 0,
            maxInstructions: 100000,
            code: [],
            handlers: {},
            secretKey: null,
            checksum: null,
            enableStringOps: true,
            enableArrayOps: true,
            enableTimeOps: true
        };

        function initVM() {
            _0xVM.memory.fill(0);
            _0xVM.registers.fill(0);
            _0xVM.stack = [];
            _0xVM.stringTable = [];
            _0xVM.arrayTable = [];
            _0xVM.objectTable = [];
            _0xVM.ip = 0;
            _0xVM.sp = 0;
            _0xVM.running = false;
            _0xVM.instructionCount = 0;
            _0xVM.secretKey = generateSecretKey();
        }

        function resetVM() {
            initVM();
        }

        function getVMStatus() {
            return {
                running: _0xVM.running,
                instructionCount: _0xVM.instructionCount,
                stackDepth: _0xVM.sp,
                memoryUsed: _0xVM.memory.filter(v => v !== 0).length,
                stringsCount: _0xVM.stringTable.length,
                arraysCount: _0xVM.arrayTable.length,
                objectsCount: _0xVM.objectTable.length,
                maxInstructions: _0xVM.maxInstructions
            };
        }

        function setMaxInstructions(max) {
            if (typeof max === 'number' && max > 0) {
                _0xVM.maxInstructions = max;
                return true;
            }
            return false;
        }

        function enableStringOperations(enabled) {
            _0xVM.enableStringOps = enabled;
        }

        function enableArrayOperations(enabled) {
            _0xVM.enableArrayOps = enabled;
        }

        function enableTimeOperations(enabled) {
            _0xVM.enableTimeOps = enabled;
        }

        function addBreakpoint(address) {
            _0xVM.breakpoints.add(address);
        }

        function removeBreakpoint(address) {
            _0xVM.breakpoints.delete(address);
        }

        function clearBreakpoints() {
            _0xVM.breakpoints.clear();
        }

        function generateSecretKey() {
            const array = new Uint8Array(32);
            crypto.getRandomValues(array);
            return Array.from(array);
        }

        function encodeInstruction(opcode, ...args) {
            const instruction = [opcode];
            for (const arg of args) {
                if (typeof arg === 'number') {
                    instruction.push(arg & 0xFF);
                    instruction.push((arg >> 8) & 0xFF);
                    instruction.push((arg >> 16) & 0xFF);
                    instruction.push((arg >> 24) & 0xFF);
                } else if (typeof arg === 'string') {
                    const encoded = stringToBytes(arg);
                    instruction.push(encoded.length);
                    instruction.push(...encoded);
                }
            }
            return instruction;
        }

        function stringToBytes(str) {
            const encoder = new TextEncoder();
            return Array.from(encoder.encode(str));
        }

        function bytesToString(bytes) {
            const decoder = new TextDecoder();
            return decoder.decode(new Uint8Array(bytes));
        }

        function decodeInstruction(code, offset) {
            const opcode = code[offset];
            const args = [];
            
            let pos = offset + 1;
            
            switch (opcode) {
                case OPCODES.PUSH:
                case OPCODES.JMP:
                case OPCODES.JZ:
                case OPCODES.JNZ:
                case OPCODES.LOAD:
                case OPCODES.STORE:
                    const value = (code[pos] | (code[pos + 1] << 8) | (code[pos + 2] << 16) | (code[pos + 3] << 24)) >>> 0;
                    args.push(value);
                    pos += 4;
                    break;
                    
                case OPCODES.CALL:
                    const len = code[pos];
                    const strBytes = code.slice(pos + 1, pos + 1 + len);
                    args.push(bytesToString(strBytes));
                    pos += 1 + len;
                    break;
                    
                case OPCODES.ENCRYPT:
                case OPCODES.DECRYPT:
                case OPCODES.HASH:
                    const dataLen = code[pos];
                    const dataBytes = code.slice(pos + 1, pos + 1 + dataLen);
                    args.push(bytesToString(dataBytes));
                    pos += 1 + dataLen;
                    break;
            }
            
            return { opcode, args, length: pos - offset };
        }

        function executeInstruction(instruction) {
            const { opcode, args } = instruction;

            switch (opcode) {
                case OPCODES.NOP:
                    break;
                    
                case OPCODES.PUSH:
                    _0xVM.stack.push(args[0]);
                    _0xVM.sp++;
                    break;
                    
                case OPCODES.POP:
                    if (_0xVM.sp > 0) {
                        _0xVM.sp--;
                        _0xVM.stack.pop();
                    }
                    break;
                    
                case OPCODES.ADD:
                    if (_0xVM.sp >= 2) {
                        const b = _0xVM.stack.pop();
                        const a = _0xVM.stack.pop();
                        _0xVM.stack.push((a + b) >>> 0);
                        _0xVM.sp--;
                    }
                    break;
                    
                case OPCODES.SUB:
                    if (_0xVM.sp >= 2) {
                        const b = _0xVM.stack.pop();
                        const a = _0xVM.stack.pop();
                        _0xVM.stack.push((a - b) >>> 0);
                        _0xVM.sp--;
                    }
                    break;
                    
                case OPCODES.MUL:
                    if (_0xVM.sp >= 2) {
                        const b = _0xVM.stack.pop();
                        const a = _0xVM.stack.pop();
                        _0xVM.stack.push((a * b) >>> 0);
                        _0xVM.sp--;
                    }
                    break;
                    
                case OPCODES.DIV:
                    if (_0xVM.sp >= 2) {
                        const b = _0xVM.stack.pop();
                        const a = _0xVM.stack.pop();
                        _0xVM.stack.push(b !== 0 ? (a / b) >>> 0 : 0);
                        _0xVM.sp--;
                    }
                    break;
                    
                case OPCODES.XOR:
                    if (_0xVM.sp >= 2) {
                        const b = _0xVM.stack.pop();
                        const a = _0xVM.stack.pop();
                        _0xVM.stack.push(a ^ b);
                        _0xVM.sp--;
                    }
                    break;
                    
                case OPCODES.SHL:
                    if (_0xVM.sp >= 2) {
                        const shift = _0xVM.stack.pop();
                        const value = _0xVM.stack.pop();
                        _0xVM.stack.push((value << (shift & 0x1F)) >>> 0);
                        _0xVM.sp--;
                    }
                    break;
                    
                case OPCODES.SHR:
                    if (_0xVM.sp >= 2) {
                        const shift = _0xVM.stack.pop();
                        const value = _0xVM.stack.pop();
                        _0xVM.stack.push(value >>> (shift & 0x1F));
                        _0xVM.sp--;
                    }
                    break;
                    
                case OPCODES.JMP:
                    _0xVM.ip = args[0];
                    break;
                    
                case OPCODES.JZ:
                    if (_0xVM.sp > 0) {
                        const value = _0xVM.stack.pop();
                        _0xVM.sp--;
                        if (value === 0) {
                            _0xVM.ip = args[0];
                        }
                    }
                    break;
                    
                case OPCODES.JNZ:
                    if (_0xVM.sp > 0) {
                        const value = _0xVM.stack.pop();
                        _0xVM.sp--;
                        if (value !== 0) {
                            _0xVM.ip = args[0];
                        }
                    }
                    break;
                    
                case OPCODES.LOAD:
                    const addr = args[0];
                    if (addr < _0xVM.memory.length) {
                        _0xVM.stack.push(_0xVM.memory[addr]);
                        _0xVM.sp++;
                    }
                    break;
                    
                case OPCODES.STORE:
                    const storeAddr = args[0];
                    if (_0xVM.sp > 0 && storeAddr < _0xVM.memory.length) {
                        _0xVM.memory[storeAddr] = _0xVM.stack.pop();
                        _0xVM.sp--;
                    }
                    break;
                    
                case OPCODES.XOR_ROT:
                    if (_0xVM.sp >= 2) {
                        const key = _0xVM.stack.pop();
                        const value = _0xVM.stack.pop();
                        const rotated = ((value << 8) | (value >>> 24)) ^ key;
                        _0xVM.stack.push(rotated);
                        _0xVM.sp--;
                    }
                    break;
                    
                case OPCODES.RANDOM:
                    const rand = Math.floor(Math.random() * 0xFFFFFFFF);
                    _0xVM.stack.push(rand);
                    _0xVM.sp++;
                    break;
                    
                case OPCODES.HASH:
                    if (_0xVM.sp > 0) {
                        const data = args[0];
                        const hash = computeVirtualHash(data);
                        _0xVM.stack.push(hash);
                    }
                    break;
                    
                case OPCODES.ENCRYPT:
                    const plaintext = args[0];
                    const encrypted = virtualEncrypt(plaintext);
                    _0xVM.stack.push(encrypted.length);
                    _0xVM.sp++;
                    break;
                    
                case OPCODES.CHECKSUM:
                    const computedChecksum = computeChecksum(_0xVM.code);
                    _0xVM.stack.push(computedChecksum);
                    _0xVM.sp++;
                    break;
                    
                case OPCODES.VALIDATE:
                    const expected = args[0];
                    if (_0xVM.sp > 0) {
                        const actual = _0xVM.stack.pop();
                        _0xVM.sp--;
                        if (actual !== expected) {
                            throw new Error('Validation failed');
                        }
                    }
                    break;
                    
                case OPCODES.STRING_LENGTH:
                    if (_0xVM.sp > 0) {
                        const strIdx = _0xVM.stack.pop();
                        _0xVM.sp--;
                        if (strIdx >= 0 && strIdx < _0xVM.stringTable.length) {
                            _0xVM.stack.push(_0xVM.stringTable[strIdx].length);
                            _0xVM.sp++;
                        } else {
                            _0xVM.stack.push(0);
                            _0xVM.sp++;
                        }
                    }
                    break;
                    
                case OPCODES.STRING_CHAR_AT:
                    if (_0xVM.sp >= 2) {
                        const idx = _0xVM.stack.pop();
                        const strIdx = _0xVM.stack.pop();
                        _0xVM.sp -= 2;
                        if (strIdx >= 0 && strIdx < _0xVM.stringTable.length) {
                            const str = _0xVM.stringTable[strIdx];
                            _0xVM.stack.push(idx >= 0 && idx < str.length ? str.charCodeAt(idx) : 0);
                            _0xVM.sp++;
                        } else {
                            _0xVM.stack.push(0);
                            _0xVM.sp++;
                        }
                    }
                    break;
                    
                case OPCODES.STRING_CONCAT:
                    if (_0xVM.sp >= 2) {
                        const str2 = _0xVM.stack.pop();
                        const str1 = _0xVM.stack.pop();
                        _0xVM.sp -= 2;
                        _0xVM.stringTable.push(String(str1) + String(str2));
                        _0xVM.stack.push(_0xVM.stringTable.length - 1);
                        _0xVM.sp++;
                    }
                    break;
                    
                case OPCODES.ARRAY_CREATE:
                    const arrSize = args[0] || 0;
                    _0xVM.arrayTable.push(new Array(arrSize).fill(0));
                    _0xVM.stack.push(_0xVM.arrayTable.length - 1);
                    _0xVM.sp++;
                    break;
                    
                case OPCODES.ARRAY_GET:
                    if (_0xVM.sp >= 2) {
                        const arrIdx = _0xVM.stack.pop();
                        const index = _0xVM.stack.pop();
                        _0xVM.sp -= 2;
                        if (arrIdx >= 0 && arrIdx < _0xVM.arrayTable.length) {
                            const arr = _0xVM.arrayTable[arrIdx];
                            _0xVM.stack.push(index >= 0 && index < arr.length ? arr[index] : 0);
                            _0xVM.sp++;
                        } else {
                            _0xVM.stack.push(0);
                            _0xVM.sp++;
                        }
                    }
                    break;
                    
                case OPCODES.ARRAY_SET:
                    if (_0xVM.sp >= 3) {
                        const value = _0xVM.stack.pop();
                        const index = _0xVM.stack.pop();
                        const arrIdx = _0xVM.stack.pop();
                        _0xVM.sp -= 3;
                        if (arrIdx >= 0 && arrIdx < _0xVM.arrayTable.length) {
                            _0xVM.arrayTable[arrIdx][index] = value;
                        }
                    }
                    break;
                    
                case OPCODES.ARRAY_LENGTH:
                    if (_0xVM.sp > 0) {
                        const arrIdx = _0xVM.stack.pop();
                        _0xVM.sp--;
                        if (arrIdx >= 0 && arrIdx < _0xVM.arrayTable.length) {
                            _0xVM.stack.push(_0xVM.arrayTable[arrIdx].length);
                            _0xVM.sp++;
                        } else {
                            _0xVM.stack.push(0);
                            _0xVM.sp++;
                        }
                    }
                    break;
                    
                case OPCODES.TIME_NOW:
                    _0xVM.stack.push(Date.now());
                    _0xVM.sp++;
                    break;
                    
                case OPCODES.CONSOLE_LOG:
                    if (_0xVM.sp > 0) {
                        const msg = _0xVM.stack.pop();
                        _0xVM.sp--;
                        console.log('[VM]', msg);
                    }
                    break;
                    
                case OPCODES.ERROR_THROW:
                    const errorMsg = args[0] || 'Unknown error';
                    throw new Error(errorMsg);
                    
                case OPCODES.ASSERT:
                    if (_0xVM.sp > 0) {
                        const condition = _0xVM.stack.pop();
                        _0xVM.sp--;
                        if (!condition) {
                            throw new Error('Assertion failed');
                        }
                    }
                    break;
                    
                case OPCODES.HALT:
                    _0xVM.running = false;
                    break;
            }
        }

        function computeVirtualHash(data) {
            let hash = 0x811C9DC5;
            const fnvPrime = 0x1000193;
            
            for (let i = 0; i < data.length; i++) {
                hash ^= data.charCodeAt(i);
                hash = (hash * fnvPrime) >>> 0;
            }
            
            return hash;
        }

        function virtualEncrypt(data) {
            let encrypted = [];
            for (let i = 0; i < data.length; i++) {
                const keyByte = _0xVM.secretKey[i % _0xVM.secretKey.length];
                const rotated = ((data.charCodeAt(i) << 3) | (data.charCodeAt(i) >>> 5));
                encrypted.push(rotated ^ keyByte);
            }
            return encrypted;
        }

        function computeChecksum(code) {
            let checksum = 0;
            for (let i = 0; i < code.length; i++) {
                checksum = (checksum + code[i]) & 0xFFFFFFFF;
            }
            return checksum;
        }

        function run(code) {
            initVM();
            _0xVM.code = code;
            _0xVM.running = true;
            _0xVM.ip = 0;
            
            while (_0xVM.running && _0xVM.instructionCount < _0xVM.maxInstructions) {
                if (_0xVM.ip >= code.length) {
                    break;
                }
                
                const instruction = decodeInstruction(code, _0xVM.ip);
                executeInstruction(instruction);
                _0xVM.ip += instruction.length;
                _0xVM.instructionCount++;
                
                if (_0xVM.breakpoints.has(_0xVM.ip)) {
                    _0xVM.running = false;
                }
            }
            
            return {
                stack: [..._0xVM.stack],
                registers: [..._0xVM.registers],
                instructionCount: _0xVM.instructionCount,
                completed: !_0xVM.running
            };
        }

        function compile(instructions) {
            let code = [];
            for (const instr of instructions) {
                const encoded = encodeInstruction(...instr);
                code.push(...encoded);
            }
            return code;
        }

        function generateVirtualizedCode(data) {
            const instructions = [
                [OPCODES.PUSH, data.length],
                [OPCODES.HASH, data],
                [OPCODES.CHECKSUM],
                [OPCODES.XOR_ROT],
                [OPCODES.VALIDATE, computeVirtualHash(data)],
                [OPCODES.HALT]
            ];
            return compile(instructions);
        }

        function createVirtualizedFunction(fn) {
            const source = fn.toString();
            const virtualCode = generateVirtualizedCode(source);
            
            return function(...args) {
                const result = run(virtualCode);
                
                if (!result.completed) {
                    throw new Error('Virtual machine execution incomplete');
                }
                
                return fn.apply(this, args);
            };
        }

        function protectFunction(fn) {
            const virtualized = createVirtualizedFunction(fn);
            
            Object.defineProperty(virtualized, 'toString', {
                value: function() {
                    return 'function() { [Virtualized Code] }';
                },
                configurable: false
            });
            
            return virtualized;
        }

        function getStatus() {
            return {
                memoryUsage: _0xVM.memory.length,
                instructionCount: _0xVM.instructionCount,
                maxInstructions: _0xVM.maxInstructions,
                stackSize: _0xVM.sp,
                running: _0xVM.running,
                version: VERSION
            };
        }

        return {
            VERSION: VERSION,
            OPCODES: OPCODES,
            run: run,
            compile: compile,
            generateVirtualizedCode: generateVirtualizedCode,
            protectFunction: protectFunction,
            getStatus: getStatus,
            initVM: initVM,
            resetVM: resetVM,
            getVMStatus: getVMStatus,
            setMaxInstructions: setMaxInstructions,
            enableStringOperations: enableStringOperations,
            enableArrayOperations: enableArrayOperations,
            enableTimeOperations: enableTimeOperations,
            addBreakpoint: addBreakpoint,
            removeBreakpoint: removeBreakpoint,
            clearBreakpoints: clearBreakpoints
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CodeVirtualization;
    } else {
        globalContext.CodeVirtualization = CodeVirtualization;
    }

})(typeof window !== 'undefined' ? window : (typeof global !== 'undefined' ? global : this));