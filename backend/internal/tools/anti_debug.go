package tools

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"
)

type AntiDebugConfig struct {
	EnableDevToolsDetection   bool
	EnableMemoryProtection    bool
	EnableIntegrityCheck      bool
	EnableSelfDestruct        bool
	EnableAntiTampering       bool
	EnableAutomationDetection bool
	CheckInterval             time.Duration
}

var defaultAntiDebugConfig = AntiDebugConfig{
	EnableDevToolsDetection:   true,
	EnableMemoryProtection:    true,
	EnableIntegrityCheck:      true,
	EnableSelfDestruct:        true,
	EnableAntiTampering:       true,
	EnableAutomationDetection: true,
	CheckInterval:             2000 * time.Millisecond,
}

type AntiDebug struct {
	config         AntiDebugConfig
	integrityHash  string
	signatureKey   []byte
	nonceCache     map[string]int64
	mu             sync.RWMutex
	isCompromised  bool
	detectCount    int
	lastDetectTime time.Time
}

func NewAntiDebug(config ...AntiDebugConfig) *AntiDebug {
	cfg := defaultAntiDebugConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	key := make([]byte, 32)
	io.ReadFull(rand.Reader, key)

	return &AntiDebug{
		config:       cfg,
		signatureKey: key,
		nonceCache:   make(map[string]int64),
		isCompromised: false,
		detectCount:   0,
	}
}

func (a *AntiDebug) SetSignatureKey(key []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.signatureKey = key
}

func (a *AntiDebug) GetSignatureKey() []byte {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.signatureKey
}

func (a *AntiDebug) GenerateIntegrityHash(code string) string {
	hash := sha256.Sum256([]byte(code + string(a.signatureKey)))
	return hex.EncodeToString(hash[:])
}

func (a *AntiDebug) SetIntegrityHash(hash string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.integrityHash = hash
}

func (a *AntiDebug) VerifyIntegrity(code string) bool {
	if a.integrityHash == "" {
		return true
	}

	computedHash := a.GenerateIntegrityHash(code)
	return subtle.ConstantTimeCompare([]byte(computedHash), []byte(a.integrityHash)) == 1
}

func (a *AntiDebug) RecordDetection() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.detectCount++
	a.lastDetectTime = time.Now()
}

func (a *AntiDebug) GetDetectionCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.detectCount
}

func (a *AntiDebug) IsCompromised() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isCompromised
}

func (a *AntiDebug) MarkCompromised() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isCompromised = true
}

func (a *AntiDebug) GenerateChallenge() (string, error) {
	challenge := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, challenge); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(challenge), nil
}

func (a *AntiDebug) ValidateChallengeResponse(challenge, response string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	ts, exists := a.nonceCache[challenge]
	if !exists {
		return false
	}

	delete(a.nonceCache, challenge)

	if time.Since(time.Unix(ts, 0)) > 30*time.Second {
		return false
	}

	expected := a.computeResponse(challenge)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(response)) == 1
}

func (a *AntiDebug) computeResponse(challenge string) string {
	h := hmac.New(sha256.New, a.signatureKey)
	h.Write([]byte(challenge))
	return hex.EncodeToString(h.Sum(nil))
}

func (a *AntiDebug) GenerateToken() (string, error) {
	nonce := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	timestamp := make([]byte, 8)
	binary.PutVarint(timestamp, time.Now().Unix())

	data := append(nonce, timestamp...)
	signature := a.computeResponse(base64.StdEncoding.EncodeToString(data))

	token := base64.StdEncoding.EncodeToString(append(data, []byte(signature)...))
	return token, nil
}

func (a *AntiDebug) ValidateToken(token string) bool {
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil || len(decoded) < 24 {
		return false
	}

	data := decoded[:len(decoded)-64]
	signature := string(decoded[len(decoded)-64:])

	expectedSig := a.computeResponse(base64.StdEncoding.EncodeToString(data))
	if subtle.ConstantTimeCompare([]byte(expectedSig), []byte(signature)) != 1 {
		return false
	}

	timestampBytes := data[16 : 16+8]
	ts, _ := binary.Varint(timestampBytes)
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return false
	}

	return true
}

func (a *AntiDebug) GenerateCodeSignature(code string, timestamp int64) string {
	h := hmac.New(sha256.New, a.signatureKey)
	h.Write([]byte(code))
	h.Write([]byte(fmt.Sprintf("%d", timestamp)))
	return hex.EncodeToString(h.Sum(nil))
}

