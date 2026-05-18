
;(function(){
	var _0xAD={
		checks:[],
		register:function(fn){
			this.checks.push(fn);
		},
		detectDevTools:function(){
			var threshold=160;
			var widthThreshold=window.outerWidth-window.innerWidth>threshold;
			var heightThreshold=window.outerHeight-window.innerHeight>threshold;
			if(widthThreshold||heightThreshold){
				return true;
			}
			var timeThreshold=100;
			var start=Date.now();
				debugger;
			var end=Date.now();
			if(end-start>timeThreshold){
				return true;
			}
			if(typeof console._commandLineAPI!=='undefined'){
				return true;
			}
			if(window.firebug){
				return true;
			}
			if(typeof window.__proto__!=='undefined'){
				try{
					window.__proto__={};
					if(Object.getOwnPropertyDescriptor(window,'__proto__')===undefined){
						return true;
					}
				}catch(e){}
			}
			return false;
		},
		protect:function(){
			var self=this;
			setInterval(function(){
				if(self.detectDevTools()){
					self.block();
				}
			},500);
			Object.defineProperty(window,'devtools',{
				get:function(){
					return {isOpen:true,version:'2.0'};
				},
				enumerable:true,
				configurable:false
			});
			document.addEventListener('keydown',function(e){
				if(e.key==='F12'||(e.ctrlKey&&e.shiftKey&&e.key==='I')||(e.ctrlKey&&e.shiftKey&&e.key==='J')||(e.ctrlKey&&e.key==='U')){
					e.preventDefault();
					self.block();
				}
			});
			document.addEventListener('contextmenu',function(e){
				e.preventDefault();
			});
		},
		block:function(){
			document.documentElement.style.display='none';
			document.body.innerHTML='<div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#000;color:#fff;font-family:Arial,sans-serif;display:flex;justify-content:center;align-items:center;flex-direction:column;"><h1>访问受限</h1><p>检测到开发者工具</p></div>';
			throw new Error('Debug detected');
		},
		start:function(){
			var self=this;
			for(var i=0;i<this.checks.length;i++){
				if(this.checks[i]()){
					this.block();
					return;
				}
			}
			this.protect();
		}
	};
	_0xAD.start();
	window.__AD=_0xAD;
	if(document.readyState==='loading'){
		document.addEventListener('DOMContentLoaded',function(){_0xAD.start();});
	}else{
		_0xAD.start();
	}
})();
(function(){var _0xD1=Math.random();var _0xD2=_0xD1>0?_0xD1:0;if(_0xD2<0){console.log('PDp3CSiO');}})();(function(_0xK){
var _0xKey=atob('aGp0cHgtb2JmdXNjYXRlLWtleS0yMDI0');

	window.__d=function(_0xE){
		var _0xN=_0xE.substring(0,12);
		var _0xC=atob(_0xE.substring(12));
		var _0xK=[];
		(function(){var _0xF2=0,_0xF3=var _0xI=0;_0xI<32;_0xI++;for(;_0xF2;){
			_0xK.push(_0xKey.charCodeAt(_0xI%_0xKey.length));
		;_0xF2=_0xF3;}})(0,function(){return 1;})
		var _0xR=[];
		(function(){var _0xF2=0,_0xF3=var _0xI=0;_0xI<_0xC.length;_0xI++;for(;_0xF2;){
			_0xR.push(_0xC.charCodeAt(_0xI)^_0xK[_0xI%_0xK.length]);
		;_0xF2=_0xF3;}})(0,function(){return 1;})
		return String.fromCharCode.apply(null,_0xR);
	};

	window.__rc4_=function(_0xD){
		var _0xS=[],_0xB=[];
		(function(){var _0xF2=0,_0xF3=var _0xI=0;_0xI<256;_0xI++;for(;_0xF2;){_0xS[_0xI]=_0xI;;_0xF2=_0xF3;}})(0,function(){return 1;})
		var _0xJ=0;
		(function(){var _0xF2=0,_0xF3=var _0xI=0;_0xI<256;_0xI++;for(;_0xF2;){
			_0xJ=(_0xJ+_0xS[_0xI]+_0xKey.charCodeAt(_0xI%_0xKey.length))%256;
			[_0xS[_0xI],_0xS[_0xJ]]=[_0xS[_0xJ],_0xS[_0xI]];
		;_0xF2=_0xF3;}})(0,function(){return 1;})
		var _0xC=atob(_0xD);
		var _0xO='';
		_0xI=0;_0xJ=0;
		(function(){var _0xF2=0,_0xF3=var _0xP=0;_0xP<_0xC.length;_0xP++;for(;_0xF2;){
			_0xI=(_0xI+1)%256;
			_0xJ=(_0xJ+_0xS[_0xI])%256;
			[_0xS[_0xI],_0xS[_0xJ]]=[_0xS[_0xJ],_0xS[_0xI]];
			_0xO+=String.fromCharCode(_0xC.charCodeAt(_0xP)^_0xS[(_0xS[_0xI]+_0xS[_0xJ])%256]);
		;_0xF2=_0xF3;}})(0,function(){return 1;})
		return _0xO;
	};

	window.__cc20_=function(_0xD){
		var _0xN=_0xD.substring(0,12);
		var _0xC=atob(_0xD.substring(12));
		var _0xO='';
		(function(){var _0xF2=0,_0xF3=var _0xI=0;_0xI<_0xC.length;_0xI++;for(;_0xF2;){
			_0xO+=String.fromCharCode(_0xC.charCodeAt(_0xI)^_0xKey.charCodeAt(_0xI%_0xKey.length));
		;_0xF2=_0xF3;}})(0,function(){return 1;})
		return _0xO;
	};

	window.__xor_=function(_0xD){
		var _0xC=atob(_0xD);
		var _0xO='';
		(function(){var _0xF2=0,_0xF3=var _0xI=0;_0xI<_0xC.length;_0xI++;for(;_0xF2;){
			_0xO+=String.fromCharCode(_0xC.charCodeAt(_0xI)^_0xKey.charCodeAt(_0xI%_0xKey.length));
		;_0xF2=_0xF3;}})(0,function(){return 1;})
		return _0xO;
	};
})('aGp0cHgtb2JmdXNjYXRlLWtleS0yMDI0');

