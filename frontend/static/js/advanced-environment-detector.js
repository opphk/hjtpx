(function(window,document){
'use strict';
var _0xqwerty={_0x:'_0x'};
var _0xabc=['a','b','c','d','e','f','g','h','i','j','k','l','m','n','o','p','q','r','s','t','u','v','w','x','y','z'];
var _0xhash=function(_0xstr){
var _0xsum=0;
for(var _0xi=0;_0xi<_0xstr['length'];_0xi++){
_0xsum=((_0xsum<<5)-_0xsum)+_0xstr['charCodeAt'](_0xi);
_0xsum=_0xsum&_0xsum;
}
return Math['abs'](_0xsum);
};
var _0xencode=function(_0xstr){
var _0xresult='';
var _0xkeys=_0xhash(Date['now']()['toString']())['toString']();
for(var _0xi=0;_0xi<_0xstr['length'];_0xi++){
var _0xchar=_0xstr['charCodeAt'](_0xi);
var _0xkeychar=_0xkeys[_0xi%_0xkeys['length']]['charCodeAt'](0);
_0xresult+=String['fromCharCode']((_0xchar^_0xkeychar));
}
return btoa(_0xresult);
};
var _0xobfuscate=function(_0xstr){
var _0xbase=btoa(encodeURIComponent(_0xstr));
var _0xhashval=_0xhash(_0xbase);
return _0xhashval['toString'](36)+btoa(_0xbase)['substring'](0,16);
};
var AdvancedEnvironmentDetector=(function(){
function _0xDetector(_0xoptions){
this['_0xoptions']=Object['assign']({
apiBase:'/api/v1',
sampleRate:0.3,
chainCount:12,
enableAll:true,
sessionId:null,
obfuscate:true
},_0xoptions);
this['_0xresults']={};
this['_0xscore']=0;
this['_0xchain']=[];
this['_0xid']=_0xgenerateId();
this['_0xweights']={
canvas:8,
webgl:10,
webgl2:8,
audio:9,
fonts:7,
webrtc_ip:10,
webdriver:15,
selenium:18,
puppeteer:18,
playwright:18,
chrome_runtime:10,
headless:12,
permissions:6,
plugins:5,
languages:4,
timezone:5,
screen:3,
hardware:4,
memory:3,
storage:5,
navigator:4,
window_props:4,
iframe:6,
notification:3,
battery:3,
media_devices:4,
connection:5,
adblock:4,
math:3,
gpu:6,
speech:3,
browser_engine:10,
vm_detection:15,
cloud_detection:12,
container:8,
tor:8
};
}
function _0xgenerateId(){
var _0xtimestamp=Date['now']()['toString'](36);
var _0xrandom=Math['random']()['toString'](36)['substring'](2,8);
return 'det_'+_0xtimestamp+'_'+_0xrandom;
}
function _0xgetMethods(){
return[
'detectBrowserEngine',
'detectHeadless',
'detectWebDriver',
'detectPuppeteer',
'detectPlaywright',
'detectSelenium',
'detectChromeRuntime',
'detectPermissions',
'detectPlugins',
'detectLanguages',
'detectTimezone',
'detectScreen',
'detectHardwareConcurrency',
'detectDeviceMemory',
'detectStorage',
'detectCanvas',
'detectWebGL',
'detectWebGL2',
'detectAudio',
'detectFonts',
'detectNavigatorProps',
'detectWindowProps',
'detectIframe',
'detectNotification',
'detectBattery',
'detectMediaDevices',
'detectWebRTCIP',
'detectConnection',
'detectAdBlock',
'detectMathFingerprint',
'detectGPUFingerprint',
'detectSpeech',
'detectVMIndicators',
'detectCloudEnvironment',
'detectContainerEnvironment',
'detectTorBrowser'
];
}
function _0xgenerateChain(_0xcount){
var _0xmethods=_0xgetMethods();
var _0xshuffled=[..._0xmethods]['sort'](function(){return Math['random']()-0.5;});
var _0xselected=_0xshuffled['slice'](0,Math['min'](_0xcount,_0xmethods['length']));
var _0xaliases={};
_0xselected['forEach'](function(_0xmethod,_0xi){
_0xaliases[_0xmethod]='chk_'+_0xi['toString'](36)+'_'+Math['random']()['toString'](36)['substring'](2,6);
});
return{selected:_0xselected,aliases:_0xaliases};
}
function _0xrunChain(){
var _0xself=this;
return new Promise(function(_0xresolve){
var _0xchainData=_0xgenerateChain(_0xself['_0xoptions']['chainCount']);
_0xself['_0xchain']=_0xchainData['selected'];
var _0xchainResults={};
var _0xstartTime=performance['now']();
var _0xpromises=[];
_0xchainData['selected']['forEach'](function(_0xmethod){
var _0xpromise=(function(_0xm){
return new Promise(function(_0xres){
try{
var _0xresult=_0xself[_0xm]();
if(_0xresult instanceof Promise){
_0xresult['then'](function(_0xr){
_0xchainResults[_0xchainData['aliases'][_0xm]]=_0xr;
_0xself['_0xresults'][_0xm]=_0xr;
_0xres();
})['catch'](function(){
_0xchainResults[_0xchainData['aliases'][_0xm]]={detected:false,score:0,error:true};
_0xres();
});
}else{
_0xchainResults[_0xchainData['aliases'][_0xm]]=_0xresult;
_0xself['_0xresults'][_0xm]=_0xresult;
_0xres();
}
}catch(_0xe){
_0xchainResults[_0xchainData['aliases'][_0xm]]={detected:false,score:0,error:true};
_0xres();
}
});
})(_0xmethod);
_0xpromises['push'](_0xpromise);
});
Promise['all'](_0xpromises)['then'](function(){
var _0xduration=performance['now']()-_0xstartTime;
_0xself['_0xscore']=_0xcalculateScore();
_resolve({
detection_id:_0xself['_0xid'],
chain:_0xchainResults,
chain_order:Object['values'](_0xchainData['aliases']),
risk_score:_0xself['_0xscore'],
duration_ms:Math['round'](_0xduration),
timestamp:Date['now']()
});
});
});
}
function _0xcalculateScore(){
var _0xself=this;
var _0xweightedScore=0;
var _0xtotalWeight=0;
for(var _0xkey in _0xself['_0xresults']){
var _0xresult=_0xself['_0xresults'][_0xkey];
if(_0xresult&&typeof _0xresult['score']==='number'){
var _0xweight=_0xself['_0xweights'][_0xkey]||5;
_0xweightedScore+=_0xresult['score']*_0xweight;
_0xtotalWeight+=_0xweight;
}
}
if(_0xtotalWeight===0)return 0;
var _0xbaseScore=_0xweightedScore/_0xtotalWeight;
var _0xautoTools=['detectWebDriver','detectPuppeteer','detectPlaywright','detectSelenium'];
var _0xautoDetected=_0xautoTools['filter'](function(_0xm){
var _0xr=_0xself['_0xresults'][_0xm];
return _0xr&&_0xr['detected']===true;
})['length'];
if(_0xautoDetected>=2){
_0xbaseScore=Math['min'](_0xbaseScore*1.5+20,100);
}else if(_0xautoDetected>=1){
_0xbaseScore=Math['min'](_0xbaseScore*1.3+10,100);
}
return Math['round'](Math['min'](Math['max'](_0xbaseScore,0),100));
}
_0xDetector['prototype']['runChain']=_0xrunChain;
_0xDetector['prototype']['detectBrowserEngine']=async function(){
var _0xresult={detected:false,score:0,data:{}};
try{
var _0xua=navigator['userAgent']||'';
var _0xuaLower=_0xua['toLowerCase']();
var _0xengine='unknown';
var _0xengineVersion='';
var _0xbrowser='unknown';
var _0xbrowserVersion='';
var _0xchromeMatch=_0xua['match'](/chrome\/([\d.]+)/);
var _0xedgeMatch=_0xua['match'](/edg[e]?\/([\d.]+)/);
var _0xffMatch=_0xua['match'](/firefox\/([\d.]+)/);
var _0xsafariMatch=_0xua['match'](/version\/([\d.]+)/);
if(_0xedgeMatch){
_0xbrowser='edge';
_0xbrowserVersion=_0xedgeMatch[1];
_0xengine='blink';
_0xengineVersion=_0xchromeMatch?_0xchromeMatch[1]['split']('.')[0]:'';
}else if(_0xchromeMatch){
if(_0xuaLower['indexOf']('edg/')>-1){
_0xbrowser='edge';
_0xengine='blink';
}else{
_0xbrowser='chrome';
_0xengine='blink';
_0xengineVersion=_0xchromeMatch[1]['split']('.')[0];
}
_0xbrowserVersion=_0xchromeMatch[1];
}else if(_0xffMatch){
_0xbrowser='firefox';
_0xengine='gecko';
_0xbrowserVersion=_0xffMatch[1];
}
_0xresult['data']={
engine:_0xengine,
engineVersion:_0xengineVersion,
browser:_0xbrowser,
browserVersion:_0xbrowserVersion
};
}catch(_0xe){}
return _0xresult;
};
_0xDetector['prototype']['detectHeadless']=async function(){
var _0xscore=0;
var _0xdetections=[];
try{
if(navigator['webdriver']===true){
_0xscore+=30;
_0xdetections['push']('webdriver_true');
}
}catch(_0xe){}
try{
if(navigator['plugins']&&navigator['plugins']['length']===0){
_0xscore+=15;
_0xdetections['push']('no_plugins');
}
}catch(_0xe){}
try{
var _0xua=navigator['userAgent']||'';
if(/headless|phantom/i['test'](_0xua)){
_0xscore+=35;
_0xdetections['push']('headless_ua');
}
}catch(_0xe){}
try{
if(window['outerHeight']===0&&window['outerWidth']===0){
_0xscore+=25;
_0xdetections['push']('zero_window_size');
}
}catch(_0xe){}
return{detected:_0xscore>30,score:Math['min'](_0xscore,100),detections:_0xdetections};
};
_0xDetector['prototype']['detectWebDriver']=async function(){
var _0xscore=0;
var _0xdetections=[];
try{
if(navigator['webdriver']===true){
_0xscore+=30;
_0xdetections['push']('navigator.webdriver');
}
}catch(_0xe){}
return{detected:_0xscore>20,score:Math['min'](_0xscore,100),detections:_0xdetections};
};
_0xDetector['prototype']['detectPuppeteer']=async function(){
var _0xscore=0;
var _0xdetections=[];
try{
if(document['$cdc_asdjflasutopfhvcZLmcfl_']){
_0xscore+=35;
_0xdetections['push']('cdc_marker');
}
}catch(_0xe){}
try{
var _0xua=navigator['userAgent']||'';
if(/puppet/i['test'](_0xua)){
_0xscore+=40;
_0xdetections['push']('puppeteer_ua');
}
}catch(_0xe){}
return{detected:_0xscore>30,score:Math['min'](_0xscore,100),detections:_0xdetections};
};
_0xDetector['prototype']['detectPlaywright']=async function(){
var _0xscore=0;
var _0xdetections=[];
try{
if(window['__playwright__']!==undefined||window['__pw_resume__']!==undefined){
_0xscore+=45;
_0xdetections['push']('playwright_global');
}
}catch(_0xe){}
try{
var _0xua=navigator['userAgent']||'';
if(/playwright/i['test'](_0xua)){
_0xscore+=50;
_0xdetections['push']('playwright_ua');
}
}catch(_0xe){}
return{detected:_0xscore>30,score:Math['min'](_0xscore,100),detections:_0xdetections};
};
_0xDetector['prototype']['detectSelenium']=async function(){
var _0xscore=0;
var _0xdetections=[];
try{
var _0xua=navigator['userAgent']||'';
if(/selenium/i['test'](_0xua)){
_0xscore+=40;
_0xdetections['push']('selenium_ua');
}
}catch(_0xe){}
return{detected:_0xscore>20,score:Math['min'](_0xscore,100),detections:_0xdetections};
};
_0xDetector['prototype']['detectCanvas']=async function(){
var _0xscore=0;
var _0xdetections=[];
try{
var _0xcanvas=document['createElement']('canvas');
_0xcanvas['width']=280;
_0xcanvas['height']=80;
var _0xctx=_0xcanvas['getContext']('2d');
if(!_0xctx){
_0xscore+=40;
_0xdetections['push']('no_canvas_context');
return{detected:true,score:Math['min'](_0xscore,100),detections:_0xdetections};
}
_0xctx['textBaseline']='alphabetic';
_0xctx['fillStyle']='#f60';
_0xctx['fillRect'](125,1,62,20);
_0xctx['fillStyle']='#069';
_0xctx['font']='11pt Arial';
_0xctx['fillText']('Cwm fjordbank glyphs vext quiz',2,15);
var _0xdataURL=_0xcanvas['toDataURL']();
var _0ximageData=_0xctx['getImageData'](0,0,10,10);
var _0xpixelSum=Array['prototype']['slice']['call'](_0ximageData['data']['slice'](0,40))['reduce'](function(_0xa,_0xb){return _0xa+_0xb;},0);
if(_0xpixelSum===0){
_0xscore+=20;
_0xdetections['push']('canvas_empty_readback');
}
}catch(_0xe){
_0xscore+=35;
_0xdetections['push']('canvas_error');
}
return{detected:_0xscore>30,score:Math['min'](_0xscore,100),detections:_0xdetections};
};
_0xDetector['prototype']['detectWebGL']=async function(){
var _0xscore=0;
var _0xdetections=[];
try{
var _0xcanvas=document['createElement']('canvas');
var _0xgl=_0xcanvas['getContext']('webgl')||_0xcanvas['getContext']('experimental-webgl');
if(!_0xgl){
_0xscore+=40;
_0xdetections['push']('no_webgl');
return{detected:true,score:Math['min'](_0xscore,100),detections:_0xdetections};
}
var _0xdebugInfo=_0xgl['getExtension']('WEBGL_debug_renderer_info');
if(_0xdebugInfo){
var _0xrenderer=_0xgl['getParameter'](_0xdebugInfo['UNMASKED_RENDERER_WEBGL']);
if(/swiftshader|llvmpipe|mesa|virtual|google\s*inc/i['test'](_0xrenderer||'')){
_0xscore+=30;
_0xdetections['push']('software_renderer');
}
}else{
_0xscore+=20;
_0xdetections['push']('no_webgl_debug');
}
}catch(_0xe){
_0xscore+=35;
_0xdetections['push']('webgl_error');
}
return{detected:_0xscore>30,score:Math['min'](_0xscore,100),detections:_0xdetections};
};
_0xDetector['prototype']['detectAudio']=async function(){
var _0xscore=0;
var _0xdetections=[];
try{
var _0xAudioContext=window['OfflineAudioContext']||window['webkitOfflineAudioContext'];
if(!_0xAudioContext){
_0xscore+=30;
_0xdetections['push']('no_audiocontext');
return{detected:true,score:Math['min'](_0xscore,100),detections:_0xdetections};
}
var _0xctx=new _0xAudioContext(1,44100,44100);
var _0xosc=_0xctx['createOscillator']();
_0xosc['type']='triangle';
_0xosc['frequency']['setValueAtTime'](10000,_0xctx['currentTime']);
var _0xcompressor=_0xctx['createDynamicsCompressor']();
_0xosc['connect'](_0xcompressor);
_0xcompressor['connect'](_0xctx['destination']);
_0xosc['start'](0);
var _0xbuffer=await _0xctx['startRendering']();
var _0xchannelData=_0xbuffer['getChannelData'](0);
var _0xsumAbs=0;
for(var _0xi=4500;_0xi<5000;_0xi++){
_0xsumAbs+=Math['abs'](_0xchannelData[_0xi]);
}
if(_0xsumAbs===0){
_0xscore+=25;
_0xdetections['push']('audio_silent');
}
}catch(_0xe){
_0xscore+=30;
_0xdetections['push']('audio_error');
}
return{detected:_0xscore>25,score:Math['min'](_0xscore,100),detections:_0xdetections};
};
_0xDetector['prototype']['detectVMIndicators']=async function(){
var _0xscore=0;
var _0xdetections=[];
var _0xvmKeywords=['VirtualBox','VMware','QEMU','KVM','Xen','Hyper-V','Parallels'];
try{
var _0xcanvas=document['createElement']('canvas');
var _0xgl=_0xcanvas['getContext']('webgl');
if(_0xgl){
var _0xdebugInfo=_0xgl['getExtension']('WEBGL_debug_renderer_info');
if(_0xdebugInfo){
var _0xrenderer=_0xgl['getParameter'](_0xdebugInfo['UNMASKED_RENDERER_WEBGL'])||'';
for(var _0xi=0;_0xi<_0xvmKeywords['length'];_0xi++){
if(_0xrenderer['toLowerCase']()['indexOf'](_0xvmKeywords[_0xi]['toLowerCase']())>-1){
_0xscore+=40;
_0xdetections['push']('vm_detected:'+_0xvmKeywords[_0xi]);
break;
}
}
if(/swiftshader|llvmpipe|mesa|software/i['test'](_0xrenderer)){
_0xscore+=30;
_0xdetections['push']('software_rendering');
}
}
}
}catch(_0xe){}
try{
if(screen['width']===0||screen['height']===0){
_0xscore+=25;
_0xdetections['push']('zero_screen_size');
}
}catch(_0xe){}
return{detected:_0xscore>25,score:Math['min'](_0xscore,100),detections:_0xdetections};
};
_0xDetector['prototype']['detectCloudEnvironment']=async function(){
var _0xscore=0;
var _0xdetections=[];
var _0xcloudKeywords=['aws','amazon','gcp','google cloud','azure','microsoft','digitalocean','linode','vultr'];
try{
var _0xua=navigator['userAgent']||'';
_0xua=_0xua['toLowerCase']();
for(var _0xi=0;_0xi<_0xcloudKeywords['length'];_0xi++){
if(_0xua['indexOf'](_0xcloudKeywords[_0xi])>-1){
_0xscore+=30;
_0xdetections['push']('cloud_in_ua:'+_0xcloudKeywords[_0xi]);
break;
}
}
}catch(_0xe){}
return{detected:_0xscore>25,score:Math['min'](_0xscore,100),detections:_0xdetections};
};
_0xDetector['prototype']['detectContainerEnvironment']=async function(){
var _0xscore=0;
var _0xdetections=[];
try{
var _0xcores=navigator['hardwareConcurrency'];
if(_0xcores===1||_0xcores===2){
_0xscore+=20;
_0xdetections['push']('low_cpu_cores');
}
}catch(_0xe){}
try{
var _0xmem=navigator['deviceMemory'];
if(_0xmem!==undefined&&_0xmem<=0.5){
_0xscore+=25;
_0xdetections['push']('low_device_memory');
}
}catch(_0xe){}
return{detected:_0xscore>20,score:Math['min'](_0xscore,100),detections:_0xdetections};
};
_0xDetector['prototype']['detectTorBrowser']=async function(){
var _0xscore=0;
var _0xdetections=[];
try{
var _0xua=navigator['userAgent']||'';
if(/tor|onion/i['test'](_0xua)){
_0xscore+=30;
_0xdetections['push']('tor_in_ua');
}
}catch(_0xe){}
return{detected:_0xscore>25,score:Math['min'](_0xscore,100),detections:_0xdetections};
};
_0xDetector['prototype']['generateFingerprint']=function(){
var _0xcomponents=[];
try{
_0xcomponents['push']('scrn:'+screen['width']+'x'+screen['height']+'x'+screen['colorDepth']);
}catch(_0xe){}
try{
_0xcomponents['push']('lang:'+(navigator['language']||''));
}catch(_0xe){}
try{
_0xcomponents['push']('cpu:'+(navigator['hardwareConcurrency']||''));
}catch(_0xe){}
return _0xcomponents['join']('|');
};
_0xDetector['prototype']['runAll']=async function(){
var _0xchainResult=await this['runChain']();
var _0xfingerprint=this['generateFingerprint']();
return Object['assign'](_0xchainResult,{fingerprint:_0xfingerprint});
};
_0xDetector['prototype']['toJSON']=function(){
return{
risk_score:this['_0xscore'],
chain_count:this['_0xchain']['length'],
results:this['_0xresults']
};
};
return _0xDetector;
})();
if(typeof window!=='undefined'){
window['AdvancedEnvironmentDetector']=AdvancedEnvironmentDetector;
window['_0xq']={encode:_0xencode,obfuscate:_0xobfuscate,hash:_0xhash};
}
})(window,document);