func (a *AntiDebug) VerifyCodeSignature(code string, timestamp int64, signature string) bool {
	expected := a.GenerateCodeSignature(code, timestamp)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 1
}

type IntegrityChecker struct {
	hashes      map[string]string
	keys        map[string][]byte
	mu          sync.RWMutex
	maxEntries  int
}

func NewIntegrityChecker() *IntegrityChecker {
	return &IntegrityChecker{
		hashes:     make(map[string]string),
		keys:       make(map[string][]byte),
		maxEntries: 1000,
	}
}

func (ic *IntegrityChecker) RegisterCode(name, code string) error {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	if len(ic.hashes) >= ic.maxEntries {
		ic.cleanup()
	}

	key := make([]byte, 32)
	io.ReadFull(rand.Reader, key)

	hash := sha256.Sum256(append([]byte(code), key...))
	hashStr := hex.EncodeToString(hash[:])

	ic.hashes[name] = hashStr
	ic.keys[name] = key

	return nil
}

func (ic *IntegrityChecker) VerifyCode(name, code string) (bool, error) {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	hashStr, exists := ic.hashes[name]
	if !exists {
		return false, errors.New("code not registered")
	}

	key, exists := ic.keys[name]
	if !exists {
		return false, errors.New("key not found")
	}

	computedHash := sha256.Sum256(append([]byte(code), key...))
	computedStr := hex.EncodeToString(computedHash[:])

	return subtle.ConstantTimeCompare([]byte(hashStr), []byte(computedStr)) == 1, nil
}

func (ic *IntegrityChecker) cleanup() {
	count := len(ic.hashes) / 4
	removed := 0
	for name := range ic.hashes {
		if removed >= count {
			break
		}
		delete(ic.hashes, name)
		delete(ic.keys, name)
		removed++
	}
}

type MemoryProtection struct {
	protectedFunctions map[string]bool
	originalCode      map[string]string
	mu                sync.RWMutex
}

func NewMemoryProtection() *MemoryProtection {
	return &MemoryProtection{
		protectedFunctions: make(map[string]bool),
		originalCode:       make(map[string]string),
	}
}

func (mp *MemoryProtection) ProtectFunction(name string, code string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.protectedFunctions[name] = true
	mp.originalCode[name] = code
}

func (mp *MemoryProtection) IsFunctionProtected(name string) bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.protectedFunctions[name]
}

func (mp *MemoryProtection) VerifyFunction(name string, currentCode string) bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if !mp.protectedFunctions[name] {
		return true
	}

	original, exists := mp.originalCode[name]
	if !exists {
		return false
	}

	return original == currentCode
}

func (mp *MemoryProtection) DetectModification(name string, currentCode string) bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if !mp.protectedFunctions[name] {
		return false
	}

	original, exists := mp.originalCode[name]
	if !exists {
		return true
	}

	return original != currentCode
}

type RuntimeMonitor struct {
	antiDebug    *AntiDebug
	integrChecker *IntegrityChecker
	memProtect   *MemoryProtection
	eventLog     []MonitorEvent
	mu           sync.RWMutex
}

type MonitorEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	EventType  string    `json:"event_type"`
	Details    string    `json:"details"`
	Severity   string    `json:"severity"`
}

func NewRuntimeMonitor() *RuntimeMonitor {
	return &RuntimeMonitor{
		antiDebug:    NewAntiDebug(),
		integrChecker: NewIntegrityChecker(),
		memProtect:   NewMemoryProtection(),
		eventLog:     make([]MonitorEvent, 0),
	}
}

func (rm *RuntimeMonitor) LogEvent(eventType, details, severity string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	event := MonitorEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		Details:   details,
		Severity:  severity,
	}

	rm.eventLog = append(rm.eventLog, event)

	if len(rm.eventLog) > 10000 {
		rm.eventLog = rm.eventLog[len(rm.eventLog)-5000:]
	}
}

func (rm *RuntimeMonitor) GetRecentEvents(limit int) []MonitorEvent {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if limit > len(rm.eventLog) {
		limit = len(rm.eventLog)
	}

	events := make([]MonitorEvent, limit)
	copy(events, rm.eventLog[len(rm.eventLog)-limit:])
	return events
}

func (rm *RuntimeMonitor) CheckForCompromise() bool {
	if rm.antiDebug.IsCompromised() {
		rm.LogEvent("compromise", "anti-debug marked as compromised", "critical")
		return true
	}

	detectionCount := rm.antiDebug.GetDetectionCount()
	if detectionCount > 10 {
		rm.LogEvent("high_detections", fmt.Sprintf("detection count: %d", detectionCount), "warning")
	}

	return false
}

