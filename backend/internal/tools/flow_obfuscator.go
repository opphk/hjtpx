package tools

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

type FlowObfuscator struct {
	variableMap map[string]string
	functionMap map[string]string
	usedNames   map[string]bool
	seed        int64
}

func NewFlowObfuscator() *FlowObfuscator {
	fo := &FlowObfuscator{
		variableMap: make(map[string]string),
		functionMap: make(map[string]string),
		usedNames:   make(map[string]bool),
		seed:        0,
	}

	var seedBytes [8]byte
	rand.Read(seedBytes[:])
	
	var seed uint64
	for i := 0; i < 8; i++ {
		seed = seed<<8 + uint64(seedBytes[i])
	}
	fo.seed = int64(seed)

	return fo
}

func (fo *FlowObfuscator) generateObfuscatedName(prefix string) string {
	var name [8]byte
	rand.Read(name[:])
	obfuscated := prefix + hex.EncodeToString(name[:])

	for fo.usedNames[obfuscated] {
		rand.Read(name[:])
		obfuscated = prefix + hex.EncodeToString(name[:])
	}

	fo.usedNames[obfuscated] = true
	return obfuscated
}

func (fo *FlowObfuscator) obfuscateVariables(code string) string {
	variableRegex := regexp.MustCompile(`\b(var|let|const)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\b`)
	
	code = variableRegex.ReplaceAllStringFunc(code, func(match string) string {
		parts := variableRegex.FindStringSubmatch(match)
		if len(parts) >= 3 {
		 keyword := parts[1]
			varName := parts[2]
			
			if fo.shouldSkipVariable(varName) {
				return match
			}

			if _, exists := fo.variableMap[varName]; !exists {
				fo.variableMap[varName] = fo.generateObfuscatedName("_0x")
			}

			return keyword + " " + fo.variableMap[varName]
		}
		return match
	})

	return code
}

func (fo *FlowObfuscator) shouldSkipVariable(name string) bool {
	skipList := []string{
		"window", "document", "console", "Math", "JSON", "Array", "Object",
		"String", "Number", "Boolean", "Date", "RegExp", "Error",
		"undefined", "null", "true", "false", "NaN", "Infinity",
		"prototype", "constructor", "toString", "valueOf", "hasOwnProperty",
	}
	
	for _, skip := range skipList {
		if name == skip {
			return true
		}
	}
	
	return false
}

func (fo *FlowObfuscator) obfuscateFunctions(code string) string {
	funcRegex := regexp.MustCompile(`\bfunction\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\s*\(`)
	
	code = funcRegex.ReplaceAllStringFunc(code, func(match string) string {
		parts := funcRegex.FindStringSubmatch(match)
		if len(parts) >= 2 {
			funcName := parts[1]
			
			if fo.shouldSkipFunction(funcName) {
				return match
			}

			if _, exists := fo.functionMap[funcName]; !exists {
				fo.functionMap[funcName] = fo.generateObfuscatedName("_0xf")
			}

			return "function " + fo.functionMap[funcName] + "("
		}
		return match
	})

	return code
}

func (fo *FlowObfuscator) shouldSkipFunction(name string) bool {
	skipList := []string{
		"init", "start", "stop", "run", "execute", "handle",
		"callback", "onClick", "onLoad", "onError", "onSuccess",
	}
	
	for _, skip := range skipList {
		if name == skip {
			return true
		}
	}
	
	return false
}

func (fo *FlowObfuscator) flattenControlFlow(code string) string {
	flatCode := `
(function(){
	var _0xcf = {
		state: 0,
		switch: function(_0xv) {
			switch(this.state) {
				case 0:
					%s
					this.state = 1;
					break;
				case 1:
					%s
					this.state = 2;
					break;
				case 2:
					%s
					this.state = 0;
					break;
				default:
					this.state = 0;
			}
		}
	};
	_0xcf.switch(0);
})();
`
	stmts := fo.splitStatements(code)
	
	var stmt1, stmt2, stmt3 string
	if len(stmts) >= 1 {
		stmt1 = fo.obfuscateSingleStatement(stmts[0])
	}
	if len(stmts) >= 2 {
		stmt2 = fo.obfuscateSingleStatement(stmts[1])
	}
	if len(stmts) >= 3 {
		stmt3 = fo.obfuscateSingleStatement(stmts[2])
	}

	return fmt.Sprintf(flatCode, stmt1, stmt2, stmt3)
}

