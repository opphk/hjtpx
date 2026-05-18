/**
 * Node.js 环境加密模块测试
 * 
 * 使用方法: node backend/static/js/crypto-wasm-node-test.js
 */

const crypto = require('crypto');

const CryptoWasm = (function() {
    const VERSION = '1.0.0';
    const DEFAULT_ITERATIONS = 100000;
    const AES_KEY_LENGTH = 256;
    const IV_LENGTH = 12;
    const SALT_LENGTH = 16;

    function arrayBufferToBase64(buffer) {
        const bytes = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
        let binary = '';
        for (let i = 0; i < bytes.byteLength; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return Buffer.from(binary, 'binary').toString('base64');
    }

    function base64ToArrayBuffer(base64) {
        const binaryString = Buffer.from(base64, 'base64').toString('binary');
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }
        return bytes.buffer;
    }

    function generateRandomBytes(length) {
        const array = new Uint8Array(length);
        crypto.randomFillSync(array);
        return array;
    }

    async function pbkdf2DeriveKey(password, salt, iterations, keyLength) {
        return new Promise((resolve, reject) => {
            crypto.pbkdf2(password, salt, iterations, keyLength / 8, 'sha256', (err, derivedKey) => {
                if (err) reject(err);
                else resolve(new Uint8Array(derivedKey));
            });
        });
    }

    async function aes256GcmEncrypt(plaintext, key, options = {}) {
        options = options || {};
        
        let keyData;
        if (typeof key === 'string') {
            const salt = options.salt || generateRandomBytes(SALT_LENGTH);
            keyData = await pbkdf2DeriveKey(
                key,
                salt,
                options.iterations || DEFAULT_ITERATIONS,
                AES_KEY_LENGTH
            );
        } else {
            keyData = key instanceof Uint8Array ? key : new Uint8Array(key);
        }

        const iv = options.iv || generateRandomBytes(IV_LENGTH);

        const cipher = crypto.createCipheriv('aes-256-gcm', Buffer.from(keyData));
        const encrypted = Buffer.concat([cipher.update(plaintext, 'utf8'), cipher.final()]);
        const authTag = cipher.getAuthTag();

        const combined = Buffer.concat([iv, authTag, encrypted]);

        return {
            ciphertext: combined.toString('base64'),
            iv: iv.toString('base64'),
            salt: options.salt ? (options.salt instanceof Uint8Array ? options.salt.toString('base64') : options.salt) : null,
            algorithm: 'AES-256-GCM',
            wasmUsed: false
        };
    }

    async function aes256GcmDecrypt(encryptedData, key, options = {}) {
        options = options || {};
        
        let keyData;
        if (typeof key === 'string') {
            const salt = options.salt ? 
                (typeof options.salt === 'string' ? Buffer.from(options.salt, 'base64') : options.salt) :
                generateRandomBytes(SALT_LENGTH);
            keyData = await pbkdf2DeriveKey(
                key,
                salt,
                options.iterations || DEFAULT_ITERATIONS,
                AES_KEY_LENGTH
            );
        } else {
            keyData = key instanceof Uint8Array ? key : new Uint8Array(key);
        }

        const combined = Buffer.from(encryptedData.ciphertext, 'base64');
        const iv = combined.subarray(0, IV_LENGTH);
        const authTag = combined.subarray(IV_LENGTH, IV_LENGTH + 16);
        const ciphertext = combined.subarray(IV_LENGTH + 16);

        const decipher = crypto.createDecipheriv('aes-256-gcm', Buffer.from(keyData));
        decipher.setAuthTag(authTag);

        const decrypted = Buffer.concat([decipher.update(ciphertext), decipher.final()]);
        return decrypted.toString('utf8');
    }

    async function hashSHA256(data) {
        const hash = crypto.createHash('sha256');
        hash.update(data);
        return hash.digest('base64');
    }

    async function hmacSHA256(data, key) {
        return new Promise((resolve, reject) => {
            crypto.hmac('sha256', key, data, (err, hmac) => {
                if (err) reject(err);
                else resolve(hmac.digest('base64'));
            });
        });
    }

    async function generateKeyPair() {
        return new Promise((resolve, reject) => {
            crypto.generateKeyPair('rsa', {
                modulusLength: 2048,
                publicExponent: 0x10001,
                publicKeyEncoding: {
                    type: 'spki',
                    format: 'pem'
                },
                privateKeyEncoding: {
                    type: 'pkcs8',
                    format: 'pem'
                }
            }, (err, publicKey, privateKey) => {
                if (err) reject(err);
                else resolve({ publicKey, privateKey });
            });
        });
    }

    function getStatus() {
        return {
            wasmLoaded: false,
            wasmSupported: false,
            usingWasm: false,
            version: VERSION
        };
    }

    return {
        VERSION: VERSION,
        getStatus: getStatus,
        encrypt: aes256GcmEncrypt,
        decrypt: aes256GcmDecrypt,
        pbkdf2: pbkdf2DeriveKey,
        generateRandomBytes: generateRandomBytes,
        hashSHA256: hashSHA256,
        hmacSHA256: hmacSHA256,
        generateKeyPair: generateKeyPair,
        utils: {
            arrayBufferToBase64: arrayBufferToBase64,
            base64ToArrayBuffer: base64ToArrayBuffer
        }
    };
})();