func (rm *RuntimeMonitor) GetStatus() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return map[string]interface{}{
		"is_compromised":  rm.antiDebug.IsCompromised(),
		"detection_count":  rm.antiDebug.GetDetectionCount(),
		"protected_codes":  len(rm.integrChecker.hashes),
		"protected_funcs":  len(rm.memProtect.protectedFunctions),
		"event_log_size":  len(rm.eventLog),
		"last_detection":  rm.antiDebug.lastDetectTime,
	}
}

type AntiDebugGenerator struct {
	config AntiDebugConfig
}

func NewAntiDebugGenerator(config ...AntiDebugConfig) *AntiDebugGenerator {
	cfg := defaultAntiDebugConfig
	if len(config) > 0 {
		cfg = config[0]
	}
	return &AntiDebugGenerator{config: cfg}
}

func (g *AntiDebugGenerator) GenerateJavaScriptProtection() string {
	var code strings.Builder

	code.WriteString(`;(function(){
var _0xAD={};
_0xAD.config={`)

	if g.config.EnableDevToolsDetection {
		code.WriteString(`enableDevTools:true,`)
	}
	if g.config.EnableMemoryProtection {
		code.WriteString(`enableMemoryProtection:true,`)
	}
	if g.config.EnableSelfDestruct {
		code.WriteString(`enableSelfDestruct:true,`)
	}
	if g.config.EnableAntiTampering {
		code.WriteString(`enableAntiTampering:true,`)
	}
	if g.config.EnableAutomationDetection {
		code.WriteString(`enableAutomationDetection:true,`)
	}

	code.WriteString(`};`)

	code.WriteString(`
_0xAD.detectors=[];
_0xAD.lastCheck=0;
_0xAD.compromised=false;
_0xAD.detectionCount=0;

_0xAD.registerDetector=function(fn){
this.detectors.push(fn);
};

_0xAD.check=function(){
if(this.compromised)return true;
var now=Date.now();
if(now-this.lastCheck<500)return false;
this.lastCheck=now;

for(var i=0;i<this.detectors.length;i++){
try{
if(this.detectors[i]()){
this.detectionCount++;
this.onDetected();
return true;
}
}catch(e){}
}
return false;
};

_0xAD.onDetected=function(){
if(this.compromised)return;
this.compromised=true;
this.triggerSelfDestruct();
};

_0xAD.triggerSelfDestruct=function(){
if(this.config.enableSelfDestruct){
document.documentElement.style.display='none';
document.body.innerHTML='<div style="position:fixed;top:0;left:0;right:0;bottom:0;display:flex;align-items:center;justify-content:center;background:#000;color:#fff;font-family:sans-serif;z-index:2147483647;"><h1 style="color:#f00;">Access Denied</h1></div>';
setTimeout(function(){
var scripts=document.getElementsByTagName('script');
for(var i=scripts.length-1;i>=0;i--){
if(scripts[i].parentNode)scripts[i].parentNode.removeChild(scripts[i]);
}
},100);
throw new Error('Security violation');
}
};

_0xAD.generateChallenge=function(){
var nonce='';
var chars='ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
for(var i=0;i<32;i++){
nonce+=chars.charAt(Math.floor(Math.random()*chars.length));
}
return nonce;
};

_0xAD.validateChallenge=function(challenge,response){
return challenge&&response&&response.length===64;
};

_0xAD.init=function(){
var self=this;

`)
	if g.config.EnableDevToolsDetection {
		code.WriteString(g.generateDevToolsDetector())
	}
	if g.config.EnableMemoryProtection {
		code.WriteString(g.generateMemoryProtection())
	}
	if g.config.EnableAntiTampering {
		code.WriteString(g.generateAntiTampering())
	}
	if g.config.EnableAutomationDetection {
		code.WriteString(g.generateAutomationDetector())
	}

	code.WriteString(`
var checkInterval=setInterval(function(){
if(self.check()){
clearInterval(checkInterval);
}
},`)

	code.WriteString(fmt.Sprintf("%d", g.config.CheckInterval.Milliseconds()))

	code.WriteString(`);

document.addEventListener('keydown',function(e){
if(e.keyCode===123||e.ctrlKey&&e.shiftKey&&e.keyCode===73){
e.preventDefault();
self.onDetected();
}
});

document.addEventListener('contextmenu',function(e){
e.preventDefault();
});
};

if(document.readyState==='loading'){
document.addEventListener('DOMContentLoaded',function(){_0xAD.init();});
}else{
_0xAD.init();
}

window.__AntiDebug=_0xAD;
})();
`)
	return code.String()
}

