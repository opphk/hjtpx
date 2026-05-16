package handler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type DetectionSession struct {
	ID          string                 `json:"detection_id"`
	RiskScore   float64                `json:"risk_score"`
	Chain       []string               `json:"chain"`
	Fingerprint string                 `json:"fingerprint"`
	SessionID   string                 `json:"session_id"`
	Timestamp   int64                  `json:"timestamp"`
	Details     map[string]interface{} `json:"details,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	ClientIP    string                 `json:"client_ip,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
}

type DetectionSubmitRequest struct {
	DetectionID string              `json:"detection_id" binding:"required"`
	RiskScore   float64             `json:"risk_score" binding:"required"`
	Chain       []string            `json:"chain"`
	Fingerprint string              `json:"fingerprint"`
	SessionID   string              `json:"session_id"`
	Timestamp   int64               `json:"timestamp"`
	Details     json.RawMessage     `json:"details,omitempty"`
}

type EnvironmentDetails struct {
	BrowserFingerprint map[string]interface{} `json:"browser_fingerprint,omitempty"`
	AutomationIndicators map[string]interface{} `json:"automation_indicators,omitempty"`
	NetworkInfo        map[string]interface{} `json:"network_info,omitempty"`
	HardwareInfo       map[string]interface{} `json:"hardware_info,omitempty"`
	ProxyIndicators    map[string]interface{} `json:"proxy_indicators,omitempty"`
	AnomalyScore       float64                `json:"anomaly_score,omitempty"`
}

var (
	detectionSessions = make(map[string]*DetectionSession)
	detectionMutex    sync.RWMutex
)

var (
	proxyHeaders = []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"CF-Connecting-IP",
		"X-Forwarded-Proto",
		"X-Forwarded-Host",
		"Via",
		"X-Varnish",
		"True-Client-IP",
		"X-Original-For",
	}

	proxyRegex = regexp.MustCompile(`(?i)(proxy|vpn|tor|exitnode|anonymizer)`)

	automationUARegex = regexp.MustCompile(`(?i)(headless|phantom|puppet|selenium|playwright|chromium|electron)`)

	knownProxyIPRanges = []string{
		"10.", "172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.",
		"172.25.", "172.26.", "172.27.", "172.28.", "172.29.",
		"172.30.", "172.31.", "192.168.",
	}
)

func init() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cleanupExpiredDetectionSessions()
		}
	}()
}

type detectionMethod struct {
	Name      string
	Code      string
	ReturnVar string
}

