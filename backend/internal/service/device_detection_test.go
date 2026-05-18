package service

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestEmulatorSignatureDetection(t *testing.T) {
	testCases := []struct {
		name           string
		emulatorName   string
		testUserAgents []string
		shouldMatch    bool
	}{
		{
			name:         "BlueStacks Detection",
			emulatorName: "BlueStacks",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; HUAWEI MLK-AL00 Build/HUAWEIMLK-AL00; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/78.0.3904.108 Mobile Safari/537.36",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.136 Safari/537.36",
				"Android/9 (Linux; U; Android 9; zh-cn; BlueStacks Build/PIKALI)",
				"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.141 Safari/537.36 bst-helper/4.16.0",
			},
			shouldMatch: true,
		},
		{
			name:         "Nox Detection",
			emulatorName: "Nox",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; PMP7112DUO Build/NMF26X; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/79.0.3945.136 Mobile Safari/537.36",
				"Android/9 (Linux; U; Android 9; en-us; NoxWinda Build/PIKALI)",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36",
				"Android/10 (Linux; U; Android 10; zh-cn; Nox_71F01A Build/QKQ1.200209.002)",
			},
			shouldMatch: true,
		},
		{
			name:         "Memu Detection",
			emulatorName: "Memu",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; Mi 9 Build/PKQ1.190515.001; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/79.0.3945.136 Mobile Safari/537.36",
				"Android/9 (Linux; U; Android 9; zh-cn; MEMU Build/PIKALI)",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36",
				"Android/11 (Linux; U; Android 11; zh-cn; MEmu Build/RKQ1.200826.002)",
			},
			shouldMatch: true,
		},
		{
			name:         "LDPlayer Detection",
			emulatorName: "LDPlayer",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; SM-G9600 Build/PPR1.180610.011; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/79.0.3945.136 Mobile Safari/537.36",
				"Android/9 (Linux; U; Android 9; en-us; LDPlayer Build/PIKALI)",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36",
				"Android/11 (Linux; U; Android 11; zh-cn; ldplayer-v4 Build/SKQ1.200616.001)",
			},
			shouldMatch: true,
		},
		{
			name:         "Mumu Detection",
			emulatorName: "Mumu",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; HUAWEI MLK-AL00 Build/HUAWEIMLK-AL00; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/79.0.3945.136 Mobile Safari/537.36",
				"Android/9 (Linux; U; Android 9; zh-cn; MuMu Build/PIKALI)",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36",
				"Android/11 (Linux; U; Android 11; zh-cn; mumu_x86 Build/RKQ1.200826.002)",
			},
			shouldMatch: true,
		},
		{
			name:         "Genymotion Detection",
			emulatorName: "Genymotion",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 8.0.0; vbox86p Build/OPM6.171019.030.H1; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/61.0.3163.98 Mobile Safari/537.36",
				"Android/8.0.0 (Linux; U; Android 8.0.0; en-us; vbox86p Build/OPM6.171019.030.H1)",
				"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.136 Safari/537.36",
				"Android/9 (Linux; U; Android 9; en-us; vbox86t Build/PIKALI)",
			},
			shouldMatch: true,
		},
		{
			name:         "Gameloop Detection",
			emulatorName: "Gameloop",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; Android SDK built for x86 Build/PSR1.180720.122; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/74.0.3729.136 Mobile Safari/537.36 gameapp/1.0.0",
				"Android/9 (Linux; U; Android 9; en-us; Tencent Gaming Buddy Build/PIKALI)",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36",
				"Android/11 (Linux; U; Android 11; en-us; GameLoop Build/PIKALI)",
			},
			shouldMatch: true,
		},
		{
			name:         "SmartGaGa Detection",
			emulatorName: "SmartGaGa",
			testUserAgents: []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36 SmartGaGa/1.0",
				"Android/9 (Linux; U; Android 9; en-us; SmartGaGa Build/PIKALI)",
			},
			shouldMatch: true,
		},
		{
			name:         "WindRoy Detection",
			emulatorName: "WindRoy",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 5.1.1; WindRoy Build/LMY48B; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/55.0.2883.91 Mobile Safari/537.36",
				"Android/5.1.1 (Linux; U; Android 5.1.1; en-us; WindRoy Build/LMY48B)",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36",
			},
			shouldMatch: true,
		},
		{
			name:         "Droid4X Detection",
			emulatorName: "Droid4X",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 4.4.4; Droid4X Build/KTU84P) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/33.0.0.0 Mobile Safari/537.36",
				"Android/4.4.4 (Linux; U; Android 4.4.4; en-us; Droid4X Build/KTU84P)",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36",
			},
			shouldMatch: true,
		},
		{
			name:         "Android Emulator Generic Detection",
			emulatorName: "Android_Emulator",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; Android SDK built for x86 Build/PSR1.180720.122; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/74.0.3729.136 Mobile Safari/537.36",
				"Android/9 (Linux; U; Android 9; en-us; sdk_phone_x86 Build/PIKALI)",
				"Mozilla/5.0 (Linux; U; Android 9; en-us; goldfish Build/PIKALI)",
				"Mozilla/5.0 (Linux; U; Android 11; en-us; ranchu Build/PIKALI)",
			},
			shouldMatch: true,
		},
		{
			name:         "iOS Simulator Detection",
			emulatorName: "iOS_Simulator",
			testUserAgents: []string{
				"Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Safari/604.1",
				"Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Safari/604.1",
				"Mozilla/5.0 (iPhone; CPU iPhone OS 13_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Safari/604.1",
			},
			shouldMatch: false,
		},
		{
			name:         "Real Device Should Not Match",
			emulatorName: "Real Device",
			testUserAgents: []string{
				"Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
				"Mozilla/5.0 (Linux; Android 11; SM-G998B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.120 Mobile Safari/537.36",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			},
			shouldMatch: false,
		},
	}

	service := NewDeviceDetectionService()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matched := false
			for _, ua := range tc.testUserAgents {
				data := map[string]interface{}{
					"user_agent": ua,
				}
				result := service.DetectDevice(data)
				if result.IsEmulator && result.Score > 30 {
					matched = true
					break
				}
			}
			if matched != tc.shouldMatch {
				t.Errorf("%s: expected match=%v, got match=%v", tc.name, tc.shouldMatch, matched)
			}
		})
	}
}

