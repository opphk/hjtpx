package tools

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

type ProtectionLevel int

const (
	ProtectionLevelBasic   ProtectionLevel = 1
	ProtectionLevelMedium  ProtectionLevel = 2
	ProtectionLevelAdvanced ProtectionLevel = 3
	ProtectionLevelMaximum  ProtectionLevel = 4
)

type CodeProtectionV15Config struct {
	ProtectionLevel ProtectionLevel
	EnableWASM      bool
	EnableWASMDecryption bool
	EnableControlFlowFlattening bool
	EnableIntegrityCheck bool
	EnableAntiAutomation bool
	EnableRuntimeDecryption bool
	EnableVirtualization bool
	EnablePolymorphicCode bool
	EnableCodeSplitting bool
	EnableSelfHealing bool
	KeyRotationInterval time.Duration
	EnableKeyDerivation bool
	HashAlgorithm string
	EnableAntiTampering bool
	EnableBehavioralAnalysis bool
	EnableTimingProtection bool
}

var defaultProtectionConfig = CodeProtectionV15Config{
	ProtectionLevel: ProtectionLevelAdvanced,
	EnableWASM: true,
	EnableWASMDecryption: true,
	EnableControlFlowFlattening: true,
	EnableIntegrityCheck: true,
	EnableAntiAutomation: true,
	EnableRuntimeDecryption: true,
	EnableVirtualization: true,
	EnablePolymorphicCode: true,
	EnableCodeSplitting: true,
	EnableSelfHealing: false,
	KeyRotationInterval: 24 * time.Hour,
	EnableKeyDerivation: true,
	HashAlgorithm: "sha256",
	EnableAntiTampering: true,
	EnableBehavioralAnalysis: true,
	EnableTimingProtection: true,
}

type CodeProtectionV15 struct {
	config      CodeProtectionV15Config
	keyManager  *KeyManager
	obfuscator  *FlowObfuscator
	integrity   *IntegrityService
	detector    *AutomationDetector
	crypto      *ProtectionCryptoService
	mu          sync.RWMutex
	version     string
}

type ProtectionCryptoService struct {
	masterKey []byte
	nonceMap  map[string]bool
	mu        sync.RWMutex
}

func NewProtectionCryptoService(key []byte) *ProtectionCryptoService {
	if len(key) == 0 {
		key = []byte("hjtpx-v15-master-key-2024")
	}
	return &ProtectionCryptoService{
		masterKey: key,
		nonceMap:  make(map[string]bool),
	}
}

func (s *ProtectionCryptoService) deriveKey(salt []byte, info string) ([]byte, error) {
	if len(salt) == 0 {
		salt = []byte("hjtpx-salt-v15")
	}
	
	h := sha256.New()
	h.Write(s.masterKey)
	h.Write(salt)
	h.Write([]byte(info))
	
	derived := h.Sum(nil)
	return derived[:32], nil
}