func generateDetectionMethods() []detectionMethod {
	return []detectionMethod{
		{
			Name: "webgl",
			Code: `var __v0=document.createElement('canvas');var __v1=__v0.getContext('webgl')||__v0.getContext('experimental-webgl');if(__v1){var __v2=__v1.getExtension('WEBGL_debug_renderer_info');__v3=__v2?__v1.getParameter(__v2.UNMASKED_VENDOR_WEBGL)+'|'+__v1.getParameter(__v2.UNMASKED_RENDERER_WEBGL):'no_ext';__v4=__v1.getParameter(__v1.VERSION);__v5=__v1.getParameter(__v1.SHADING_LANGUAGE_VERSION);__v6=__v1.getParameter(__v1.MAX_TEXTURE_SIZE);__v7=__v1.getParameter(__v1.MAX_VERTEX_ATTRIBS);__v8=__v1.getParameter(__v1.ALIASED_LINE_WIDTH_RANGE).join(',');__v9=__v1.getParameter(__v1.ALIASED_POINT_SIZE_RANGE).join(',');__v10=__v1.getExtension('EXT_texture_filter_anisotropic')?'aniso_yes':'aniso_no';var __v11=__v1.getSupportedExtensions().join(',')}else{__v3='no_webgl';__v4='';__v5='';__v6='';__v7='';__v8='';__v9='';__v10='';__v11=''}`,
			ReturnVar: "__v3",
		},
		{
			Name: "webgl2",
			Code: `var __v12=document.createElement('canvas');var __v13=__v12.getContext('webgl2');if(__v13){var __v14=__v13.getExtension('WEBGL_debug_renderer_info');__v15=__v14?__v13.getParameter(__v14.UNMASKED_VENDOR_WEBGL)+'|'+__v13.getParameter(__v14.UNMASKED_RENDERER_WEBGL):'no_ext';__v16=__v13.getParameter(__v13.MAX_TEXTURE_SIZE);__v17=__v13.getParameter(__v13.MAX_VERTEX_ATTRIBS);__v18=__v13.getParameter(__v13.ALIASED_LINE_WIDTH_RANGE).join(',');__v19=__v13.getSupportedExtensions().join(',')}else{__v15='no_webgl2'}`,
			ReturnVar: "__v15",
		},
		{
			Name: "canvas",
			Code: `var __v20=document.createElement('canvas');__v20.width=280;__v20.height=60;var __v21=__v20.getContext('2d');__v21.textBaseline='alphabetic';__v21.fillStyle='#f60';__v21.fillRect(125,1,62,20);__v21.fillStyle='#069';__v21.font='11pt Arial';__v21.fillText('Cwm fjordbank glyphs vext quiz, \ud83d\ude03',2,15);__v21.fillStyle='rgba(102,204,0,0.7)';__v21.font='18pt Arial';__v21.fillText('Cwm fjordbank glyphs vext quiz, \ud83d\ude03',4,45);__v21.globalCompositeOperation='multiply';__v21.fillStyle='rgb(255,0,255)';__v21.beginPath();__v21.arc(50,50,50,0,Math.PI*2,true);__v21.closePath();__v21.fill();__v21.fillStyle='rgb(0,255,255)';__v21.beginPath();__v21.arc(100,50,50,0,Math.PI*2/3,true);__v21.closePath();__v21.fill();__v21.fillStyle='rgb(255,255,0)';__v21.beginPath();__v21.arc(75,50,50,0,Math.PI*2/3,false);__v21.closePath();__v21.fill();var __v22=__v20.toDataURL();var __v23=__v22.split(',')[1]||__v22`,
			ReturnVar: "__v23",
		},
		{
			Name: "audio",
			Code: `var __v24=window.OfflineAudioContext||window.webkitOfflineAudioContext;if(__v24){try{var __v25=new __v24(1,44100,44100);var __v26=__v25.createOscillator();__v26.type='triangle';__v26.frequency.setValueAtTime(1e4,__v25.currentTime);var __v27=__v25.createDynamicsCompressor();__v27.threshold.setValueAtTime(-50,__v25.currentTime);__v27.knee.setValueAtTime(40,__v25.currentTime);__v27.ratio.setValueAtTime(12,__v25.currentTime);__v27.attack.setValueAtTime(0,__v25.currentTime);__v27.release.setValueAtTime(0.25,__v25.currentTime);__v26.connect(__v27);__v27.connect(__v25.destination);__v25.startRendering();__v25.oncomplete=function(__v28){var __v29=__v28.renderedBuffer.getChannelData(0);var __v30=0;for(var __v31=4500;__v31<5000;__v31++){__v30+=Math.abs(__v29[__v31])}var __v32=0;for(var __v33=0;__v33<__v29.length;__v33++){__v32+=__v29[__v33]*__v29[__v33]}__v34=__v30.toString()+':'+__v32.toString()};__v25.onerror=function(){__v34='audio_err'}}catch(__v35){__v34='audio_err'}}else{__v34='no_audio'}`,
			ReturnVar: "__v34",
		},
		{
			Name: "screen",
			Code: `var __v36=window.screen.width+'x'+window.screen.height+'x'+window.screen.colorDepth+'x'+window.screen.pixelDepth+'x'+window.devicePixelRatio+'x'+window.screen.availWidth+'x'+window.screen.availHeight`,
			ReturnVar: "__v36",
		},
		{
			Name: "timezone",
			Code: `var __v37=Intl.DateTimeFormat().resolvedOptions().timeZone+'|'+new Date().getTimezoneOffset()`,
			ReturnVar: "__v37",
		},
		{
			Name: "languages",
			Code: `var __v38=navigator.language+'|'+(navigator.languages?navigator.languages.join(','):'')`,
			ReturnVar: "__v38",
		},
		{
			Name: "platform",
			Code: `var __v39=navigator.platform+'|'+navigator.oscpu+'|'+navigator.vendor+'|'+navigator.productSub+'|'+navigator.product`,
			ReturnVar: "__v39",
		},
		{
			Name: "fonts",
			Code: `var __v40=function(){var __v41=['monospace','sans-serif','serif'];var __v42=['Arial','Helvetica','Times New Roman','Courier New','Verdana','Georgia','Comic Sans MS','Impact','Lucida Console','Tahoma','Trebuchet MS','Palatino','Garamond','Bookman','Futura','Optima','Candara','Calibri','Cambria','Corbel','Segoe UI','Roboto','Open Sans','Lato','Montserrat','Source Sans Pro','Raleway','Ubuntu','Noto Sans','Droid Sans','Fira Sans','Merriweather','Playfair Display','PT Sans','Nunito','Quicksand','Work Sans','Oswald','Roboto Condensed','Noto Serif','Lora','IBM Plex Sans','JetBrains Mono','SF Pro Display','SF Pro Text'];var __v43={};var __v44=document.createElement('div');__v44.style.cssText='position:absolute;left:-9999px;top:-9999px;visibility:hidden;white-space:nowrap;font-size:72px';document.body.appendChild(__v44);for(var __v45=0;__v45<__v41.length;__v45++){var __v46=document.createElement('span');__v46.style.cssText='font-family:'+__v41[__v45]+';position:absolute;left:-9999px;top:-9999px;visibility:hidden;white-space:nowrap;font-size:72px';__v46.textContent='mmmmmmmmmmlli';__v44.appendChild(__v46);__v43[__v41[__v45]]=__v46.offsetWidth}var __v47=[];for(var __v48=0;__v48<__v42.length;__v48++){var __v49=document.createElement('span');__v49.style.cssText='font-family:'+__v42[__v48]+','+__v41.join(',')+';position:absolute;left:-9999px;top:-9999px;visibility:hidden;white-space:nowrap;font-size:72px';__v49.textContent='mmmmmmmmmmlli';__v44.appendChild(__v49);for(var __v50=0;__v50<__v41.length;__v50++){if(__v49.offsetWidth!==__v43[__v41[__v50]]){__v47.push(__v42[__v48]);break}}}document.body.removeChild(__v44);return __v47.join(',')}()`,
			ReturnVar: "__v40",
		},
		{
			Name: "webrtc_ip",
			Code: `var __v51=window.RTCPeerConnection||window.webkitRTCPeerConnection||window.mozRTCPeerConnection;if(__v51){try{var __v52={};var __v53=new __v51({iceServers:[{urls:'stun:stun.l.google.com:19302'},{urls:'stun:stun1.l.google.com:19302'}]});__v53.createDataChannel('');__v53.createOffer(function(__v54){__v53.setLocalDescription(__v54,function(){try{var __v55=__v53.localDescription.sdp.split('\\n');for(var __v56=0;__v56<__v55.length;__v56++){if(__v55[__v56].indexOf('candidate')>-1){var __v57=__v55[__v56].split(' ');if(__v57[4]&&__v57[4]!=='0.0.0.0'&&__v57[7]!=='host'){__v52[__v57[7]]=__v57[4]}}}__v58=JSON.stringify(__v52)},function(){__v58='webrtc_err'})},function(){__v58='webrtc_offer_err'});setTimeout(function(){if(!__v58){__v58='webrtc_timeout'}},5000)}catch(__v59){__v58='webrtc_err'}}else{__v58='no_webrtc'}`,
			ReturnVar: "__v58",
		},
		{
			Name: "battery",
			Code: `if(navigator.getBattery){navigator.getBattery().then(function(__v60){__v61=__v60.level+'|'+__v60.charging+'|'+__v60.chargingTime+'|'+__v60.dischargingTime})}else{__v61='no_battery'}`,
			ReturnVar: "__v61",
		},
		{
			Name: "memory",
			Code: `var __v62=navigator.deviceMemory||'unknown'`,
			ReturnVar: "__v62",
		},
		{
			Name: "cpu",
			Code: `var __v63=navigator.hardwareConcurrency||'unknown'`,
			ReturnVar: "__v63",
		},
		{
			Name: "touch",
			Code: `var __v64='ontouchstart'in window||navigator.maxTouchPoints>0?'touch:'+navigator.maxTouchPoints:'no_touch'`,
			ReturnVar: "__v64",
		},
		{
			Name: "webdriver",
			Code: `var __v65=navigator.webdriver===true||navigator.webdriver!==undefined?'wd:'+navigator.webdriver:'no_wd'`,
			ReturnVar: "__v65",
		},
		{
			Name: "selenium",
			Code: `var __v66=window.document.__selenium||window.__selenium||window.callSelenium||window._selenium||document.__selenium||'no_selenium'`,
			ReturnVar: "__v66",
		},
		{
			Name: "puppeteer",
			Code: `var __v67=window.navigator.webdriver===true?'pw_webdriver':(document.$cdc_asdjflasutopfhvcZLmcfl_?'pw_cdc':'no_puppeteer')`,
			ReturnVar: "__v67",
		},
		{
			Name: "playwright",
			Code: `var __v68=window.__playwright__!==undefined?'pw_global':(window.__pw_resume__!==undefined?'pw_resume':'no_playwright')`,
			ReturnVar: "__v68",
		},
		{
			Name: "chrome_runtime",
			Code: `var __v69=window.chrome?('chrome_runtime:'+(window.chrome.runtime?typeof window.chrome.runtime.id:'missing')):'no_chrome'`,
			ReturnVar: "__v69",
		},
		{
			Name: "dnt",
			Code: `var __v70=navigator.doNotTrack||window.doNotTrack||navigator.msDoNotTrack||'unknown'`,
			ReturnVar: "__v70",
		},
		{
			Name: "storage",
			Code: `var __v71='';try{__v71+='ls:'+('localStorage'in window?'1':'0')};try{__v71+='|ss:'+('sessionStorage'in window?'1':'0')};try{__v71+='|idb:'+('indexedDB'in window?'1':'0')};try{__v71+='|odb:'+('openDatabase'in window?'1':'0')};try{__v71+='|cw:'+('cache'in window?'1':'0')}`,
			ReturnVar: "__v71",
		},
		{
			Name: "mime",
			Code: `var __v72=navigator.mimeTypes?Array.from(navigator.mimeTypes).map(function(__v73){return __v73.type}).join(','):'no_mime'`,
			ReturnVar: "__v72",
		},
		{
			Name: "plugins",
			Code: `var __v74=navigator.plugins?Array.from(navigator.plugins).map(function(__v75){return __v75.name}).join(','):'no_plugins'`,
			ReturnVar: "__v74",
		},
		{
			Name: "color",
			Code: `var __v76=window.screen.colorDepth+'|'+window.screen.pixelDepth+'|'+window.devicePixelRatio`,
			ReturnVar: "__v76",
		},
		{
			Name: "pixel_ratio",
			Code: `var __v77=window.matchMedia?'mq:'+window.matchMedia('(resolution: '+window.devicePixelRatio+'dppx)').matches:'no_mq'`,
			ReturnVar: "__v77",
		},
		{
			Name: "nav_props",
			Code: `var __v78=navigator.cookieEnabled+'|'+navigator.onLine+'|'+navigator.vendorSub+'|'+navigator.maxTouchPoints`,
			ReturnVar: "__v78",
		},
		{
			Name: "performance",
			Code: `var __v79=window.performance&&window.performance.timing?window.performance.timing.domLoading-window.performance.timing.navigationStart+'|'+window.performance.timing.domComplete-window.performance.timing.domLoading:'no_perf'`,
			ReturnVar: "__v79",
		},
		{
			Name: "connection",
			Code: `var __v80=navigator.connection?navigator.connection.effectiveType+'|'+navigator.connection.downlink+'|'+navigator.connection.rtt+'|'+navigator.connection.saveData:'no_conn'`,
			ReturnVar: "__v80",
		},
		{
			Name: "adblock",
			Code: `var __v81=document.createElement('div');__v81.innerHTML='&nbsp;';__v81.className='adsbox';__v81.style.cssText='position:absolute;left:-9999px;top:-9999px;width:1px;height:1px';document.body.appendChild(__v81);var __v82=__v81.offsetHeight===0?'adblock':'no_adblock';document.body.removeChild(__v81)`,
			ReturnVar: "__v82",
		},
		{
			Name: "math",
			Code: `var __v83=Math.sin(Math.PI/3)+'|'+Math.tan(1e7)+'|'+Math.log10(100)+'|'+Math.asin(0.5)+'|'+Math.atan2(1,2)+'|'+Math.cos(Math.PI/4)+'|'+Math.exp(1)`,
			ReturnVar: "__v83",
		},
		{
			Name: "window_size",
			Code: `var __v84=window.outerWidth+'x'+window.outerHeight+'|'+window.innerWidth+'x'+window.innerHeight+'|'+window.screenX+'x'+window.screenY`,
			ReturnVar: "__v84",
		},
		{
			Name: "iframe",
			Code: `var __v85='self:'+(window.self===window.top?'top':'iframe');try{__v85+='|parent:'+window.parent.location.href}catch(__v86){__v85+='|parent:cross'}`,
			ReturnVar: "__v85",
		},
		{
			Name: "notification",
			Code: `var __v87='Notification'in window?window.Notification.permission:'no_notification_api'`,
			ReturnVar: "__v87",
		},
		{
			Name: "media_devices",
			Code: `if(navigator.mediaDevices&&navigator.mediaDevices.enumerateDevices){navigator.mediaDevices.enumerateDevices().then(function(__v88){var __v89=__v88.map(function(__v90){return __v90.kind+':'+__v90.label}).join(',');__v91=__v89}).catch(function(){__v91='media_err'})}else{__v91='no_media_api'}`,
			ReturnVar: "__v91",
		},
		{
			Name: "gpu",
			Code: `var __v92='';try{var __v93=document.createElement('canvas');var __v94=__v93.getContext('webgl')||__v93.getContext('experimental-webgl');if(__v94){__v92=__v94.getParameter(__v94.MAX_RENDERBUFFER_SIZE)+'|'+__v94.getParameter(__v94.MAX_VIEWPORT_DIMS).join(',')+'|'+__v94.getParameter(__v94.MAX_COMBINED_TEXTURE_IMAGE_UNITS)}}catch(__v95){__v92='gpu_err'}`,
			ReturnVar: "__v92",
		},
		{
			Name: "speech",
			Code: `var __v96='speechSynthesis'in window?'synth:'+window.speechSynthesis.getVoices().length:'no_speech'`,
			ReturnVar: "__v96",
		},
	}
}