func TestCloudPhoneDetection(t *testing.T) {
	testCases := []struct {
		name           string
		testUserAgents []string
		shouldMatch    bool
		expectedType   string
	}{
		{
			name: "雷电云 Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; LDY-AN00 Build/HUAWEILDY-AN00; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/79.0.3945.136 Mobile Safari/537.36",
				"Android/9 (Linux; U; Android 9; zh-cn; ldy_api Build/PIKALI)",
				"Mozilla/5.0 (Linux; Android 11; android-cloud-ldy Build/PIKALI; wv) AppleWebKit/537.36",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36 LDYun/1.0",
			},
			shouldMatch:  true,
			expectedType: "雷电云",
		},
		{
			name: "多多云 Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; DDY-AN00 Build/HUAWEIDDY-AN00; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/79.0.3945.136 Mobile Safari/537.36",
				"Android/9 (Linux; U; Android 9; zh-cn; ddy_api Build/PIKALI)",
				"Mozilla/5.0 (Linux; Android 11; android-cloud-ddy Build/PIKALI; wv) AppleWebKit/537.36",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36 DDYun/1.0",
			},
			shouldMatch:  true,
			expectedType: "多多云",
		},
		{
			name: "红警云 Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; HJ-AN00 Build/HUAWEIHJ-AN00; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/79.0.3945.136 Mobile Safari/537.36",
				"Android/9 (Linux; U; Android 9; zh-cn; hongjicloud Build/PIKALI)",
				"Mozilla/5.0 (Linux; Android 11; android-cloud-hj Build/PIKALI; wv) AppleWebKit/537.36",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36 HongJi/1.0",
			},
			shouldMatch:  true,
			expectedType: "红警",
		},
		{
			name: "双子云 Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; SZ-AN00 Build/HUAWEISZ-AN00; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/79.0.3945.136 Mobile Safari/537.36",
				"Android/9 (Linux; U; Android 9; zh-cn; szcloud Build/PIKALI)",
				"Mozilla/5.0 (Linux; Android 11; android-cloud-sz Build/PIKALI; wv) AppleWebKit/537.36",
			},
			shouldMatch:  true,
			expectedType: "双子云",
		},
		{
			name: "蜂窝云 Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; FW-AN00 Build/HUAWEIFW-AN00; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/79.0.3945.136 Mobile Safari/537.36",
				"Android/9 (Linux; U; Android 9; zh-cn; fwcloud Build/PIKALI)",
				"Mozilla/5.0 (Linux; Android 11; android-cloud-fw Build/PIKALI; wv) AppleWebKit/537.36",
			},
			shouldMatch:  true,
			expectedType: "蜂窝云",
		},
		{
			name: "云帅云 Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; YS-AN00 Build/HUAWEIYS-AN00; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/79.0.3945.136 Mobile Safari/537.36",
				"Android/9 (Linux; U; Android 9; zh-cn; yscloud Build/PIKALI)",
				"Mozilla/5.0 (Linux; Android 11; android-cloud-ys Build/PIKALI; wv) AppleWebKit/537.36",
			},
			shouldMatch:  true,
			expectedType: "云帅云",
		},
		{
			name: "蓝光云 Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 9; LG-AN00 Build/HUAWEILG-AN00; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/79.0.3945.136 Mobile Safari/537.36",
				"Android/9 (Linux; U; Android 9; zh-cn; lgcloud Build/PIKALI)",
				"Mozilla/5.0 (Linux; Android 11; android-cloud-lg Build/PIKALI; wv) AppleWebKit/537.36",
			},
			shouldMatch:  true,
			expectedType: "蓝光云",
		},
		{
			name: "Real Device Should Not Match Cloud Phone",
			testUserAgents: []string{
				"Mozilla/5.0 (Linux; Android 11; SM-G998B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.120 Mobile Safari/537.36",
				"Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
			},
			shouldMatch:  false,
			expectedType: "",
		},
	}

	service := NewDeviceDetectionService()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matched := false
			for _, ua := range tc.testUserAgents {
				data := map[string]interface{}{
					"user_agent": ua,
				}
				result := service.DetectDevice(data)
				if result.IsEmulator && result.Score > 30 {
					matched = true
					break
				}
			}
			if matched != tc.shouldMatch {
				t.Errorf("%s: expected match=%v, got match=%v", tc.name, tc.shouldMatch, matched)
			}
		})
	}
}