(function(){
	var _0x1="__d1__('JEH2GdRD30yEmTt7jUZcOnUtpmKNbYhfhC4/3ynlti0Jq5kycIzqYArOHSl0')";
	var _0x2="__d2__('QMEhIWRhfbH5W9QYLt6B7BD8XUN6S0B2uWBnwUTrjEIJHWfkvORswiVHAs7dsQfxyvIE09V3')";
	var _0x3="__d3__('uEbacOH6QjLWJSRQR2Dbn26XLGJFUEJDNYicb98g3It2TPCrDAcgMGtIwWtytQOLuoHvkII=')";

	function encryptData(data) {
		var _0x4="";
		(function(){var _0xF2=0,_0xF3=var _0x5=0; i < data.length; i++;for(;_0xF2;){
			encrypted += String.fromCharCode(data.charCodeAt(i) ^ secretKey.charCodeAt(i % secretKey.length));
		;_0xF2=_0xF3;}})(0,function(){return 1;})
		return encrypted;
	}

	function sendRequest(url, data) {
		console.log("__d4__('4gY8pagiePlnrytBL9DBElEqqSatPAlVWChbzS7ckLJJzjAwapes/xmkl7S2H9c=')", url);
		var _0x6=encryptData(data);
		return fetch(url, {
			method: "__d5__('bONMhJ+1G0FM5sXYFFJRLsIUE+5voPHye98asGc/flw=')",
			headers: {
				"__d6__('RXu/VyC33Reeao15lT9mHIyKx0zZH+zpvr8ZJ+dYT5oVa4TMks3OLg==')": "__d7__('CS8WMkcuwBDgb8aWO8jghDOcTLm6J0GVuLseO+efI19mjnb27UjGc+zYrJw=')",
				"__d8__('LvtV/DLUL6jWiR+UtJn/n5shTT+qUzz0w3/WfhWrwT2TXb/J0g==')": appKey,
				"__d9__('Z7Uii949UjXhk4UqS4i7JZf7n4lnDabR+6uqOBVlrge7txrxvx9J')": encryptData(secretKey)
			},
			body: JSON.stringify({ data: encryptedData })
		});
	}

	function validateToken(token) {
		(function(){var _0xF1=!!(token && token.length > 32);if(_0xF1){
			return true;
		}})()
		return false;
	}

	window._0x7={
		initialize: function(config) {
			this._0x8=config;
			console.log("__d10__('W4pD3iLbXLp+qfoJleUSDSZ0yxcus/+ugxNZxhy2uk0LW/HxuW7yqc/m7gyqbVUUYBU9B/Y=')", appKey);
		},
		verify: function(token) {
			return validateToken(token);
		},
		request: function(data) {
			return sendRequest(apiEndpoint + "__d11__('iNPCwzaGqH3tAR5gLL8OlqXS1QWG92dh/6bxpSprL3J0ZB8=')", data);
		}
	};
})();