func generateRandomVarName(seen map[string]bool) string {
	prefixes := []string{"_x", "_y", "_z", "_w", "_q", "_r", "_s", "_t", "_u", "_p", "_n", "_m", "_k", "_j", "_h", "_g", "_f", "_d", "_c", "_b"}
	suffixes := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}
	for i := 0; i < 50; i++ {
		name := prefixes[rand.Intn(len(prefixes))] + suffixes[rand.Intn(len(suffixes))] + suffixes[rand.Intn(len(suffixes))]
		if !seen[name] {
			seen[name] = true
			return name
		}
	}
	name := fmt.Sprintf("_%s%d", suffixes[rand.Intn(len(suffixes))], rand.Intn(9999))
	seen[name] = true
	return name
}

func GetDetectionScript(c *gin.Context) {
	allMethods := generateDetectionMethods()

	count := 8 + rand.Intn(6)
	if count > len(allMethods) {
		count = len(allMethods)
	}

	perm := rand.Perm(len(allMethods))
	selected := make([]detectionMethod, count)
	for i := 0; i < count; i++ {
		selected[i] = allMethods[perm[i]]
	}

	seenVars := make(map[string]bool)

	varNameMap := make(map[string]string)
	for _, m := range selected {
		oldName := m.ReturnVar
		if _, exists := varNameMap[oldName]; !exists {
			varNameMap[oldName] = generateRandomVarName(seenVars)
		}
	}

	collectVar := generateRandomVarName(seenVars)
	chainVar := generateRandomVarName(seenVars)
	fpVar := generateRandomVarName(seenVars)
	resultVar := generateRandomVarName(seenVars)
	payloadVar := generateRandomVarName(seenVars)
	urlVar := generateRandomVarName(seenVars)
	xhrVar := generateRandomVarName(seenVars)
	dataVar := generateRandomVarName(seenVars)
	strVar := generateRandomVarName(seenVars)
	iVar := generateRandomVarName(seenVars)
	hashVar := generateRandomVarName(seenVars)
	timeVar := generateRandomVarName(seenVars)
	sidVar := generateRandomVarName(seenVars)
	didVar := generateRandomVarName(seenVars)
	errVar := generateRandomVarName(seenVars)

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`(function(){
var %s={};
var %s=[];
var %s='';
var %s='';
var %s='';
var %s=Date.now();
var %s='sess_'+%s+'_'+Math.floor(Math.random()*99999);
`, collectVar, chainVar, fpVar, resultVar, payloadVar, timeVar, sidVar, timeVar))

	for _, m := range selected {
		renamedCode := m.Code
		for oldName, newName := range varNameMap {
			renamedCode = strings.ReplaceAll(renamedCode, oldName, newName)
		}
		sb.WriteString(fmt.Sprintf("try{%s}catch(%s){%s='err_%s'}\n", renamedCode, errVar, varNameMap[m.ReturnVar], m.Name))
		sb.WriteString(fmt.Sprintf("%s['%s']=%s;\n", collectVar, m.Name, varNameMap[m.ReturnVar]))
		sb.WriteString(fmt.Sprintf("%s.push('%s');\n", chainVar, m.Name))
	}

	sb.WriteString(fmt.Sprintf(`
%s=function(%s){
var %s=0;
if(%s.length===0)return %s;
for(var %s=0;%s<%s.length;%s++){%s=(%s<<5)-%s+%s.charCodeAt(%s);%s=%s&%s}
return %s>>>0;
};
%s=%s(JSON.stringify(%s));
%s=btoa(unescape(encodeURIComponent(%s)));
%s=encodeURIComponent(%s);
%s='%s';
%s=new XMLHttpRequest();
%s.open('POST',%s+'/api/v1/detect/submit',true);
%s.setRequestHeader('Content-Type','application/json;charset=UTF-8');
%s.send(JSON.stringify({detection_id:%s,risk_score:%s,chain:%s,fingerprint:%s,session_id:%s,timestamp:%s,details:%s}));
`, hashVar, strVar, didVar, strVar, didVar, iVar, iVar, strVar, iVar, didVar, didVar, didVar, strVar, iVar, didVar, didVar, didVar, fpVar, hashVar, collectVar, resultVar, collectVar, payloadVar, resultVar, urlVar, c.Request.Host, xhrVar, urlVar, xhrVar, xhrVar, dataVar, didVar, resultVar, chainVar, fpVar, sidVar, timeVar, resultVar, resultVar))

	callback := c.DefaultQuery("callback", "")
	if callback != "" {
		sb.WriteString(fmt.Sprintf("window['%s']&&window['%s']({detection_id:%s,risk_score:%s,fingerprint:%s});", callback, callback, didVar, resultVar, fpVar))
	}

	sb.WriteString("})();")

	c.Header("Content-Type", "application/javascript")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.String(http.StatusOK, sb.String())
}