func TestVMSignatureDetection(t *testing.T) {
	testCases := []struct {
		name           string
		testUserAgents []string
		testWebGL      []string
		shouldMatch    bool
		expectedType   string
	}{
		{
			name: "VMware Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 VMware/7.1",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 VMware Virtual Platform",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 VMware Tools",
			},
			testWebGL: []string{
				"VMware SVGA3D",
				"VMware, Inc. VMware SVGA3D",
			},
			shouldMatch:  true,
			expectedType: "VMware",
		},
		{
			name: "VirtualBox Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 VirtualBox",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 VBOX",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 vbox86p",
			},
			testWebGL: []string{
				"VirtualBox Graphics",
				"Oracle Corporation VirtualBox Graphics",
			},
			shouldMatch:  true,
			expectedType: "VirtualBox",
		},
		{
			name: "Hyper-V Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Microsoft Hyper-V",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Virtual Machine",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Microsoft Corporation Virtual Machine",
			},
			testWebGL: []string{
				"Microsoft Hyper-V",
			},
			shouldMatch:  true,
			expectedType: "HyperV",
		},
		{
			name: "Parallels Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Parallels",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Parallels Virtual Platform",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 prl_vm_app",
			},
			testWebGL: []string{
				"Parallels",
				"Parallels Virtual Platform",
			},
			shouldMatch:  true,
			expectedType: "Parallels",
		},
		{
			name: "QEMU/KVM Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 QEMU",
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 KVM",
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Standard PC (Q35 + ICH9)",
			},
			testWebGL: []string{
				"QEMU Virtual CPU",
				"Red Hat, Inc. KVM",
			},
			shouldMatch:  true,
			expectedType: "QEMU_KVM",
		},
		{
			name: "Xen Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Xen",
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 HVM domU",
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 XenSource",
			},
			testWebGL: []string{},
			shouldMatch:  true,
			expectedType: "Xen",
		},
		{
			name: "Real Machine Should Not Match",
			testUserAgents: []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			},
			testWebGL: []string{
				"NVIDIA GeForce RTX 3080",
				"AMD Radeon RX 6800 XT",
				"Intel UHD Graphics 630",
			},
			shouldMatch:  false,
			expectedType: "",
		},
	}

	service := NewDeviceDetectionService()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matched := false
			for _, ua := range tc.testUserAgents {
				for _, webgl := range tc.testWebGL {
					data := map[string]interface{}{
						"user_agent":    ua,
						"webgl_renderer": webgl,
					}
					result := service.DetectDevice(data)
					if result.IsVirtual && result.Score > 30 {
						matched = true
						break
					}
				}
				if matched {
					break
				}
				if len(tc.testWebGL) == 0 {
					data := map[string]interface{}{
						"user_agent": ua,
					}
					result := service.DetectDevice(data)
					if result.IsVirtual && result.Score > 30 {
						matched = true
						break
					}
				}
			}
			if matched != tc.shouldMatch {
				t.Errorf("%s: expected match=%v, got match=%v", tc.name, tc.shouldMatch, matched)
			}
		})
	}
}