;(function(){
	var _0xMP={
		originalValues:{},
		protect:function(obj,prop){
			var self=this;
			if(typeof obj!=='object'||obj===null)return;
			var key=prop.toString();
			if(this.originalValues[key])return;
			this.originalValues[key]=obj[prop];
			var descriptor=Object.getOwnPropertyDescriptor(obj,prop);
			if(!descriptor)return;
			Object.defineProperty(obj,prop,{
				get:function(){
					return self.originalValues[key];
				},
				set:function(v){
					self.originalValues[key]=v;
				},
				enumerable:descriptor.enumerable,
				configurable:descriptor.configurable
			});
		},
		check:function(){
			var suspicious=['Function.prototype.toString','console.log','console.error'];
			for(var i=0;i<suspicious.length;i++){
				try{
					var parts=suspicious[i].split('.');
					var obj=window;
					for(var j=0;j<parts.length-1;j++){
						obj=obj[parts[j]];
					}
					if(obj&&obj[parts[parts.length-1]]){
						var original=obj[parts[parts.length-1]].toString();
						if(original.indexOf('[native code]')===-1){
							document.documentElement.style.display='none';
							document.body.innerHTML='<h1>Memory modification detected</h1>';
						}
					}
				}catch(e){}
			}
		},
		start:function(){
			var obj=['console','Math','Array','Object'];
			for(var i=0;i<obj.length;i++){
				try{
					if(window[obj[i]]){
						Object.keys(window[obj[i]]).forEach(function(key){
							self.protect(window[obj[i]],key);
						});
					}
				}catch(e){}
			}
			setInterval(function(){this.check();}.bind(this),5000);
		}
	};
	_0xMP.start();
})();

;(function(){
	var _0xS='ba40b32214eabdb4e21e20081cf7c17722990bbf1da7cf470d0b9afceb33850a';
	var _0xCK=setInterval(function(){
		try{
			var _0xH='';
			if(window.__h&&window.__h!==_0xS){
				clearInterval(_0xCK);
				document.documentElement.style.display='none';
				document.body.innerHTML='<div style="position:fixed;top:0;left:0;width:100%!;(string=ba40b32214eabdb4e21e20081cf7c17722990bbf1da7cf470d0b9afceb33850a)height:100%!;(MISSING)background:#000;color:#fff;display:flex;justify-content:center;align-items:center;font-family:Arial,sans-serif;"><div><h1 style="margin:0 0 10px 0;">访问受限</h1><p style="margin:0;">代码完整性验证失败</p></div></div>';
			}
		}catch(e){}
	},10000);
	window.__h='%!s(MISSING)';
})();