func SubmitDetectionData(c *gin.Context) {
	var req DetectionSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request parameters",
		})
		return
	}

	if req.RiskScore < 0 || req.RiskScore > 100 || math.IsNaN(req.RiskScore) || math.IsInf(req.RiskScore, 0) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid risk_score: must be between 0 and 100",
		})
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	serverRiskScore := req.RiskScore
	envDetails := EnvironmentDetails{}

	envDetails.NetworkInfo = analyzeNetworkHeaders(c)
	envDetails.ProxyIndicators = detectProxyFromRequest(c, clientIP, userAgent)
	envDetails.AutomationIndicators = detectAutomationFromRequest(userAgent, req.Details)

	if req.Details != nil {
		var detailsMap map[string]interface{}
		if err := json.Unmarshal(req.Details, &detailsMap); err == nil {
			analyzeFingerprintDetails(detailsMap, &envDetails)
		}
	}

	serverRiskScore = calculateEnhancedRiskScore(req.RiskScore, &envDetails, req)

	if req.Timestamp > 0 {
		now := time.Now().UnixMilli()
		diff := now - req.Timestamp
		if diff < -60000 || diff > 600000 {
			serverRiskScore = math.Min(serverRiskScore+20, 100)
			envDetails.AnomalyScore += 20
		}
	}

	if req.Fingerprint != "" {
		hash := md5.Sum([]byte(req.Fingerprint))
		req.DetectionID = hex.EncodeToString(hash[:])[:16]
	}

	session := &DetectionSession{
		ID:          req.DetectionID,
		RiskScore:   serverRiskScore,
		Chain:       req.Chain,
		Fingerprint: req.Fingerprint,
		SessionID:   req.SessionID,
		Timestamp:   req.Timestamp,
		CreatedAt:   time.Now(),
		ClientIP:    clientIP,
		UserAgent:   userAgent,
	}

	if req.Details != nil {
		var detailsMap map[string]interface{}
		if err := json.Unmarshal(req.Details, &detailsMap); err == nil {
			session.Details = detailsMap
		}
	}

	detectionMutex.Lock()
	detectionSessions[req.DetectionID] = session
	detectionMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"risk_score":  serverRiskScore,
		"anomalies":   envDetails.AnomalyScore,
	})
}