func (fo *FlowObfuscator) splitStatements(code string) []string {
	var stmts []string
	depth := 0
	current := strings.Builder{}
	
	for i, ch := range code {
		if ch == '{' {
			depth++
			current.WriteRune(ch)
		} else if ch == '}' {
			depth--
			current.WriteRune(ch)
		} else if ch == ';' && depth == 0 {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" && stmt != "{}" {
				stmts = append(stmts, stmt)
			}
			current.Reset()
		} else {
			if ch != '\n' && ch != '\t' && ch != '\r' {
				current.WriteRune(ch)
			}
		}
		
		if i == len(code)-1 {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" && stmt != "{}" {
				stmts = append(stmts, stmt)
			}
		}
	}

	if len(stmts) == 0 {
		stmts = append(stmts, code)
	}

	return stmts
}

func (fo *FlowObfuscator) obfuscateSingleStatement(stmt string) string {
	stmt = fo.obfuscateVariables(stmt)
	stmt = fo.obfuscateStrings(stmt)
	stmt = fo.addJunkCode(stmt)
	return stmt
}

func (fo *FlowObfuscator) obfuscateStrings(code string) string {
	stringRegex := regexp.MustCompile(`"([^"\\]*(\\.[^"\\]*)*)"`)
	
	code = stringRegex.ReplaceAllStringFunc(code, func(match string) string {
		parts := stringRegex.FindStringSubmatch(match)
		if len(parts) >= 2 {
			str := parts[1]
			var hexStr string
			for _, c := range str {
				hexStr += fmt.Sprintf("\\x%02x", c)
			}
			return "\"" + hexStr + "\""
		}
		return match
	})

	stringRegex2 := regexp.MustCompile(`'([^'\\]*(\\.[^'\\]*)*)'`)
	code = stringRegex2.ReplaceAllStringFunc(code, func(match string) string {
		parts := stringRegex2.FindStringSubmatch(match)
		if len(parts) >= 2 {
			str := parts[1]
			var hexStr string
			for _, c := range str {
				hexStr += fmt.Sprintf("\\x%02x", c)
			}
			return "'" + hexStr + "'"
		}
		return match
	})

	return code
}

func (fo *FlowObfuscator) addJunkCode(code string) string {
	junkCode := []string{
		";var _0xj=Math.random()*100;",
		";var _0xk=new Date().getTime();",
		";void(0);",
	}

	junk := junkCode[fo.seed%int64(len(junkCode))]
	return code + junk
}

func (fo *FlowObfuscator) ObfuscateControlFlow(code string) (string, error) {
	if code == "" {
		return "", fmt.Errorf("code is empty")
	}

	code = fo.obfuscateVariables(code)
	code = fo.obfuscateFunctions(code)
	code = fo.obfuscateStrings(code)
	code = fo.addControlFlowObfuscation(code)
	code = fo.addDeadCode(code)

	return code, nil
}

func (fo *FlowObfuscator) addControlFlowObfuscation(code string) string {
	stateVar := fo.generateObfuscatedName("_0xs")
	indexVar := fo.generateObfuscatedName("_0xi")
	
	obfuscated := fmt.Sprintf(`
(function(){
	var %s = 0;
	var %s = 0;
	%s++;
	if(%s > 1) { %s = 0; }
}, %s);
`, stateVar, indexVar, stateVar, stateVar, stateVar, stateVar)

	return obfuscated + code
}

func (fo *FlowObfuscator) addDeadCode(code string) string {
	deadCode := `
(function(){
	var _0xd = new Date();
	var _0xn = _0xd.getTime();
	if(_0xn > 0) {
		var _0xf = function() { return _0xn; };
	}
})();
`
	return deadCode + code
}