func (g *AntiDebugGenerator) generateDevToolsDetector() string {
	return `
self.registerDetector(function(){
var threshold=160;
if(window.outerWidth-window.innerWidth>threshold||window.outerHeight-window.innerHeight>threshold){
return true;
}
});

self.registerDetector(function(){
var start=Date.now();
debugger;
var end=Date.now();
if(end-start>100)return true;
return false;
});

self.registerDetector(function(){
(function(x){
var d=document.createElement('div');
d.innerHTML='<x/>';
Object.defineProperty(x,'inspect',{
get:function(){return function(){};}
});
})(window);
return window.__y&&window.__y.innerHTML==='';
});

self.registerDetector(function(){
return window.devtools&&window.devtools.isOpen;
});
`
}

func (g *AntiDebugGenerator) generateMemoryProtection() string {
	return `
var originalToString=Function.prototype.toString.toString();
self.registerDetector(function(){
if(Function.prototype.toString.toString()!==originalToString){
return true;
}
});

self.registerDetector(function(){
if(console.log.toString().indexOf('[native code]')===-1){
return true;
}
});
`
}

func (g *AntiDebugGenerator) generateAntiTampering() string {
	return `
var scriptHashes=[];
self.registerDetector(function(){
var scripts=document.getElementsByTagName('script');
for(var i=0;i<scripts.length;i++){
if(scripts[i].src&&scripts[i].src.indexOf('crypto-utils')!==-1){
var content=scripts[i].innerHTML||scripts[i].textContent;
if(scriptHashes[i]&&scriptHashes[i]!==self.hashCode(content)){
return true;
}
scriptHashes[i]=self.hashCode(content);
}
}
return false;
});

self.hashCode=function(str){
var hash=0;
for(var i=0;i<str.length;i++){
var char=str.charCodeAt(i);
hash=((hash<<5)-hash)+char;
hash=hash&hash;
}
return hash;
};
`
}

func (g *AntiDebugGenerator) generateAutomationDetector() string {
	return `
var automationIndicators=[
{name:'HeadlessChrome',check:function(){return navigator.userAgent.indexOf('HeadlessChrome')!==-1;}},
{name:'PhantomJS',check:function(){return navigator.userAgent.indexOf('PhantomJS')!==-1;}},
{name:'Selenium',check:function(){return navigator.userAgent.indexOf('Selenium')!==-1;}},
{name:'webdriver',check:function(){return navigator.webdriver===true;}},
{name:'puppeteer',check:function(){return navigator.userAgent.indexOf('puppeteer')!==-1;}},
{name:'callPhantom',check:function(){return typeof window.callPhantom!=='undefined';}},
{name:'_phantom',check:function(){return typeof window._phantom!=='undefined';}}
];

automationIndicators.forEach(function(indicator){
self.registerDetector(function(){
try{
return indicator.check();
}catch(e){return false;}
});
});
`
}

func (g *AntiDebugGenerator) GenerateIntegrityCheckCode(codeHash string) string {
	return fmt.Sprintf(`
;(function(){
var _0xIH='%s';
window.__integrityHash=_0xIH;
window.__verifyIntegrity=function(){
var scripts=document.getElementsByTagName('script');
for(var i=0;i<scripts.length;i++){
if(scripts[i].src&&scripts[i].src.indexOf('crypto-utils')!==-1){
var content=scripts[i].innerHTML||scripts[i].textContent;
var hash=0;
for(var j=0;j<content.length;j++){
var char=content.charCodeAt(j);
hash=((hash<<5)-hash)+char;
hash=hash&hash;
}
if(hash.toString()!==_0xIH){
document.documentElement.style.display='none';
document.body.innerHTML='<h1>Integrity violation</h1>';
throw new Error('Code integrity compromised');
}
}
}
};
setInterval(function(){window.__verifyIntegrity();},5000);
})();
`, codeHash)
}