func (s *ProtectionCryptoService) encryptWithDerivedKey(plaintext []byte, info string) ([]byte, error) {
	derivedKey, err := s.deriveKey(nil, info)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (s *ProtectionCryptoService) decryptWithDerivedKey(ciphertext []byte, info string) ([]byte, error) {
	derivedKey, err := s.deriveKey(nil, info)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func (s *ProtectionCryptoService) EncryptCode(code string) (string, error) {
	encrypted, err := s.encryptWithDerivedKey([]byte(code), "code-encryption-v15")
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func (s *ProtectionCryptoService) DecryptCode(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	decrypted, err := s.decryptWithDerivedKey(data, "code-encryption-v15")
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

func (s *ProtectionCryptoService) markNonce(nonce string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonceMap[nonce] = true
}

func (s *ProtectionCryptoService) isNonceUsed(nonce string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nonceMap[nonce]
}

func NewCodeProtectionV15(config ...CodeProtectionV15Config) *CodeProtectionV15 {
	cfg := defaultProtectionConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	cp := &CodeProtectionV15{
		config:     cfg,
		keyManager: NewKeyManager(cfg.KeyRotationInterval),
		obfuscator: NewFlowObfuscator(),
		integrity:  NewIntegrityService(),
		detector:   NewAutomationDetector(),
		crypto:     NewProtectionCryptoService(nil),
		version:    "15.0.0",
	}

	return cp
}

func (cp *CodeProtectionV15) ProtectCode(code string) (string, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.config.EnableWASM && cp.config.EnableWASMDecryption {
		code = cp.addWASMDecryption(code)
	}

	if cp.config.EnableControlFlowFlattening {
		obfuscated, err := cp.obfuscator.ObfuscateControlFlow(code)
		if err == nil {
			code = obfuscated
		}
	}

	if cp.config.EnableRuntimeDecryption {
		code = cp.addRuntimeDecryption(code)
	}

	if cp.config.EnableIntegrityCheck {
		code = cp.addIntegrityCheck(code)
	}

	if cp.config.EnableAntiAutomation {
		code = cp.addAntiAutomationProtection(code)
	}

	if cp.config.EnableTimingProtection {
		code = cp.addTimingProtection(code)
	}

	if cp.config.EnableVirtualization {
		code = cp.addVirtualization(code)
	}

	if cp.config.EnablePolymorphicCode {
		code = cp.addPolymorphicProtection(code)
	}

	code = cp.wrapInProtection(code)

	return code, nil
}

func (cp *CodeProtectionV15) addWASMDecryption(code string) string {
	wasmLoader := `
(function(){
	var _0xw = window._0xw || {};
	_0xw.decrypt = function(_0xc, _0xk) {
		var _0xr = atob(_0xc);
		var _0xm = new Uint8Array(_0xr.length);
		for(var _0xi = 0; _0xi < _0xr.length; _0xi++) {
			_0xm[_0xi] = _0xr.charCodeAt(_0xi);
		}
		var _0xs = _0xm.slice(0, 32);
		var _0xd = _0xm.slice(32);
		var _0xk32 = new Uint8Array(32);
		for(var _0xi = 0; _0xi < 32; _0xi++) {
			_0xk32[_0xi] = _0xs[_0xi] ^ (_0xk ? _0xk.charCodeAt(_0xi % _0xk.length) : 42);
		}
		return String.fromCharCode.apply(null, new Uint8Array(_0xd.map(function(b, i) {
			return b ^ _0xk32[i % 32];
		})));
	};
	window._0xw = _0xw;
})();
`
	return wasmLoader + code
}

func (cp *CodeProtectionV15) addRuntimeDecryption(code string) string {
	key := cp.keyManager.GetCurrentKey()
	keyHex := hex.EncodeToString(key)
	
	encrypted, err := cp.crypto.EncryptCode(code)
	if err != nil {
		return code
	}

	loader := fmt.Sprintf(`
(function(){
	var _0xke = "%s";
	var _0xec = "%s";
	var _0xdk = [];
	for(var _0xi = 0; _0xi < 32; _0xi++) {
		_0xdk[_0xi] = _0xke.charCodeAt(_0xi %% _0xke.length) ^ ((_0xi * 7 + 13) %% 256);
	}
	var _0xct = atob(_0xec);
	var _0xpt = [];
	for(var _0xi = 0; _0xi < _0xct.length; _0xi++) {
		_0xpt[_0xi] = _0xct.charCodeAt(_0xi) ^ _0xdk[_0xi %% 32];
	}
	var _0xdp = String.fromCharCode.apply(null, _0xpt);
	try {
		eval(_0xdp);
	} catch(_0xe) {
		console.error("Runtime decryption failed");
	}
})();
`, keyHex, encrypted)

	return loader
}

func (cp *CodeProtectionV15) addIntegrityCheck(code string) string {
	hash := cp.integrity.CalculateHash(code)
	hashB64 := base64.StdEncoding.EncodeToString([]byte(hash))
	
	checker := fmt.Sprintf(`
(function(){
	var _0xh = "%s";
	window.__IntegrityHash = window.__IntegrityHash || {};
	window.__IntegrityHash.verify = function() {
		var _0xc = document.querySelector('script[data-protected]');
		if(_0xc) {
			var _0xch = "%s";
			var _0xh2 = "%s";
			if(_0xc.textContent.length > 0) {
				return true;
			}
		}
		return true;
	};
	window.__IntegrityHash.getHash = function() { return "%s"; };
})();
`, hashB64, hash, hash, hash)

	return checker + code
}

func (cp *CodeProtectionV15) addAntiAutomationProtection(code string) string {
	detections := cp.detector.GenerateDetectionCode()
	return detections + code
}

func (cp *CodeProtectionV15) addTimingProtection(code string) string {
	timingProtection := `
(function(){
	var _0xst = Date.now();
	var _0xok = true;
	var _0xcl = 0;
	setInterval(function() {
		var _0xet = Date.now();
		var _0xdt = _0xet - _0xst;
		if(_0xdt > 100 && _0xok) {
			_0xcl++;
			if(_0xcl > 3) {
				document.documentElement.style.display = 'none';
				document.body.innerHTML = '<div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#000;color:#fff;display:flex;justify-content:center;align-items:center;font-family:Arial,sans-serif;z-index:2147483647;"><h1>Access Denied</h1></div>';
			}
			_0xok = false;
			setTimeout(function() { _0xok = true; _0xst = Date.now(); }, 5000);
		}
	}, 1000);
})();
`
	return timingProtection + code
}

func (cp *CodeProtectionV15) addVirtualization(code string) string {
	vmCode := `
(function(){
	var _0xvm = window._0xvm = window._0xvm || {};
	_0xvm.handlers = {};
	_0xvm.register = function(_0xid, _0xfn) {
		_0xvm.handlers[_0xid] = _0xfn;
	};
	_0xvm.execute = function(_0xid, _0xarg) {
		if(_0xvm.handlers[_0xid]) {
			return _0xvm.handlers[_0xid](_0xarg);
		}
		return null;
	};
	_0xvm.wrap = function(_0xfn, _0xid) {
		return function(_0xarg) {
			return _0xvm.execute(_0xid, _0xarg);
		};
	};
	window._0xvm = _0xvm;
})();
`
	return vmCode + code
}

func (cp *CodeProtectionV15) addPolymorphicProtection(code string) string {
	polymorphicCode := `
(function(){
	var _0xpoly = window._0xpoly = window._0xpoly || {};
	_0xpoly.transform = function(_0xc) {
		var _0xt = _0xc;
		var _0xreplacements = [
			[/\\bconsole\\.log\\b/g, 'void 0'],
			[/\\bconsole\\.warn\\b/g, 'void 0'],
			[/\\bconsole\\.error\\b/g, 'void 0'],
			[/\\bdebugger\\b/g, ';'],
		];
		_0xreplacements.forEach(function(_0xr) {
			_0xt = _0xt.replace(_0xr[0], _0xr[1]);
		});
		return _0xt;
	};
	window._0xpoly = _0xpoly;
})();
`
	return polymorphicCode + code
}

func (cp *CodeProtectionV15) wrapInProtection(code string) string {
	wrapped := fmt.Sprintf(`
(function(){
	"use strict";
	var _0xp15 = {
		version: "%s",
		startTime: Date.now(),
		initialized: true
	};
	%s
	window.__ProtectionV15 = _0xp15;
})();
`, cp.version, code)
	return wrapped
}

func (cp *CodeProtectionV15) ProtectWithLevel(code string, level ProtectionLevel) (string, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	switch level {
	case ProtectionLevelBasic:
		return cp.protectBasic(code)
	case ProtectionLevelMedium:
		return cp.protectMedium(code)
	case ProtectionLevelAdvanced:
		return cp.protectAdvanced(code)
	case ProtectionLevelMaximum:
		return cp.protectMaximum(code)
	default:
		return cp.protectAdvanced(code)
	}
}

func (cp *CodeProtectionV15) protectBasic(code string) (string, error) {
	code = cp.addIntegrityCheck(code)
	code = cp.wrapInProtection(code)
	return code, nil
}

func (cp *CodeProtectionV15) protectMedium(code string) (string, error) {
	code = cp.addTimingProtection(code)
	code = cp.addIntegrityCheck(code)
	if cp.config.EnableAntiAutomation {
		code = cp.addAntiAutomationProtection(code)
	}
	code = cp.wrapInProtection(code)
	return code, nil
}

func (cp *CodeProtectionV15) protectAdvanced(code string) (string, error) {
	code = cp.addWASMDecryption(code)
	code = cp.addTimingProtection(code)
	code = cp.addIntegrityCheck(code)
	code = cp.addAntiAutomationProtection(code)
	code = cp.addVirtualization(code)
	code = cp.wrapInProtection(code)
	return code, nil
}

func (cp *CodeProtectionV15) protectMaximum(code string) (string, error) {
	code = cp.addWASMDecryption(code)
	if cp.config.EnableControlFlowFlattening {
		obfuscated, err := cp.obfuscator.ObfuscateControlFlow(code)
		if err == nil {
			code = obfuscated
		}
	}
	code = cp.addRuntimeDecryption(code)
	code = cp.addTimingProtection(code)
	code = cp.addIntegrityCheck(code)
	code = cp.addAntiAutomationProtection(code)
	code = cp.addVirtualization(code)
	code = cp.addPolymorphicProtection(code)
	code = cp.wrapInProtection(code)
	return code, nil
}

func (cp *CodeProtectionV15) VerifyIntegrity(code string) (bool, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	expectedHash := cp.integrity.CalculateHash(code)
	verified := cp.integrity.VerifyHash(code, expectedHash)
	return verified, nil
}

func (cp *CodeProtectionV15) DetectAutomation() (bool, map[string]interface{}, error) {
	return cp.detector.DetectAutomation()
}

func (cp *CodeProtectionV15) GetVersion() string {
	return cp.version
}

func (cp *CodeProtectionV15) GetConfig() CodeProtectionV15Config {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.config
}

func (cp *CodeProtectionV15) UpdateConfig(config CodeProtectionV15Config) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.config = config
}

func (cp *CodeProtectionV15) RotateKey() error {
	return cp.keyManager.RotateKey()
}

func (cp *CodeProtectionV15) EncryptData(data []byte) ([]byte, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.crypto.encryptWithDerivedKey(data, "data-encryption-v15")
}

func (cp *CodeProtectionV15) DecryptData(data []byte) ([]byte, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.crypto.decryptWithDerivedKey(data, "data-encryption-v15")
}

func (cp *CodeProtectionV15) GenerateProtectedScript(scriptContent string, level ProtectionLevel) (string, error) {
	protected, err := cp.ProtectWithLevel(scriptContent, level)
	if err != nil {
		return "", err
	}

	script := fmt.Sprintf(`<script data-protected="true" data-version="%s" data-level="%d">
%s
</script>`, cp.version, level, protected)

	return script, nil
}

func (cp *CodeProtectionV15) BatchProtect(codes []string, level ProtectionLevel) ([]string, error) {
	results := make([]string, len(codes))
	for i, code := range codes {
		protected, err := cp.ProtectWithLevel(code, level)
		if err != nil {
			return nil, fmt.Errorf("failed to protect code at index %d: %w", i, err)
		}
		results[i] = protected
	}
	return results, nil
}

func (cp *CodeProtectionV15) AnalyzeProtection(code string) (map[string]bool, error) {
	analysis := make(map[string]bool)

	analysis["hasIntegrityCheck"] = strings.Contains(code, "__IntegrityHash")
	analysis["hasTimingProtection"] = strings.Contains(code, "_0xst") || strings.Contains(code, "_0xcl")
	analysis["hasAntiAutomation"] = strings.Contains(code, "_0xauto") || strings.Contains(code, "automation")
	analysis["hasVirtualization"] = strings.Contains(code, "_0xvm")
	analysis["hasPolymorphic"] = strings.Contains(code, "_0xpoly")
	analysis["hasWASM"] = strings.Contains(code, "_0xw")
	analysis["hasRuntimeDecryption"] = strings.Contains(code, "_0xdk") || strings.Contains(code, "_0xdp")
	analysis["isWrapped"] = strings.Contains(code, "__ProtectionV15")

	return analysis, nil
}

type ProtectionReport struct {
	Version           string                 `json:"version"`
	ProtectionLevel   ProtectionLevel        `json:"protection_level"`
	Features          map[string]bool       `json:"features"`
	IntegrityStatus  bool                  `json:"integrity_status"`
	AutomationStatus bool                  `json:"automation_status"`
	GeneratedAt       time.Time             `json:"generated_at"`
}

func (cp *CodeProtectionV15) GenerateReport() *ProtectionReport {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	report := &ProtectionReport{
		Version:         cp.version,
		ProtectionLevel: cp.config.ProtectionLevel,
		Features:        make(map[string]bool),
		GeneratedAt:     time.Now(),
	}

	report.Features["wasm"] = cp.config.EnableWASM
	report.Features["control_flow"] = cp.config.EnableControlFlowFlattening
	report.Features["integrity"] = cp.config.EnableIntegrityCheck
	report.Features["anti_automation"] = cp.config.EnableAntiAutomation
	report.Features["runtime_decryption"] = cp.config.EnableRuntimeDecryption
	report.Features["virtualization"] = cp.config.EnableVirtualization
	report.Features["polymorphic"] = cp.config.EnablePolymorphicCode
	report.Features["timing_protection"] = cp.config.EnableTimingProtection
	report.Features["anti_tampering"] = cp.config.EnableAntiTampering

	return report
}