async function runTests() {
    console.log('=== HJTPX 加密模块 Node.js 测试 ===\n');

    try {
        console.log('1. 基础状态检查:');
        const status = CryptoWasm.getStatus();
        console.log('   状态:', status);
        console.log('   ✓ Node.js 加密模块加载成功\n');

        console.log('2. 随机字节生成测试:');
        const randomBytes = CryptoWasm.generateRandomBytes(16);
        console.log('   生成的随机字节:', randomBytes.toString('hex'));
        console.log('   ✓ 随机数生成正常\n');

        console.log('3. PBKDF2 密钥派生测试:');
        const password = 'test-password';
        const salt = CryptoWasm.generateRandomBytes(16);
        const iterations = 100000;
        
        console.log('   密码:', password);
        console.log('   Salt:', salt.toString('hex'));
        console.log('   迭代次数:', iterations);
        
        const derivedKey = await CryptoWasm.pbkdf2(password, salt, iterations, 256);
        console.log('   派生的密钥:', derivedKey.toString('hex').substring(0, 32) + '...');
        console.log('   ✓ PBKDF2 密钥派生成功\n');

        console.log('4. AES-256-GCM 加密测试:');
        const plaintext = 'Hello, HJTPX! 这是测试数据。';
        const secretKey = 'captcha-trajectory-secret-key-2024';
        
        console.log('   明文:', plaintext);
        console.log('   密钥:', secretKey);
        
        const encrypted = await CryptoWasm.encrypt(plaintext, secretKey);
        console.log('   加密结果:', {
            ciphertext: encrypted.ciphertext.substring(0, 32) + '...',
            iv: encrypted.iv,
            algorithm: encrypted.algorithm,
            wasmUsed: encrypted.wasmUsed
        });
        console.log('   ✓ AES-256-GCM 加密成功\n');

        console.log('5. AES-256-GCM 解密测试:');
        const decrypted = await CryptoWasm.decrypt(encrypted, secretKey, {
            salt: encrypted.salt
        });
        console.log('   解密结果:', decrypted);
        console.log('   ✓ AES-256-GCM 解密成功\n');

        console.log('6. SHA-256 哈希测试:');
        const data = 'Hello, World!';
        const hash = await CryptoWasm.hashSHA256(data);
        console.log('   原始数据:', data);
        console.log('   SHA-256 哈希:', hash);
        console.log('   ✓ SHA-256 哈希计算成功\n');

        console.log('7. HMAC-SHA256 测试:');
        const hmacKey = 'secret-key';
        const hmac = await CryptoWasm.hmacSHA256(data, hmacKey);
        console.log('   数据:', data);
        console.log('   HMAC 密钥:', hmacKey);
        console.log('   HMAC-SHA256:', hmac);
        console.log('   ✓ HMAC-SHA256 计算成功\n');

        console.log('8. RSA 密钥对生成测试:');
        const keyPair = await CryptoWasm.generateKeyPair();
        console.log('   公钥长度:', keyPair.publicKey.length);
        console.log('   私钥长度:', keyPair.privateKey.length);
        console.log('   ✓ RSA 密钥对生成成功\n');

        console.log('9. 完整流程测试 - 模拟验证码轨迹加密:');
        const trajectory = [
            { x: 100, y: 200, t: Date.now() },
            { x: 150, y: 220, t: Date.now() + 50 },
            { x: 200, y: 250, t: Date.now() + 100 },
            { x: 250, y: 280, t: Date.now() + 150 },
            { x: 300, y: 300, t: Date.now() + 200 }
        ];
        
        console.log('   原始轨迹数据:', JSON.stringify(trajectory).substring(0, 50) + '...');
        
        const timestamp = Date.now();
        const trajectorySalt = CryptoWasm.generateRandomBytes(16);
        const trajectoryJson = JSON.stringify(trajectory);
        
        const encryptedTrajectory = await CryptoWasm.encrypt(trajectoryJson, secretKey, {
            salt: trajectorySalt
        });
        
        const signatureData = `${timestamp}:${trajectorySalt.toString('base64')}:${encryptedTrajectory.ciphertext}`;
        const signature = await CryptoWasm.hmacSHA256(signatureData, secretKey);
        
        console.log('   加密后的轨迹:', {
            timestamp: timestamp,
            salt: trajectorySalt.toString('base64').substring(0, 16) + '...',
            encrypted_data: encryptedTrajectory.ciphertext.substring(0, 32) + '...',
            signature: signature.substring(0, 32) + '...'
        });
        
        const decryptedTrajectory = await CryptoWasm.decrypt(encryptedTrajectory, secretKey, {
            salt: trajectorySalt
        });
        console.log('   解密后的轨迹:', JSON.parse(decryptedTrajectory));
        console.log('   ✓ 完整流程测试通过\n');

        console.log('=== 所有测试通过 ===');
        console.log('模块版本:', CryptoWasm.VERSION);
        console.log('注意: 此测试使用纯 Node.js crypto 模块');
        console.log('在浏览器环境中，可使用 WASM 增强版本\n');

    } catch (error) {
        console.error('❌ 测试失败:', error);
        process.exit(1);
    }
}

runTests().catch(console.error);