func (g *AntiDebugGenerator) GenerateMemoryGuardCode() string {
	return `
;(function(){
var _0xMG={
protectedObjects:{},
originalDescriptors:{},

protect:function(obj,prop){
var key=obj+':'+prop;
var descriptor=Object.getOwnPropertyDescriptor(window,prop);
if(!descriptor)return;
this.originalDescriptors[key]=descriptor;
Object.defineProperty(window,prop,{
get:function(){
return this._originalValue;
},
set:function(v){
if(v&&v.toString&&v.toString().indexOf('[native code]')===-1){
document.documentElement.style.display='none';
document.body.innerHTML='<h1>Memory modification detected</h1>';
throw new Error('Security violation');
}
this._originalValue=v;
},
configurable:false,
enumerable:descriptor.enumerable
});
},

check:function(){
var nativeIndicators=['[native code]','function()','()=>'];
var checks=[
{name:'Function.prototype.toString',obj:Function.prototype,prop:'toString'},
{name:'console.log',obj:console,prop:'log'},
{name:'console.error',obj:console,prop:'error'}
];

for(var i=0;i<checks.length;i++){
var c=checks[i];
try{
if(c.obj[c.prop]&&c.obj[c.prop].toString().indexOf('[native code]')===-1){
return true;
}
}catch(e){}
}
return false;
},

init:function(){
var self=this;
this.protect('Function','prototype');
this.protect('console','log');
this.protect('console','error');

setInterval(function(){
if(self.check()){
document.documentElement.style.display='none';
document.body.innerHTML='<h1>Memory protection triggered</h1>';
throw new Error('Memory modification detected');
}
},3000);
}
};

if(document.readyState==='complete'){
_0xMG.init();
}else{
document.addEventListener('DOMContentLoaded',function(){_0xMG.init();});
}

window.__MemoryGuard=_0xMG;
})();
`
}