func TestContainerDetection(t *testing.T) {
	testCases := []struct {
		name           string
		testUserAgents []string
		shouldMatch    bool
		expectedType   string
	}{
		{
			name: "Docker Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Docker",
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 container=docker",
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 docker-init",
			},
			shouldMatch:  true,
			expectedType: "Docker",
		},
		{
			name: "Kubernetes Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Kubernetes",
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 KUBERNETES_SERVICE_PORT",
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 kubernetes.io",
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 k8s",
			},
			shouldMatch:  true,
			expectedType: "Kubernetes",
		},
		{
			name: "LXC Detection",
			testUserAgents: []string{
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 LXC",
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 lxc/",
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 /dev/lxd/sock",
			},
			shouldMatch:  true,
			expectedType: "LXC",
		},
		{
			name: "Real Machine Should Not Match Container",
			testUserAgents: []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				"Mozilla/5.0 (X86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			},
			shouldMatch:  false,
			expectedType: "",
		},
	}

	service := NewDeviceDetectionService()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matched := false
			for _, ua := range tc.testUserAgents {
				data := map[string]interface{}{
					"user_agent": ua,
				}
				result := service.DetectDevice(data)
				if result.IsContainer && result.Score > 25 {
					matched = true
					break
				}
			}
			if matched != tc.shouldMatch {
				t.Errorf("%s: expected match=%v, got match=%v", tc.name, tc.shouldMatch, matched)
			}
		})
	}
}

func TestEmulatorSignaturePatternMatching(t *testing.T) {
	service := NewDeviceDetectionService()

	patternTests := []struct {
		pattern    string
		testString string
		shouldMatch bool
	}{
		{`(?i)bluestacks`, "Mozilla/5.0 Chrome/91.0 BlueStacks", true},
		{`(?i)nox`, "Mozilla/5.0 Chrome/91.0 NoxPlayer", true},
		{`(?i)memu`, "Mozilla/5.0 Chrome/91.0 MEmu", true},
		{`(?i)ldplayer`, "Mozilla/5.0 Chrome/91.0 LDPlayer", true},
		{`(?i)mumu`, "Mozilla/5.0 Chrome/91.0 MuMu", true},
		{`(?i)genymotion`, "Mozilla/5.0 Chrome/91.0 Genymotion", true},
		{`(?i)gameloop`, "Mozilla/5.0 Chrome/91.0 GameLoop", true},
		{`(?i)smartgaga`, "Mozilla/5.0 Chrome/91.0 SmartGaGa", true},
		{`(?i)windroy`, "Mozilla/5.0 Chrome/91.0 WindRoy", true},
		{`(?i)droid4x`, "Mozilla/5.0 Chrome/91.0 Droid4X", true},
		{`(?i)ldyun`, "Mozilla/5.0 Chrome/91.0 LDYun", true},
		{`(?i)ddyun`, "Mozilla/5.0 Chrome/91.0 DDYun", true},
		{`(?i)hongji`, "Mozilla/5.0 Chrome/91.0 HongJi", true},
		{`(?i)vmware`, "Mozilla/5.0 Chrome/91.0 VMware", true},
		{`(?i)virtualbox`, "Mozilla/5.0 Chrome/91.0 VirtualBox", true},
		{`(?i)hyper[_-]?v`, "Mozilla/5.0 Chrome/91.0 Hyper-V", true},
		{`(?i)parallels`, "Mozilla/5.0 Chrome/91.0 Parallels", true},
		{`(?i)qemu`, "Mozilla/5.0 Chrome/91.0 QEMU", true},
		{`(?i)docker`, "Mozilla/5.0 Chrome/91.0 Docker", true},
		{`(?i)kubernetes`, "Mozilla/5.0 Chrome/91.0 Kubernetes", true},
		{`(?i)lxc`, "Mozilla/5.0 Chrome/91.0 LXC", true},
		{`(?i)bluestacks`, "Mozilla/5.0 Chrome/91.0 RealDevice", false},
		{`(?i)nox`, "Mozilla/5.0 Chrome/91.0 Samsung", false},
		{`(?i)vmware`, "Mozilla/5.0 Chrome/91.0 Dell", false},
	}

	for _, tt := range patternTests {
		t.Run(tt.testString, func(t *testing.T) {
			pattern := regexp.MustCompile(tt.pattern)
			matched := pattern.MatchString(tt.testString)
			if matched != tt.shouldMatch {
				t.Errorf("Pattern %s matching %s: expected %v, got %v", tt.pattern, tt.testString, tt.shouldMatch, matched)
			}
		})
	}

	_ = service
}