func analyzeNetworkHeaders(c *gin.Context) map[string]interface{} {
	info := make(map[string]interface{})
	var proxyChain []string

	for _, header := range proxyHeaders {
		val := c.GetHeader(header)
		if val != "" {
			proxyChain = append(proxyChain, header+":"+val)
		}
	}

	if len(proxyChain) > 0 {
		info["proxy_headers"] = proxyChain
		info["has_proxy_headers"] = true
	} else {
		info["has_proxy_headers"] = false
	}

	info["client_ip"] = c.ClientIP()
	info["scheme"] = c.Request.URL.Scheme
	if info["scheme"] == "" {
		if c.Request.TLS != nil {
			info["scheme"] = "https"
		} else {
			info["scheme"] = "http"
		}
	}

	return info
}

func detectProxyFromRequest(c *gin.Context, clientIP string, userAgent string) map[string]interface{} {
	indicators := make(map[string]interface{})
	var risks []string

	xff := c.GetHeader("X-Forwarded-For")
	xri := c.GetHeader("X-Real-IP")
	via := c.GetHeader("Via")
	cfIP := c.GetHeader("CF-Connecting-IP")

	proxyHeaderCount := 0
	if xff != "" {
		proxyHeaderCount++
		ips := strings.Split(xff, ",")
		if len(ips) > 2 {
			risks = append(risks, "multi_hop_proxy")
			indicators["xff_hop_count"] = len(ips)
		}
	}
	if xri != "" {
		proxyHeaderCount++
	}
	if via != "" {
		proxyHeaderCount++
		risks = append(risks, "via_header_present")
		if proxyRegex.MatchString(via) {
			risks = append(risks, "known_proxy_service")
			indicators["via_proxy_detected"] = true
		}
	}
	if cfIP != "" {
		proxyHeaderCount++
	}

	if proxyHeaderCount >= 2 {
		risks = append(risks, "multiple_proxy_headers")
	}
	indicators["proxy_header_count"] = proxyHeaderCount

	if clientIP != "" {
		ip := net.ParseIP(clientIP)
		if ip != nil {
			for _, prefix := range knownProxyIPRanges {
				if strings.HasPrefix(clientIP, prefix) {
					risks = append(risks, "private_ip_range")
					break
				}
			}
			if ip.IsPrivate() {
				risks = append(risks, "private_ip")
			}
		}
	}

	if userAgent != "" {
		if proxyRegex.MatchString(userAgent) {
			risks = append(risks, "proxy_in_ua")
		}
	}

	indicators["risks"] = risks
	indicators["risk_count"] = len(risks)
	return indicators
}

