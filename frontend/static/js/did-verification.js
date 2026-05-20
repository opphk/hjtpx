const DIDVerification = (function() {
    'use strict';

    const CONFIG = {
        API_BASE_URL: '/api/v1/did',
        TIMEOUT: 30000,
        RETRY_COUNT: 3,
        WS_ENDPOINT: 'wss://demos.example.com/did'
    };

    const DID_METHODS = {
        WEB: 'web',
        ETHR: 'ethr',
        ION: 'ion',
        SOL: 'sol'
    };

    const CREDENTIAL_TYPES = {
        IDENTITY: 'IdentityCredential',
        EMAIL: 'EmailCredential',
        PHONE: 'PhoneCredential',
        AGE_OVER: 'AgeOverCredential',
        MEMBERSHIP: 'MembershipCredential'
    };

    class DIDVerifier {
        constructor() {
            this.didRegistry = null;
            this.vcService = null;
            this.zkProver = null;
            this.eventListeners = {};
        }

        async initialize() {
            try {
                this.didRegistry = await this.loadDIDRegistry();
                this.vcService = await this.loadVCService();
                this.emit('initialized', { status: 'ready' });
                return true;
            } catch (error) {
                this.emit('error', { type: 'init', error: error.message });
                return false;
            }
        }

        async loadDIDRegistry() {
            const response = await fetch(`${CONFIG.API_BASE_URL}/registry`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) {
                throw new Error('Failed to load DID registry');
            }

            return await response.json();
        }

        async loadVCService() {
            const response = await fetch(`${CONFIG.API_BASE_URL}/vc/service`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) {
                throw new Error('Failed to load VC service');
            }

            return await response.json();
        }

        async createDID(method, methodSpecificId, publicKey, services = []) {
            try {
                const didString = `did:${method}:${methodSpecificId}`;
                
                const payload = {
                    method: method,
                    methodSpecificId: methodSpecificId,
                    publicKey: this.arrayBufferToBase64(publicKey),
                    services: services
                };

                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/create`,
                    {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(payload)
                    }
                );

                if (!response.ok) {
                    throw new Error('Failed to create DID');
                }

                const result = await response.json();
                
                this.emit('didCreated', { 
                    did: didString, 
                    document: result.document 
                });
                
                return result;
            } catch (error) {
                this.emit('error', { type: 'createDID', error: error.message });
                throw error;
            }
        }

        async resolveDID(didString) {
            try {
                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/resolve?did=${encodeURIComponent(didString)}`,
                    {
                        method: 'GET',
                        headers: {
                            'Content-Type': 'application/json'
                        }
                    }
                );

                if (!response.ok) {
                    throw new Error('Failed to resolve DID');
                }

                const document = await response.json();
                
                this.emit('didResolved', { 
                    did: didString, 
                    document: document 
                });
                
                return document;
            } catch (error) {
                this.emit('error', { type: 'resolveDID', error: error.message });
                throw error;
            }
        }

        async updateDID(didString, updates) {
            try {
                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/update`,
                    {
                        method: 'PUT',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            did: didString,
                            updates: updates
                        })
                    }
                );

                if (!response.ok) {
                    throw new Error('Failed to update DID');
                }

                this.emit('didUpdated', { did: didString });
                
                return true;
            } catch (error) {
                this.emit('error', { type: 'updateDID', error: error.message });
                throw error;
            }
        }

        async verifyDIDAuthentication(didString, challenge, signature) {
            try {
                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/verify/auth`,
                    {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            did: didString,
                            challenge: challenge,
                            signature: this.arrayBufferToBase64(signature)
                        })
                    }
                );

                const result = await response.json();
                
                this.emit('authVerified', { 
                    did: didString, 
                    valid: result.valid 
                });
                
                return result.valid;
            } catch (error) {
                this.emit('error', { type: 'verifyAuth', error: error.message });
                return false;
            }
        }
    }

    class VCCredentialManager {
        constructor() {
            this.credentials = new Map();
        }

        async issueCredential(issuerDID, holderDID, credentialType, claims, expirationDate = null) {
            try {
                const payload = {
                    issuerDID: issuerDID,
                    holderDID: holderDID,
                    credentialType: credentialType,
                    claims: claims,
                    expirationDate: expirationDate
                };

                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/vc/issue`,
                    {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(payload)
                    }
                );

                if (!response.ok) {
                    throw new Error('Failed to issue credential');
                }

                const credential = await response.json();
                
                this.credentials.set(credential.id, credential);
                
                return credential;
            } catch (error) {
                this.emit('error', { type: 'issueCredential', error: error.message });
                throw error;
            }
        }

        async verifyCredential(credentialId, options = {}) {
            try {
                const {
                    checkExpired = true,
                    checkRevoked = true,
                    checkProof = true
                } = options;

                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/vc/verify`,
                    {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            credentialId: credentialId,
                            checkExpired: checkExpired,
                            checkRevoked: checkRevoked,
                            checkProof: checkProof
                        })
                    }
                );

                const result = await response.json();
                
                return result;
            } catch (error) {
                this.emit('error', { type: 'verifyCredential', error: error.message });
                throw error;
            }
        }

        async createPresentation(holderDID, credentialIds, challenge, domain) {
            try {
                const credentials = credentialIds
                    .map(id => this.credentials.get(id))
                    .filter(c => c !== undefined);

                if (credentials.length === 0) {
                    throw new Error('No valid credentials found');
                }

                const payload = {
                    holderDID: holderDID,
                    credentials: credentials,
                    challenge: challenge,
                    domain: domain
                };

                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/vc/presentation/create`,
                    {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(payload)
                    }
                );

                if (!response.ok) {
                    throw new Error('Failed to create presentation');
                }

                const presentation = await response.json();
                
                return presentation;
            } catch (error) {
                this.emit('error', { type: 'createPresentation', error: error.message });
                throw error;
            }
        }

        async verifyPresentation(presentation, challenge, domain) {
            try {
                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/vc/presentation/verify`,
                    {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            presentation: presentation,
                            challenge: challenge,
                            domain: domain
                        })
                    }
                );

                const result = await response.json();
                
                return result;
            } catch (error) {
                this.emit('error', { type: 'verifyPresentation', error: error.message });
                throw error;
            }
        }

        async revokeCredential(credentialId, reason) {
            try {
                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/vc/revoke`,
                    {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            credentialId: credentialId,
                            reason: reason
                        })
                    }
                );

                if (!response.ok) {
                    throw new Error('Failed to revoke credential');
                }

                this.credentials.delete(credentialId);
                
                return true;
            } catch (error) {
                this.emit('error', { type: 'revokeCredential', error: error.message });
                throw error;
            }
        }
    }

    class ZKProofGenerator {
        constructor() {
            this.circuits = new Map();
        }

        async generateProof(did, claimTypes, challenge) {
            try {
                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/zk/prove`,
                    {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            did: did,
                            claimTypes: claimTypes,
                            challenge: challenge
                        })
                    }
                );

                if (!response.ok) {
                    throw new Error('Failed to generate ZK proof');
                }

                const proof = await response.json();
                
                return proof;
            } catch (error) {
                this.emit('error', { type: 'generateProof', error: error.message });
                throw error;
            }
        }

        async verifyProof(proof) {
            try {
                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/zk/verify`,
                    {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({ proof: proof })
                    }
                );

                const result = await response.json();
                
                return result.valid;
            } catch (error) {
                this.emit('error', { type: 'verifyProof', error: error.message });
                return false;
            }
        }

        async loadCircuit(circuitId) {
            try {
                const response = await fetch(
                    `${CONFIG.API_BASE_URL}/zk/circuit/${encodeURIComponent(circuitId)}`,
                    {
                        method: 'GET',
                        headers: {
                            'Content-Type': 'application/json'
                        }
                    }
                );

                if (!response.ok) {
                    throw new Error('Failed to load circuit');
                }

                const circuit = await response.json();
                
                this.circuits.set(circuitId, circuit);
                
                return circuit;
            } catch (error) {
                this.emit('error', { type: 'loadCircuit', error: error.message });
                throw error;
            }
        }
    }

    class CrossChainBridge {
        constructor() {
            this.connections = new Map();
        }

        async configureChain(chainId, config) {
            try {
                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/bridge/chain/configure`,
                    {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            chainId: chainId,
                            config: config
                        })
                    }
                );

                if (!response.ok) {
                    throw new Error('Failed to configure chain');
                }

                const result = await response.json();
                
                this.connections.set(chainId, result);
                
                return result;
            } catch (error) {
                this.emit('error', { type: 'configureChain', error: error.message });
                throw error;
            }
        }

        async createBridge(sourceChain, targetChain) {
            try {
                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/bridge/connect`,
                    {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            sourceChain: sourceChain,
                            targetChain: targetChain
                        })
                    }
                );

                if (!response.ok) {
                    throw new Error('Failed to create bridge');
                }

                const bridge = await response.json();
                
                const bridgeKey = `${sourceChain}->${targetChain}`;
                this.connections.set(bridgeKey, bridge);
                
                return bridge;
            } catch (error) {
                this.emit('error', { type: 'createBridge', error: error.message });
                throw error;
            }
        }

        async syncDID(did, sourceChain, targetChain) {
            try {
                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/bridge/sync`,
                    {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            did: did,
                            sourceChain: sourceChain,
                            targetChain: targetChain
                        })
                    }
                );

                if (!response.ok) {
                    throw new Error('Failed to sync DID');
                }

                return true;
            } catch (error) {
                this.emit('error', { type: 'syncDID', error: error.message });
                throw error;
            }
        }

        async getCrossChainState(did) {
            try {
                const response = await this.retryFetch(
                    `${CONFIG.API_BASE_URL}/bridge/state?did=${encodeURIComponent(did)}`,
                    {
                        method: 'GET',
                        headers: {
                            'Content-Type': 'application/json'
                        }
                    }
                );

                if (!response.ok) {
                    throw new Error('Failed to get cross-chain state');
                }

                const states = await response.json();
                
                return states;
            } catch (error) {
                this.emit('error', { type: 'getCrossChainState', error: error.message });
                throw error;
            }
        }
    }

    const verifier = new DIDVerifier();
    const vcManager = new VCCredentialManager();
    const zkProver = new ZKProofGenerator();
    const bridge = new CrossChainBridge();

    function emit(event, data) {
        const listeners = verifier.eventListeners[event] || [];
        listeners.forEach(callback => callback(data));
    }

    function on(event, callback) {
        if (!verifier.eventListeners[event]) {
            verifier.eventListeners[event] = [];
        }
        verifier.eventListeners[event].push(callback);
    }

    function off(event, callback) {
        const listeners = verifier.eventListeners[event] || [];
        verifier.eventListeners[event] = listeners.filter(cb => cb !== callback);
    }

    async function retryFetch(url, options, retries = CONFIG.RETRY_COUNT) {
        let lastError;

        for (let i = 0; i < retries; i++) {
            try {
                const controller = new AbortController();
                const timeout = setTimeout(() => controller.abort(), CONFIG.TIMEOUT);

                const response = await fetch(url, {
                    ...options,
                    signal: controller.signal
                });

                clearTimeout(timeout);

                if (response.ok) {
                    return response;
                }

                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            } catch (error) {
                lastError = error;
                
                if (i < retries - 1) {
                    await new Promise(resolve => setTimeout(resolve, 1000 * Math.pow(2, i)));
                }
            }
        }

        throw lastError;
    }

    function arrayBufferToBase64(buffer) {
        const bytes = new Uint8Array(buffer);
        let binary = '';
        
        for (let i = 0; i < bytes.byteLength; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        
        return btoa(binary);
    }

    function base64ToArrayBuffer(base64) {
        const binary = atob(base64);
        const bytes = new Uint8Array(binary.length);
        
        for (let i = 0; i < binary.length; i++) {
            bytes[i] = binary.charCodeAt(i);
        }
        
        return bytes.buffer;
    }

    async function generateKeyPair() {
        try {
            const keyPair = await crypto.subtle.generateKey(
                {
                    name: 'ECDSA',
                    namedCurve: 'P-256'
                },
                true,
                ['sign', 'verify']
            );

            return keyPair;
        } catch (error) {
            this.emit('error', { type: 'generateKeyPair', error: error.message });
            throw error;
        }
    }

    async function signData(data, privateKey) {
        try {
            const encodedData = new TextEncoder().encode(JSON.stringify(data));
            
            const signature = await crypto.subtle.sign(
                {
                    name: 'ECDSA',
                    hash: { name: 'SHA-256' }
                },
                privateKey,
                encodedData
            );

            return signature;
        } catch (error) {
            this.emit('error', { type: 'signData', error: error.message });
            throw error;
        }
    }

    return {
        initialize: () => verifier.initialize(),
        
        createDID: (method, methodSpecificId, publicKey, services) => 
            verifier.createDID(method, methodSpecificId, publicKey, services),
        
        resolveDID: (didString) => verifier.resolveDID(didString),
        
        updateDID: (didString, updates) => verifier.updateDID(didString, updates),
        
        verifyDIDAuthentication: (didString, challenge, signature) => 
            verifier.verifyDIDAuthentication(didString, challenge, signature),
        
        issueCredential: (issuerDID, holderDID, credentialType, claims, expirationDate) =>
            vcManager.issueCredential(issuerDID, holderDID, credentialType, claims, expirationDate),
        
        verifyCredential: (credentialId, options) =>
            vcManager.verifyCredential(credentialId, options),
        
        createPresentation: (holderDID, credentialIds, challenge, domain) =>
            vcManager.createPresentation(holderDID, credentialIds, challenge, domain),
        
        verifyPresentation: (presentation, challenge, domain) =>
            vcManager.verifyPresentation(presentation, challenge, domain),
        
        revokeCredential: (credentialId, reason) =>
            vcManager.revokeCredential(credentialId, reason),
        
        generateZKProof: (did, claimTypes, challenge) =>
            zkProver.generateProof(did, claimTypes, challenge),
        
        verifyZKProof: (proof) => zkProver.verifyProof(proof),
        
        loadCircuit: (circuitId) => zkProver.loadCircuit(circuitId),
        
        configureChain: (chainId, config) => bridge.configureChain(chainId, config),
        
        createBridge: (sourceChain, targetChain) => bridge.createBridge(sourceChain, targetChain),
        
        syncDID: (did, sourceChain, targetChain) => bridge.syncDID(did, sourceChain, targetChain),
        
        getCrossChainState: (did) => bridge.getCrossChainState(did),
        
        generateKeyPair: () => generateKeyPair(),
        
        signData: (data, privateKey) => signData(data, privateKey),
        
        on: (event, callback) => on(event, callback),
        
        off: (event, callback) => off(event, callback),
        
        CONFIG: CONFIG,
        
        DID_METHODS: DID_METHODS,
        
        CREDENTIAL_TYPES: CREDENTIAL_TYPES
    };
})();

if (typeof module !== 'undefined' && module.exports) {
    module.exports = DIDVerification;
}