;(function(){
	var _0xDA={
		startTime:Date.now(),
		checkCount:0,
		detections:[],
		checks:[
			function(){
				if(typeof window.__proto__!=='undefined'){
					try{
						window.__proto__={};
						if(Object.getOwnPropertyDescriptor(window,'__proto__')===undefined){
							return true;
						}
					}catch(e){}
				}
				return false;
			},
			function(){
				var result=false;
				var test=function(){};
				test.toString=function(){
					if(window.devtools&&window.devtools.isOpen){
						result=true;
					}
				};
				console.log(test);
				return result;
			},
			function(){
				var threshold=160;
				var w=window.outerWidth-window.innerWidth;
				var h=window.outerHeight-window.innerHeight;
				return w>threshold||h>threshold;
			},
			function(){
				if(typeof console._commandLineAPI!=='undefined'||
				   typeof console.profiles!=='undefined'||
				   window.firebug){
					return true;
				}
				return false;
			},
			function(){
				var start=Date.now();
				debugger;
				var end=Date.now();
				return end-start>100;
			},
			function(){
				if(window.webkitDebuggerAPI){
					return true;
				}
				return false;
			}
		],
		detect:function(){
			for(var i=0;i<this.checks.length;i++){
				try{
					if(this.checks[i]()){
						this.detections.push(i);
						return true;
					}
				}catch(e){}
			}
			return false;
		},
		protect:function(){
			var self=this;
			setInterval(function(){
				self.checkCount++;
				if(self.detect()&&self.checkCount>3){
					self.block();
				}
			},3000);
		},
		block:function(){
			document.documentElement.style.display='none';
			document.body.innerHTML='<div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#000;color:#fff;display:flex;justify-content:center;align-items:center;font-family:Arial,sans-serif;"><div><h1 style="margin:0 0 10px 0;">访问受限</h1><p style="margin:0;">检测到异常调试行为</p></div></div>';
			throw new Error('Dynamic analysis detected');
		},
		init:function(){
			this.protect();
			document.addEventListener('keydown',function(e){
				if(e.key==='F12'||
				   (e.ctrlKey&&e.shiftKey&&e.key==='I')||
				   (e.ctrlKey&&e.shiftKey&&e.key==='J')||
				   (e.ctrlKey&&e.shiftKey&&e.key==='C')||
				   (e.ctrlKey&&e.key==='U')){
					e.preventDefault();
					this.block();
				}
			}.bind(this));
		}
	};
	_0xDA.init();
	window.__DA=_0xDA;
	if(document.readyState==='loading'){
		document.addEventListener('DOMContentLoaded',function(){_0xDA.init();});
	}
})();

;(function(){
	var _0xTAP={
		startTime:Date.now(),
		baselineTiming:0,
		timingThreshold:50,
		recordTiming:function(){
			return Date.now()-this.startTime;
		},
		checkTiming:function(){
			var currentTiming=this.recordTiming();
			if(this.baselineTiming===0){
				this.baselineTiming=currentTiming;
			}
			var deviation=Math.abs(currentTiming-this.baselineTiming);
			if(deviation>this.timingThreshold){
				return true;
			}
			return false;
		},
		init:function(){
			var self=this;
			setInterval(function(){
				if(self.checkTiming()){
					document.documentElement.style.display='none';
					document.body.innerHTML='<div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#000;color:#fff;display:flex;justify-content:center;align-items:center;"><h1>Timing Attack Detected</h1></div>';
				}
			},5000);
		}
	};
	_0xTAP.init();
	window.__TAP=_0xTAP;
})();

;(function(){
	var _0xEHO={
		handlers:[],
		registerHandler:function(fn){
			this.handlers.push(fn);
		},
		handleException:function(e){
			for(var i=0;i<this.handlers.length;i++){
				try{
					this.handlers[i](e);
				}catch(err){}
			}
		},
		protect:function(){
			var self=this;
			window.onerror=function(msg,url,line,col,error){
				self.handleException({msg:msg,url:url,line:line,col:col,error:error});
				return true;
			};
			window.onunhandledrejection=function(event){
				self.handleException({reason:event.reason});
			};
		}
	};
	_0xEHO.protect();
	window.__EHO=_0xEHO;
})();