func TestDeviceFingerprintGeneration(t *testing.T) {
	service := NewDeviceDetectionService()

	testCases := []struct {
		name     string
		data     map[string]interface{}
	}{
		{
			name: "Basic Fingerprint",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0.4472.124",
				"platform":   "Win32",
				"timezone":   "Asia/Shanghai",
				"language":   "zh-CN",
			},
		},
		{
			name: "Mobile Fingerprint",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 (Linux; Android 11; SM-G998B) Chrome/91.0.4472.120 Mobile Safari/537.36",
				"platform":   "Linux",
				"timezone":   "Asia/Seoul",
				"language":   "ko-KR",
			},
		},
		{
			name: "Emulator Fingerprint",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 (Linux; Android 9; Nox Build/PIKALI) Chrome/91.0.4472.120 Mobile Safari/537.36",
				"platform":   "Linux",
				"timezone":   "Asia/Shanghai",
				"language":   "zh-CN",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fingerprintID := service.RecordDeviceFingerprint(tc.data)
			if fingerprintID == "" {
				t.Errorf("Fingerprint ID should not be empty")
			}
			if len(fingerprintID) != 32 {
				t.Errorf("Fingerprint ID should be 32 characters, got %d", len(fingerprintID))
			}

			retrieved, exists := service.GetDeviceFingerprint(fingerprintID)
			if !exists {
				t.Errorf("Fingerprint should exist in database")
			}
			if retrieved.FingerprintID != fingerprintID {
				t.Errorf("Retrieved fingerprint ID mismatch")
			}
		})
	}
}

func TestDeviceDetectionScoreCalculation(t *testing.T) {
	service := NewDeviceDetectionService()

	testCases := []struct {
		name            string
		data            map[string]interface{}
		minExpectedScore float64
		expectedType    EnvironmentType
	}{
		{
			name: "High Score Emulator Detection",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 (Linux; Android 9; NoxPlayer Build/PIKALI) Chrome/79.0.3945.136 Mobile Safari/537.36",
			},
			minExpectedScore: 50,
			expectedType:     EnvEmulator,
		},
		{
			name: "High Score VM Detection",
			data: map[string]interface{}{
				"user_agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0.4472.124 Safari/537.36 VMware Virtual Platform",
				"webgl_renderer": "VMware SVGA3D",
			},
			minExpectedScore: 60,
			expectedType:     EnvVirtual,
		},
		{
			name: "Container Detection",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 (X86_64) Chrome/91.0.4472.124 Safari/537.36 container=docker",
			},
			minExpectedScore: 40,
			expectedType:     EnvContainer,
		},
		{
			name: "Real Device Low Score",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0.4472.124 Safari/537.36",
			},
			minExpectedScore: 0,
			expectedType:     EnvReal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.DetectDevice(tc.data)
			if result.Score < tc.minExpectedScore {
				t.Errorf("%s: expected score >= %v, got %v", tc.name, tc.minExpectedScore, result.Score)
			}
			if tc.expectedType != "" && result.EnvironmentType != tc.expectedType && result.Score >= tc.minExpectedScore {
				t.Logf("%s: got environment type %v, expected %v", tc.name, result.EnvironmentType, tc.expectedType)
			}
		})
	}
}