func detectAutomationFromRequest(userAgent string, details json.RawMessage) map[string]interface{} {
	indicators := make(map[string]interface{})
	var autoRisks []string

	if userAgent != "" {
		if automationUARegex.MatchString(userAgent) {
			autoRisks = append(autoRisks, "automation_ua")
			indicators["ua_automation_detected"] = true
		}
	}

	if details != nil {
		var detMap map[string]interface{}
		if err := json.Unmarshal(details, &detMap); err == nil {
			if wd, ok := detMap["webdriver"]; ok {
				wdStr := fmt.Sprintf("%v", wd)
				if strings.Contains(wdStr, "wd:") && !strings.Contains(wdStr, "no_wd") {
					autoRisks = append(autoRisks, "webdriver_detected")
					indicators["webdriver"] = wdStr
				}
			}
			if sel, ok := detMap["selenium"]; ok {
				selStr := fmt.Sprintf("%v", sel)
				if !strings.Contains(selStr, "no_selenium") {
					autoRisks = append(autoRisks, "selenium_detected")
					indicators["selenium"] = selStr
				}
			}
			if pw, ok := detMap["puppeteer"]; ok {
				pwStr := fmt.Sprintf("%v", pw)
				if !strings.Contains(pwStr, "no_puppeteer") {
					autoRisks = append(autoRisks, "puppeteer_detected")
					indicators["puppeteer"] = pwStr
				}
			}
			if plw, ok := detMap["playwright"]; ok {
				plwStr := fmt.Sprintf("%v", plw)
				if !strings.Contains(plwStr, "no_playwright") {
					autoRisks = append(autoRisks, "playwright_detected")
					indicators["playwright"] = plwStr
				}
			}
		}
	}

	indicators["risks"] = autoRisks
	indicators["risk_count"] = len(autoRisks)
	return indicators
}