func (fo *FlowObfuscator) ObfuscateExpressions(code string) string {
	mathRegex := regexp.MustCompile(`Math\.(random|floor|ceil|round|abs|sqrt|pow|sin|cos|tan)`)
	code = mathRegex.ReplaceAllStringFunc(code, func(match string) string {
		parts := mathRegex.FindStringSubmatch(match)
		if len(parts) >= 2 {
			method := parts[1]
			obfuscated := fmt.Sprintf("_0xm['%s']", method)
			return obfuscated
		}
		return match
	})

	numberRegex := regexp.MustCompile(`\b(\d+)\b`)
	code = numberRegex.ReplaceAllStringFunc(code, func(match string) string {
		parts := numberRegex.FindStringSubmatch(match)
		if len(parts) >= 2 {
			num := parts[1]
			if len(num) <= 3 {
				return match
			}
			obfuscated := fmt.Sprintf("0x%s", fmt.Sprintf("%x", num))
			return obfuscated
		}
		return match
	})

	return code
}

func (fo *FlowObfuscator) ObfuscateProperties(code string) string {
	propRegex := regexp.MustCompile(`\.([a-zA-Z_$][a-zA-Z0-9_$]*)`)
	
	propMap := make(map[string]string)
	
	code = propRegex.ReplaceAllStringFunc(code, func(match string) string {
		parts := propRegex.FindStringSubmatch(match)
		if len(parts) >= 2 {
			prop := parts[1]
			
			if fo.shouldSkipProperty(prop) {
				return match
			}

			if _, exists := propMap[prop]; !exists {
				propMap[prop] = fo.generateObfuscatedName("_0xp")
			}

			return "." + propMap[prop]
		}
		return match
	})

	return code
}

func (fo *FlowObfuscator) shouldSkipProperty(name string) bool {
	skipList := []string{
		"length", "prototype", "constructor", "toString", "valueOf",
		"hasOwnProperty", "isPrototypeOf", "propertyIsEnumerable",
		"toLocaleString", "apply", "call", "bind",
	}
	
	for _, skip := range skipList {
		if name == skip {
			return true
		}
	}
	
	return false
}

func (fo *FlowObfuscator) AddSelfDefendingCode(code string) string {
	selfDefending := `
(function(){
	var _0xcheck = function(_0xc, _0xh) {
		var _0xr = 0;
		for(var _0xi = 0; _0xi < _0xc.length; _0xi++) {
			_0xr = (_0xr + _0xc.charCodeAt(_0xi)) % _0xh;
		}
		return _0xr;
	};
	var _0xverify = function(_0xc) {
		return _0xcheck(_0xc, 256) === 0;
	};
	if(!_0xverify(document.currentScript ? document.currentScript.textContent : '')) {
		throw new Error('Code integrity check failed');
	}
})();
`
	return code + selfDefending
}

func (fo *FlowObfuscator) AddDebuggerDetection(code string) string {
	debuggerDetection := `
(function(){
	var _0xdd = function() {
		var _0xdt = new Date();
		var _0xst = _0xdt.getTime();
		debugger;
		var _0xet = new Date().getTime();
		if(_0xet - _0xst > 100) {
			document.documentElement.style.display = 'none';
			document.body.innerHTML = '<h1>Access Denied</h1>';
			throw new Error('Debugger detected');
		}
	};
	setInterval(_0xdd, 5000);
})();
`
	return debuggerDetection + code
}

func (fo *FlowObfuscator) CreateDispatchTable(code string) string {
	stateVar := fo.generateObfuscatedName("_0xst")
	dispatchVar := fo.generateObfuscatedName("_0xdt")
	
	dispatchTable := fmt.Sprintf(`
(function(){
	var %s = 0;
	var %s = [
		function() { %s = 1; },
		function() { %s = 2; },
		function() { %s = 0; }
	];
	function dispatch() {
		%s[Math.floor(Math.random() * %s.length)]();
	}
})();
`, stateVar, dispatchVar, stateVar, stateVar, stateVar, dispatchVar, dispatchVar)

	return dispatchTable + code
}
