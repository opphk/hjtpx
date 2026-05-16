package handler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type DetectionSession struct {
	ID          string             `json:"detection_id"`
	RiskScore   float64            `json:"risk_score"`
	Chain       []string           `json:"chain"`
	Fingerprint string             `json:"fingerprint"`
	SessionID   string             `json:"session_id"`
	Timestamp   int64              `json:"timestamp"`
	Details     map[string]interface{} `json:"details,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
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

var (
	detectionSessions = make(map[string]*DetectionSession)
	detectionMutex    sync.RWMutex
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
	Name        string
	Code        string
	ReturnVar   string
}

func generateDetectionMethods() []detectionMethod {
	return []detectionMethod{
		{
			Name: "webgl",
			Code: `var __v0=document.createElement('canvas');var __v1=__v0.getContext('webgl')||__v0.getContext('experimental-webgl');if(__v1){var __v2=__v1.getExtension('WEBGL_debug_renderer_info');__v3=__v2?__v1.getParameter(__v2.UNMASKED_VENDOR_WEBGL)+'|'+__v1.getParameter(__v2.UNMASKED_RENDERER_WEBGL):'no_ext';__v4=__v1.getParameter(__v1.VERSION);__v5=__v1.getParameter(__v1.SHADING_LANGUAGE_VERSION)}else{__v3='no_webgl';__v4='';__v5=''}`,
			ReturnVar: "__v3",
		},
		{
			Name: "canvas",
			Code: `var __v6=document.createElement('canvas');__v6.width=280;__v6.height=60;var __v7=__v6.getContext('2d');__v7.textBaseline='alphabetic';__v7.fillStyle='#f60';__v7.fillRect(125,1,62,20);__v7.fillStyle='#069';__v7.font='11pt Arial';__v7.fillText('Cwm fjordbank glyphs vext quiz, \ud83d\ude03',2,15);__v7.fillStyle='rgba(102,204,0,0.7)';__v7.font='18pt Arial';__v7.fillText('Cwm fjordbank glyphs',4,45);var __v8=__v6.toDataURL();var __v9=__v8.split(',')[1]||__v8`,
			ReturnVar: "__v9",
		},
		{
			Name: "audio",
			Code: `var __v10=window.OfflineAudioContext||window.webkitOfflineAudioContext;if(__v10){try{var __v11=new __v10(1,44100,44100);var __v12=__v11.createOscillator();__v12.type='triangle';__v12.frequency.setValueAtTime(1e4,__v11.currentTime);var __v13=__v11.createDynamicsCompressor();__v13.threshold.setValueAtTime(-50,__v11.currentTime);__v13.knee.setValueAtTime(40,__v11.currentTime);__v13.ratio.setValueAtTime(12,__v11.currentTime);__v13.attack.setValueAtTime(0,__v11.currentTime);__v13.release.setValueAtTime(0.25,__v11.currentTime);__v12.connect(__v13);__v13.connect(__v11.destination);__v11.startRendering();__v11.oncomplete=function(__v14){var __v15=__v14.renderedBuffer.getChannelData(0).slice(4500,5e3).reduce(function(__v16,__v17){return __v16+Math.abs(__v17)},0).toString();__v18=__v15};__v11.onerror=function(){__v18='audio_err'}}catch(__v19){__v18='audio_err'}}else{__v18='no_audio'}`,
			ReturnVar: "__v18",
		},
		{
			Name: "screen",
			Code: `var __v20=window.screen.width+'x'+window.screen.height+'x'+window.screen.colorDepth+'x'+window.screen.pixelDepth+'x'+window.devicePixelRatio`,
			ReturnVar: "__v20",
		},
		{
			Name: "timezone",
			Code: `var __v21=Intl.DateTimeFormat().resolvedOptions().timeZone+'|'+new Date().getTimezoneOffset()`,
			ReturnVar: "__v21",
		},
		{
			Name: "languages",
			Code: `var __v22=navigator.language+'|'+(navigator.languages?navigator.languages.join(','):'')+'|'+navigator.acceptHeaders`,
			ReturnVar: "__v22",
		},
		{
			Name: "platform",
			Code: `var __v23=navigator.platform+'|'+navigator.oscpu+'|'+navigator.vendor+'|'+navigator.productSub`,
			ReturnVar: "__v23",
		},
		{
			Name: "fonts",
			Code: `var __v24=function(){var __v25=['monospace','sans-serif','serif'];var __v26=['Arial','Helvetica','Times New Roman','Courier New','Verdana','Georgia','Comic Sans MS','Impact','Lucida Console','Tahoma','Trebuchet MS','Palatino','Garamond','Bookman','Futura','Optima','Candara','Calibri','Cambria','Corbel','Segoe UI','Roboto','Open Sans','Lato','Montserrat','Source Sans Pro','Raleway','Ubuntu','Noto Sans','Droid Sans'];var __v27={};var __v28=document.createElement('div');__v28.style.cssText='position:absolute;left:-9999px;top:-9999px;visibility:hidden;white-space:nowrap;font-size:72px';document.body.appendChild(__v28);for(var __v29=0;__v29<__v25.length;__v29++){var __v30=document.createElement('span');__v30.style.cssText='font-family:'+__v25[__v29]+';position:absolute;left:-9999px;top:-9999px;visibility:hidden;white-space:nowrap;font-size:72px';__v30.textContent='mmmmmmmmmmlli';__v28.appendChild(__v30);__v27[__v25[__v29]]=__v30.offsetWidth}var __v31=[];for(var __v32=0;__v32<__v26.length;__v32++){var __v33=document.createElement('span');__v33.style.cssText='font-family:'+__v26[__v32]+','+__v25.join(',')+';position:absolute;left:-9999px;top:-9999px;visibility:hidden;white-space:nowrap;font-size:72px';__v33.textContent='mmmmmmmmmmlli';__v28.appendChild(__v33);for(var __v34=0;__v34<__v25.length;__v34++){if(__v33.offsetWidth!==__v27[__v25[__v34]]){__v31.push(__v26[__v32]);break}}}document.body.removeChild(__v28);return __v31.join(',')}()`,
			ReturnVar: "__v24",
		},
		{
			Name: "webrtc",
			Code: `var __v35=window.RTCPeerConnection||window.webkitRTCPeerConnection||window.mozRTCPeerConnection;if(__v35){var __v36=new __v35({iceServers:[{urls:'stun:stun.l.google.com:19302'}]});__v36.createDataChannel('');__v36.createOffer(function(__v37){__v36.setLocalDescription(__v37,function(){try{__v36.localDescription.sdp.split('\\n').forEach(function(__v38){if(__v38.indexOf('candidate')>-1){var __v39=__v38.split(' ');if(__v39[4]&&__v39[4]!=='0.0.0.0'){__v40=__v39[4]}}})}catch(__v41){__v40='webrtc_err'}})});setTimeout(function(){if(!__v40){__v40='webrtc_timeout'}},3e3)}else{__v40='no_webrtc'}`,
			ReturnVar: "__v40",
		},
		{
			Name: "battery",
			Code: `if(navigator.getBattery){navigator.getBattery().then(function(__v42){__v43=__v42.level+'|'+__v42.charging+'|'+__v42.chargingTime+'|'+__v42.dischargingTime})}else{__v43='no_battery'}`,
			ReturnVar: "__v43",
		},
		{
			Name: "memory",
			Code: `var __v44=navigator.deviceMemory||'unknown'`,
			ReturnVar: "__v44",
		},
		{
			Name: "cpu",
			Code: `var __v45=navigator.hardwareConcurrency||'unknown'`,
			ReturnVar: "__v45",
		},
		{
			Name: "touch",
			Code: `var __v46='ontouchstart'in window||navigator.maxTouchPoints>0?'touch:'+navigator.maxTouchPoints:'no_touch'`,
			ReturnVar: "__v46",
		},
		{
			Name: "webdriver",
			Code: `var __v47=navigator.webdriver===true||navigator.webdriver!==undefined?'wd:'+navigator.webdriver:'no_wd'`,
			ReturnVar: "__v47",
		},
		{
			Name: "dnt",
			Code: `var __v48=navigator.doNotTrack||window.doNotTrack||navigator.msDoNotTrack||'unknown'`,
			ReturnVar: "__v48",
		},
		{
			Name: "storage",
			Code: `var __v49='';try{__v49+='ls:'+('localStorage'in window?'1':'0')};try{__v49+='|ss:'+('sessionStorage'in window?'1':'0')};try{__v49+='|idb:'+('indexedDB'in window?'1':'0')};try{__v49+='|odb:'+('openDatabase'in window?'1':'0')};try{__v49+='|cw:'+('cache'in window?'1':'0')}`,
			ReturnVar: "__v49",
		},
		{
			Name: "mime",
			Code: `var __v50=navigator.mimeTypes?Array.from(navigator.mimeTypes).map(function(__v51){return __v51.type}).join(','):'no_mime'`,
			ReturnVar: "__v50",
		},
		{
			Name: "plugins",
			Code: `var __v52=navigator.plugins?Array.from(navigator.plugins).map(function(__v53){return __v53.name}).join(','):'no_plugins'`,
			ReturnVar: "__v52",
		},
		{
			Name: "color",
			Code: `var __v54=window.screen.colorDepth+'|'+window.screen.pixelDepth+'|'+window.devicePixelRatio`,
			ReturnVar: "__v54",
		},
		{
			Name: "pixel_ratio",
			Code: `var __v55=window.matchMedia?'mq:'+window.matchMedia('(resolution: '+window.devicePixelRatio+'dppx)').matches:'no_mq'`,
			ReturnVar: "__v55",
		},
		{
			Name: "nav_props",
			Code: `var __v56=navigator.cookieEnabled+'|'+navigator.onLine+'|'+navigator.product+'|'+navigator.vendorSub`,
			ReturnVar: "__v56",
		},
		{
			Name: "performance",
			Code: `var __v57=window.performance&&window.performance.timing?window.performance.timing.domLoading-window.performance.timing.navigationStart+'|'+window.performance.timing.domComplete-window.performance.timing.domLoading:'no_perf'`,
			ReturnVar: "__v57",
		},
		{
			Name: "connection",
			Code: `var __v58=navigator.connection?navigator.connection.effectiveType+'|'+navigator.connection.downlink+'|'+navigator.connection.rtt:'no_conn'`,
			ReturnVar: "__v58",
		},
		{
			Name: "adblock",
			Code: `var __v59=document.createElement('div');__v59.innerHTML='&nbsp;';__v59.className='adsbox';__v59.style.cssText='position:absolute;left:-9999px;top:-9999px;width:1px;height:1px';document.body.appendChild(__v59);var __v60=__v59.offsetHeight===0?'adblock':'no_adblock';document.body.removeChild(__v59)`,
			ReturnVar: "__v60",
		},
		{
			Name: "math",
			Code: `var __v61=Math.sin(Math.PI/3)+'|'+Math.tan(1e7)+'|'+Math.log10(100)+'|'+Math.asin(0.5)+'|'+Math.atan2(1,2)`,
			ReturnVar: "__v61",
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

	count := 5 + rand.Intn(4)
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

	if req.Timestamp > 0 {
		now := time.Now().UnixMilli()
		diff := now - req.Timestamp
		if diff < -60000 || diff > 600000 {
			req.RiskScore = math.Min(req.RiskScore+20, 100)
		}
	}

	if req.Fingerprint != "" {
		hash := md5.Sum([]byte(req.Fingerprint))
		req.DetectionID = hex.EncodeToString(hash[:])[:16]
	}

	session := &DetectionSession{
		ID:          req.DetectionID,
		RiskScore:   req.RiskScore,
		Chain:       req.Chain,
		Fingerprint: req.Fingerprint,
		SessionID:   req.SessionID,
		Timestamp:   req.Timestamp,
		CreatedAt:   time.Now(),
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
		"success":    true,
		"risk_score": req.RiskScore,
	})
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