func GenerateAntiDebugCode(options map[string]interface{}) string {
	var code strings.Builder

	enableDevTools := true
	enableMemoryProtection := true
	enableSelfDestruct := true
	enableAntiTampering := true
	enableAutomationDetection := true
	checkInterval := 2000

	if options != nil {
		if v, ok := options["enableDevTools"].(bool); ok {
			enableDevTools = v
		}
		if v, ok := options["enableMemoryProtection"].(bool); ok {
			enableMemoryProtection = v
		}
		if v, ok := options["enableSelfDestruct"].(bool); ok {
			enableSelfDestruct = v
		}
		if v, ok := options["enableAntiTampering"].(bool); ok {
			enableAntiTampering = v
		}
		if v, ok := options["enableAutomationDetection"].(bool); ok {
			enableAutomationDetection = v
		}
		if v, ok := options["checkInterval"].(int); ok {
			checkInterval = v
		}
	}

	code.WriteString(`;(function(){`)

	if enableDevTools {
		code.WriteString(`
(function(){
var _0xDT={
threshold:160,
enabled:`)
		code.WriteString(fmt.Sprintf("%t", enableDevTools))
		code.WriteString(`,

check:function(){
if(!this.enabled)return false;

if(window.outerWidth-window.innerWidth>this.threshold||window.outerHeight-window.innerHeight>this.threshold){
return true;
}

var start=Date.now();
debugger;
var end=Date.now();
if(end-start>100)return true;

(function(x){
var d=document.createElement('div');
d.innerHTML='<x id="_0xY"/>';
Object.defineProperty(x,'inspect',{get:function(){return function(){};}});
document.head.appendChild(d);
if(document.getElementById('_0xY')){
window.__detect=true;
}
})(window);
if(window.__detect)return true;

return false;
}
};
window.__DevTools=_0xDT;
})();
`)
	}

	if enableMemoryProtection {
		code.WriteString(`
(function(){
var _0xMP={
enabled:`)
		code.WriteString(fmt.Sprintf("%t", enableMemoryProtection))
		code.WriteString(`,

originalToString:Function.prototype.toString.toString(),

check:function(){
if(!this.enabled)return false;

if(Function.prototype.toString.toString()!==this.originalToString){
return true;
}

if(console.log.toString().indexOf('[native code]')===-1){
return true;
}

return false;
}
};
window.__MemoryProtect=_0xMP;
})();
`)
	}

	if enableAntiTampering {
		code.WriteString(`
(function(){
var _0xAT={
enabled:`)
		code.WriteString(fmt.Sprintf("%t", enableAntiTampering))
		code.WriteString(`,

hashCode:function(str){
var hash=0;
for(var i=0;i<str.length;i++){
var char=str.charCodeAt(i);
hash=((hash<<5)-hash)+char;
hash=hash&hash;
}
return hash;
},

scriptHashes:{},

check:function(){
if(!this.enabled)return false;

var scripts=document.getElementsByTagName('script');
for(var i=0;i<scripts.length;i++){
var s=scripts[i];
if(s.src&&s.src.indexOf('crypto-utils')!==-1){
var content=s.innerHTML||s.textContent;
var hash=this.hashCode(content);
if(this.scriptHashes[s.src]&&this.scriptHashes[s.src]!==hash){
return true;
}
this.scriptHashes[s.src]=hash;
}
}
return false;
}
};
window.__AntiTamper=_0xAT;
})();
`)
	}

	if enableAutomationDetection {
		code.WriteString(`
(function(){
var _0xAU={
enabled:`)
		code.WriteString(fmt.Sprintf("%t", enableAutomationDetection))
		code.WriteString(`,

checks:[
function(){return navigator.userAgent.indexOf('HeadlessChrome')!==-1;},
function(){return navigator.userAgent.indexOf('PhantomJS')!==-1;},
function(){return navigator.userAgent.indexOf('Selenium')!==-1;},
function(){return navigator.userAgent.indexOf('puppeteer')!==-1;},
function(){return navigator.webdriver===true;},
function(){return typeof window.callPhantom!=='undefined';},
function(){return typeof window._phantom!=='undefined';}
],

check:function(){
if(!this.enabled)return false;
for(var i=0;i<this.checks.length;i++){
try{if(this.checks[i]())return true;}catch(e){}
}
return false;
}
};
window.__AutoDetect=_0xAU;
})();
`)
	}

	code.WriteString(fmt.Sprintf(`
(function(){
var _0xSD={
enabled:%t,
triggers:[],
check:function(){
for(var i=0;i<this.triggers.length;i++){
try{if(this.triggers[i]())return true;}catch(e){}
}
return false;
},
destroy:function(){
document.documentElement.style.display='none';
document.body.innerHTML='<div style="position:fixed;top:0;left:0;right:0;bottom:0;display:flex;align-items:center;justify-content:center;background:#000;color:#fff;font-family:sans-serif;z-index:2147483647;"><h1 style="color:#f00;">Security Violation</h1></div>';
var s=document.getElementsByTagName('script');
for(var i=s.length-1;i>=0;i--){if(s[i].parentNode)s[i].parentNode.removeChild(s[i]);}
throw new Error('Debug detected');
},
init:function(){
var self=this;
if(typeof window.__DevTools!=='undefined')this.triggers.push(window.__DevTools.check);
if(typeof window.__MemoryProtect!=='undefined')this.triggers.push(window.__MemoryProtect.check);
if(typeof window.__AntiTamper!=='undefined')this.triggers.push(window.__AntiTamper.check);
if(typeof window.__AutoDetect!=='undefined')this.triggers.push(window.__AutoDetect.check);

setInterval(function(){
if(self.enabled&&self.check())self.destroy();
},%d);
}
};
_0xSD.init();
})();
`, enableSelfDestruct, checkInterval))

	code.WriteString(`
document.addEventListener('keydown',function(e){
if(e.keyCode===123||e.ctrlKey&&e.shiftKey&&e.keyCode===73||e.ctrlKey&&e.keyCode===85){
e.preventDefault();
document.documentElement.style.display='none';
document.body.innerHTML='<h1>Access Denied</h1>';
}
});

document.addEventListener('contextmenu',function(e){e.preventDefault();});
})();
`)

	return code.String()
}

type AntiDebugMiddleware struct {
	monitor *RuntimeMonitor
	enabled bool
}

func NewAntiDebugMiddleware() *AntiDebugMiddleware {
	return &AntiDebugMiddleware{
		monitor: NewRuntimeMonitor(),
		enabled: true,
	}
}

func (m *AntiDebugMiddleware) IsEnabled() bool {
	return m.enabled
}

func (m *AntiDebugMiddleware) SetEnabled(enabled bool) {
	m.enabled = enabled
}

func (m *AntiDebugMiddleware) GetMonitor() *RuntimeMonitor {
	return m.monitor
}

func (m *AntiDebugMiddleware) GenerateProtectionScript() string {
	gen := NewAntiDebugGenerator()
	return gen.GenerateJavaScriptProtection()
}

func GenerateSecureToken(secret []byte, data []byte) (string, error) {
	h := hmac.New(sha256.New, secret)
	h.Write(data)
	signature := h.Sum(nil)

	token := make([]byte, len(data)+len(signature))
	copy(token, data)
	copy(token[len(data):], signature)

	return base64.StdEncoding.EncodeToString(token), nil
}