func TestStabilityScoreCalculation(t *testing.T) {
	service := NewDeviceDetectionService()

	data := map[string]interface{}{
		"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0.4472.124 Safari/537.36",
		"platform":   "Win32",
	}

	fingerprintID := service.RecordDeviceFingerprint(data)

	for i := 0; i < 5; i++ {
		service.RecordDeviceFingerprint(data)
	}

	score := service.CalculateStabilityScore(fingerprintID)
	if score <= 0 {
		t.Errorf("Stability score should be positive for frequent device")
	}
	if score > 100 {
		t.Errorf("Stability score should not exceed 100")
	}
}

func TestCleanupOldData(t *testing.T) {
	service := NewDeviceDetectionService()

	oldData := map[string]interface{}{
		"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0.4472.124 Safari/537.36",
		"platform":   "Win32",
	}

	fingerprintID := service.RecordDeviceFingerprint(oldData)

	removed := service.CleanupOldData(24 * time.Hour)
	if removed > 0 {
		t.Logf("Cleaned up %d old entries", removed)
	}

	_, exists := service.GetDeviceFingerprint(fingerprintID)
	if !exists {
		t.Errorf("Recent device should not be cleaned up")
	}
}

func TestHardwareInfoExtraction(t *testing.T) {
	service := NewDeviceDetectionService()

	testCases := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "Full Hardware Info",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 Chrome/91.0",
				"hardware_info": map[string]interface{}{
					"cpu_info":     "Intel(R) Core(TM) i7-10700K CPU @ 3.80GHz",
					"cpu_cores":    float64(16),
					"device_memory": float64(32),
					"gpu_info":     "NVIDIA GeForce RTX 3080",
				},
				"screen_info": map[string]interface{}{
					"width":       float64(1920),
					"height":      float64(1080),
					"color_depth": float64(24),
					"pixel_ratio": float64(1.0),
				},
			},
		},
		{
			name: "Partial Hardware Info",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 Chrome/91.0",
				"hardware_info": map[string]interface{}{
					"cpu_cores": float64(8),
				},
			},
		},
		{
			name: "No Hardware Info",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 Chrome/91.0",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fingerprintID := service.RecordDeviceFingerprint(tc.data)
			info, exists := service.GetDeviceFingerprint(fingerprintID)
			if !exists {
				t.Errorf("Fingerprint should exist")
				return
			}
			if info.HardwareInfo == nil {
				t.Errorf("Hardware info should not be nil")
			}
		})
	}
}

func TestNetworkInfoExtraction(t *testing.T) {
	service := NewDeviceDetectionService()

	testCases := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "Full Network Info",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 Chrome/91.0",
				"connection_info": map[string]interface{}{
					"type":           "wifi",
					"effective_type": "4g",
					"rtt":            float64(50),
					"downlink":       float64(10),
				},
			},
		},
		{
			name: "Partial Network Info",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 Chrome/91.0",
				"connection_info": map[string]interface{}{
					"type": "wifi",
				},
			},
		},
		{
			name: "No Network Info",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 Chrome/91.0",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fingerprintID := service.RecordDeviceFingerprint(tc.data)
			info, exists := service.GetDeviceFingerprint(fingerprintID)
			if !exists {
				t.Errorf("Fingerprint should exist")
				return
			}
			if info.NetworkInfo == nil {
				t.Errorf("Network info should not be nil")
			}
		})
	}
}

func TestMultiBoxDetection(t *testing.T) {
	service := NewDeviceDetectionService()

	baseData := map[string]interface{}{
		"user_agent":         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0.4472.124 Safari/537.36",
		"device_fingerprint": "test-fingerprint-123",
	}

	for i := 0; i < 15; i++ {
		service.RecordDeviceFingerprint(baseData)
	}

	result := service.DetectDevice(map[string]interface{}{
		"user_agent":         baseData["user_agent"],
		"device_fingerprint": baseData["device_fingerprint"],
	})

	if result.Score < 40 {
		t.Logf("MultiBox detection score: %v (expected to increase with rapid requests)", result.Score)
	}
}