func analyzeFingerprintDetails(details map[string]interface{}, env *EnvironmentDetails) {
	if env.BrowserFingerprint == nil {
		env.BrowserFingerprint = make(map[string]interface{})
	}
	if env.HardwareInfo == nil {
		env.HardwareInfo = make(map[string]interface{})
	}

	fingerprintFields := []string{"webgl", "webgl2", "canvas", "audio", "fonts", "math", "platform", "languages"}
	autoFields := []string{"webdriver", "selenium", "puppeteer", "playwright", "chrome_runtime"}

	for _, field := range fingerprintFields {
		if val, ok := details[field]; ok {
			env.BrowserFingerprint[field] = val
		}
	}

	for _, field := range autoFields {
		if val, ok := details[field]; ok {
			if env.AutomationIndicators == nil {
				env.AutomationIndicators = make(map[string]interface{})
			}
			env.AutomationIndicators[field] = val
		}
	}

	if mem, ok := details["memory"]; ok {
		env.HardwareInfo["device_memory"] = mem
	}
	if cpu, ok := details["cpu"]; ok {
		env.HardwareInfo["hardware_concurrency"] = cpu
	}
	if scr, ok := details["screen"]; ok {
		env.HardwareInfo["screen"] = scr
	}
	if conn, ok := details["connection"]; ok {
		env.HardwareInfo["connection"] = conn
	}
}