func VerifySecureToken(secret []byte, token string) ([]byte, bool) {
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, false
	}

	if len(decoded) < 32 {
		return nil, false
	}

	data := decoded[:len(decoded)-32]
	signature := decoded[len(decoded)-32:]

	h := hmac.New(sha256.New, secret)
	h.Write(data)
	expected := h.Sum(nil)

	if subtle.ConstantTimeCompare(signature, expected) != 1 {
		return nil, false
	}

	return data, true
}

type CodeObfuscator struct {
	config   ObfuscatorConfig
	key      []byte
	antiDebug *AntiDebug
}

func NewCodeObfuscator(key ...[]byte) *CodeObfuscator {
	obfuscatorKey := key
	if len(key) == 0 {
		obfuscatorKey = [][]byte{[]byte("hjtpx-obfuscate-key-2024")}
	}

	return &CodeObfuscator{
		config:   defaultObfuscatorConfig,
		key:      obfuscatorKey[0],
		antiDebug: NewAntiDebug(),
	}
}

func (o *CodeObfuscator) Protect(code string) (string, error) {
	hash := o.antiDebug.GenerateIntegrityHash(code)
	o.antiDebug.SetIntegrityHash(hash)

	obf := NewObfuscator(ObfuscatorConfig{
		EnableVariableObfuscation:   true,
		EnableStringEncryption:      true,
		EnableCodeCompression:       true,
		EnableControlFlowFlattening: true,
		EnableFunctionWrapping:      true,
		EnableDebuggerDetection:     true,
		EnableSelfDestruct:          true,
		StringEncryptionKey:         o.key,
	})

	protected, err := obf.Obfuscate(code)
	if err != nil {
		return "", err
	}

	antiDebugCode := GenerateAntiDebugCode(map[string]interface{}{
		"enableDevTools":           true,
		"enableMemoryProtection":   true,
		"enableSelfDestruct":      true,
		"enableAntiTampering":      true,
		"enableAutomationDetection": true,
		"checkInterval":            2000,
	})

	integrityCode := GenerateIntegrityCheck(hash)

	return antiDebugCode + "\n" + integrityCode + "\n" + protected, nil
}

func GenerateIntegrityCheck(codeHash string) string {
	return fmt.Sprintf(`
;(function(){
var _0xH='%s';
var _0xC=document.currentScript;
var _0xV=setInterval(function(){
if(!_0xC){
_0xC=document.querySelector('script[data-protected]');
}
if(_0xC){
var _0xS=_0xC.innerHTML||_0xC.textContent;
var _0xHash=0;
for(var _0xI=0;_0xI<_0xS.length;_0xI++){
_0xHash=((_0xHash<<5)-_0xHash)+_0xS.charCodeAt(_0xI);
_0xHash=_0xHash&_0xHash;
}
if(_0xHash.toString()!==_0xH){
document.documentElement.style.display='none';
document.body.innerHTML='<h1>Code integrity violation</h1>';
clearInterval(_0xV);
}
}
},3000);
})();
`, codeHash)
}

type IntegrityReport struct {
	Timestamp      time.Time              `json:"timestamp"`
	IntegrityHash  string                 `json:"integrity_hash"`
	IsValid        bool                   `json:"is_valid"`
	Violations     []IntegrityViolation   `json:"violations"`
	DetectionCount int                    `json:"detection_count"`
}