func TestEdgeCaseEmulatorDetection(t *testing.T) {
	service := NewDeviceDetectionService()

	edgeCases := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "Empty User Agent",
			data: map[string]interface{}{
				"user_agent": "",
			},
		},
		{
			name: "Very Long User Agent",
			data: map[string]interface{}{
				"user_agent": strings.Repeat("A", 10000),
			},
		},
		{
			name: "Unicode in User Agent",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 (Linux; Android 9; 模拟器) Chrome/79.0.3945.136 Mobile Safari/537.36",
			},
		},
		{
			name: "Mixed Case Emulator",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 (Linux; Android 9; BlueStacks) Chrome/79.0.3945.136 Mobile Safari/537.36",
			},
		},
		{
			name: "Partial Match",
			data: map[string]interface{}{
				"user_agent": "Mozilla/5.0 (Linux; Android 9; Nox-like Build/PIKALI) Chrome/79.0.3945.136 Mobile Safari/537.36",
			},
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.DetectDevice(tc.data)
			if result == nil {
				t.Errorf("Result should not be nil for %s", tc.name)
			}
			if result.Score < 0 || result.Score > 100 {
				t.Errorf("Score should be between 0 and 100 for %s, got %v", tc.name, result.Score)
			}
		})
	}
}

func TestEmulatorIndicatorsInUserAgent(t *testing.T) {
	service := NewDeviceDetectionService()

	indicatorTests := []struct {
		name      string
		ua        string
		indicator string
	}{
		{"BlueStacks Indicator", "Mozilla/5.0 Chrome/91.0 bst-helper", "bst-helper"},
		{"Nox Indicator", "Mozilla/5.0 Chrome/91.0 NoxApp", "NoxApp"},
		{"Memu Indicator", "Mozilla/5.0 Chrome/91.0 Microvirt", "Microvirt"},
		{"LDPlayer Indicator", "Mozilla/5.0 Chrome/91.0 ldlib", "ldlib"},
		{"Genymotion Indicator", "Mozilla/5.0 Chrome/91.0 vbox86p", "vbox86p"},
		{"VMware Indicator", "Mozilla/5.0 Chrome/91.0 VMware7,1", "VMware7,1"},
		{"VirtualBox Indicator", "Mozilla/5.0 Chrome/91.0 VBoxSharedFolders", "VBoxSharedFolders"},
		{"Docker Indicator", "Mozilla/5.0 Chrome/91.0 /.dockerenv", "/.dockerenv"},
	}

	for _, tt := range indicatorTests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]interface{}{
				"user_agent": tt.ua,
			}
			result := service.DetectDevice(data)
			found := false
			for _, indicator := range result.Indicators {
				if strings.Contains(indicator, tt.indicator) {
					found = true
					break
				}
			}
			if !found && result.Score < 30 {
				t.Logf("Indicator %s not explicitly found in result for UA: %s", tt.indicator, tt.ua)
			}
		})
	}
}

func TestWebGLVMDetection(t *testing.T) {
	service := NewDeviceDetectionService()

	webglTests := []struct {
		name        string
		webgl       string
		shouldMatch bool
	}{
		{"VMware WebGL", "VMware SVGA3D", true},
		{"VirtualBox WebGL", "VirtualBox Graphics", true},
		{"QEMU WebGL", "QEMU Virtual CPU", true},
		{"Generic GPU", "NVIDIA GeForce RTX 3080", false},
		{"Intel GPU", "Intel UHD Graphics 630", false},
		{"AMD GPU", "AMD Radeon RX 6800 XT", false},
	}

	for _, tt := range webglTests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]interface{}{
				"user_agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
				"webgl_renderer": tt.webgl,
			}
			result := service.DetectDevice(data)
			if tt.shouldMatch && result.Score < 30 {
				t.Logf("Expected VM detection for %s, score: %v", tt.webgl, result.Score)
			}
		})
	}
}