func calculateEnhancedRiskScore(clientScore float64, env *EnvironmentDetails, req DetectionSubmitRequest) float64 {
	score := clientScore

	proxyRisks := env.ProxyIndicators
	if proxyRisks != nil {
		if count, ok := proxyRisks["risk_count"].(int); ok {
			score += float64(count) * 8
		}
		if risks, ok := proxyRisks["risks"].([]string); ok {
			for _, r := range risks {
				switch r {
				case "multi_hop_proxy":
					score += 15
				case "known_proxy_service":
					score += 25
				case "multiple_proxy_headers":
					score += 10
				case "private_ip":
					score += 10
				}
			}
		}
	}

	autoRisks := env.AutomationIndicators
	if autoRisks != nil {
		if count, ok := autoRisks["risk_count"].(int); ok {
			score += float64(count) * 12
		}
		if risks, ok := autoRisks["risks"].([]string); ok {
			for _, r := range risks {
				switch r {
				case "automation_ua":
					score += 20
				case "webdriver_detected":
					score += 30
				case "selenium_detected":
					score += 35
				case "puppeteer_detected":
					score += 35
				case "playwright_detected":
					score += 35
				}
			}
		}
	}

	if req.Chain != nil && len(req.Chain) > 0 {
		chainSet := make(map[string]bool)
		for _, m := range req.Chain {
			chainSet[m] = true
		}
		essentialChecks := []string{"webgl", "canvas", "audio", "fonts", "webdriver"}
		missingEssential := 0
		for _, ec := range essentialChecks {
			if !chainSet[ec] {
				missingEssential++
			}
		}
		if missingEssential > 2 {
			score += float64(missingEssential) * 5
		}
	}

	env.AnomalyScore = math.Max(0, score-clientScore)

	return math.Min(math.Max(score, 0), 100)
}

func cleanupExpiredDetectionSessions() {
	detectionMutex.Lock()
	defer detectionMutex.Unlock()
	now := time.Now()
	for id, session := range detectionSessions {
		if now.Sub(session.CreatedAt) > 10*time.Minute {
			delete(detectionSessions, id)
		}
	}
}