type IntegrityViolation struct {
	Type        string    `json:"type"`
	Timestamp  time.Time `json:"timestamp"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
}

func GenerateIntegrityReport(monitor *RuntimeMonitor) *IntegrityReport {
	report := &IntegrityReport{
		Timestamp:      time.Now(),
		IsValid:        !monitor.antiDebug.IsCompromised(),
		DetectionCount: monitor.antiDebug.GetDetectionCount(),
		Violations:     make([]IntegrityViolation, 0),
	}

	if monitor.antiDebug.IsCompromised() {
		report.Violations = append(report.Violations, IntegrityViolation{
			Type:        "compromise",
			Timestamp:  time.Now(),
			Description: "System marked as compromised",
			Severity:    "critical",
		})
	}

	if report.DetectionCount > 0 {
		report.Violations = append(report.Violations, IntegrityViolation{
			Type:        "detection",
			Timestamp:  time.Now(),
			Description: fmt.Sprintf("Detection count: %d", report.DetectionCount),
			Severity:    "warning",
		})
	}

	return report
}

func CreateCodeGuard(code, key string) string {
	hash := sha256.Sum256([]byte(code + key))
	hashStr := hex.EncodeToString(hash[:])

	return fmt.Sprintf(`
;(function(_0xC,_0xK,_0xH){
var _0xV=setInterval(function(){
var _0xS=document.querySelector('script[data-hash="%s"]');
if(!_0xS){
clearInterval(_0xV);
document.documentElement.style.display='none';
document.body.innerHTML='<h1>Code integrity check failed</h1>';
throw new Error('Integrity violation');
}
},3000);
})(null,'%s','%s');
`, hashStr, key, hashStr)
}

type VirtualizationEngine struct {
	instructions map[string]func(*VMContext)
}

type VMContext struct {
	Registers map[string]interface{}
	Stack     []interface{}
	IP        int
	Memory    []byte
}

func NewVirtualizationEngine() *VirtualizationEngine {
	ve := &VirtualizationEngine{
		instructions: make(map[string]func(*VMContext)),
	}

	ve.instructions["NOP"] = func(ctx *VMContext) { ctx.IP++ }
	ve.instructions["PUSH"] = func(ctx *VMContext) { ctx.IP++ }
	ve.instructions["POP"] = func(ctx *VMContext) {
		if len(ctx.Stack) > 0 {
			ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
		}
		ctx.IP++
	}
	ve.instructions["ADD"] = func(ctx *VMContext) {
		if len(ctx.Stack) >= 2 {
			a := ctx.Stack[len(ctx.Stack)-2].(int)
			b := ctx.Stack[len(ctx.Stack)-1].(int)
			ctx.Stack = ctx.Stack[:len(ctx.Stack)-2]
			ctx.Stack = append(ctx.Stack, a+b)
		}
		ctx.IP++
	}
	ve.instructions["HALT"] = func(ctx *VMContext) { ctx.IP = -1 }

	return ve
}

func (ve *VirtualizationEngine) Execute(code []byte) error {
	ctx := &VMContext{
		Registers: make(map[string]interface{}),
		Stack:     make([]interface{}, 0),
		IP:        0,
		Memory:    code,
	}

	for ctx.IP >= 0 && ctx.IP < len(code) {
		op := code[ctx.IP]
		switch op {
		case 0x00:
			ve.instructions["NOP"](ctx)
		case 0x01:
			ve.instructions["PUSH"](ctx)
		case 0x02:
			ve.instructions["POP"](ctx)
		case 0x03:
			ve.instructions["ADD"](ctx)
		case 0xFF:
			ve.instructions["HALT"](ctx)
		default:
			return fmt.Errorf("unknown opcode: %02x", op)
		}
	}

	return nil
}

func CreateVirtualizedCode(code string) string {
	var bytecode []byte

	for _, c := range code {
		bytecode = append(bytecode, byte(c))
	}

	bytecode = append(bytecode, 0xFF)

	encoded := make([]byte, len(bytecode)*2)
	for i, b := range bytecode {
		encoded[i*2] = b ^ 0x55
		encoded[i*2+1] = b ^ 0xAA
	}

	return fmt.Sprintf(`
;(function(){
var _0xBC=[%s];
var _0xVM={
regs:{},
stack:[],
ip:0,
exec:function(_0xCode){
while(this.ip>=0&&this.ip<_0xCode.length){
var op=_0xCode[this.ip];
switch(op){
case 0x00:this.ip++;break;
case 0x01:this.ip++;break;
case 0x02:if(this.stack.length>0)this.stack.pop();this.ip++;break;
case 0x03:if(this.stack.length>=2){var b=this.stack.pop();var a=this.stack.pop();this.stack.push(a+b);}this.ip++;break;
case 0xFF:return;
default:return;
}
}
}
};
_0xVM.exec(_0xBC);
})();
`, formatByteArray(encoded))
}

func formatByteArray(data []byte) string {
	var parts []string
	for _, b := range data {
		parts = append(parts, fmt.Sprintf("0x%02x", b))
	}
	return strings.Join(parts, ",")
}

func ValidateCodeSafety(code string) (bool, string) {
	dangerousPatterns := []string{
		`eval\s*\(`,
		`Function\s*\(`,
		`document\.write\s*\(`,
		`innerHTML\s*=`,
		`outerHTML\s*=`,
		`setTimeout\s*\(\s*".*"\s*,`,
		`setInterval\s*\(\s*".*"\s*,`,
		`__proto__`,
		`prototype\s*=`,
	}

	for _, pattern := range dangerousPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(code) {
			return false, fmt.Sprintf("dangerous pattern detected: %s", pattern)
		}
	}

	return true, "safe"
}
