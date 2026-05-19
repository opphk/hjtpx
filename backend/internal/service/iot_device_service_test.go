package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIoTDeviceService(t *testing.T) {
	svc := NewIoTDeviceService()
	assert.NotNil(t, svc)
}

func TestRegisterDevice(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType:   "smart_thermostat",
		DeviceName:   "Living Room Thermostat",
		Manufacturer: "SmartHome Inc",
		Model:        "TH-100",
		Firmware:     "1.0.0",
		MACAddress:   "AA:BB:CC:DD:EE:FF",
		IPAddress:    "192.168.1.100",
		Location:     "Living Room",
		OwnerID:      "user123",
		Capabilities: []string{"temperature", "humidity", "schedule"},
		Metadata:     map[string]string{"os": "embedded_linux"},
	}

	err := svc.RegisterDevice(ctx, device)

	require.NoError(t, err)
	assert.NotEmpty(t, device.DeviceID)
	assert.Equal(t, "active", device.Status)
	assert.Greater(t, device.TrustScore, float64(0))
}

func TestRegisterDevice_Nil(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	err := svc.RegisterDevice(ctx, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestGetDevice(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType:   "camera",
		DeviceName:   "Front Door Camera",
		Manufacturer: "SecureCam",
		MACAddress:   "11:22:33:44:55:66",
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	retrieved, err := svc.GetDevice(ctx, device.DeviceID)

	require.NoError(t, err)
	assert.Equal(t, device.DeviceID, retrieved.DeviceID)
	assert.Equal(t, device.DeviceName, retrieved.DeviceName)
}

func TestGetDevice_NotFound(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device, err := svc.GetDevice(ctx, "non-existent")

	assert.Error(t, err)
	assert.Nil(t, device)
	assert.Equal(t, ErrDeviceNotFound, err)
}

func TestVerifyDevice(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType: "sensor",
		DeviceName: "Temperature Sensor",
		MACAddress: "AA:BB:CC:DD:EE:01",
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	fp, err := svc.GetDeviceFingerprint(ctx, device.DeviceID)
	require.NoError(t, err)

	request := &DeviceVerifyRequest{
		DeviceID:    device.DeviceID,
		Fingerprint: fp,
		Timestamp:   time.Now(),
	}

	response, err := svc.VerifyDevice(ctx, request)

	require.NoError(t, err)
	assert.True(t, response.Valid)
	assert.Equal(t, device.DeviceID, response.DeviceID)
}

func TestVerifyDevice_Expired(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType:     "sensor",
		DeviceName:     "Expired Sensor",
		MACAddress:     "AA:BB:CC:DD:EE:02",
		ExpirationDate: time.Now().Add(-24 * time.Hour),
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	request := &DeviceVerifyRequest{
		DeviceID:  device.DeviceID,
		Timestamp: time.Now(),
	}

	response, err := svc.VerifyDevice(ctx, request)

	require.NoError(t, err)
	assert.False(t, response.Valid)
	assert.Equal(t, "critical", response.RiskLevel)
}

func TestVerifyDevice_TimestampExpired(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType: "actuator",
		DeviceName: "Smart Actuator",
		MACAddress: "AA:BB:CC:DD:EE:03",
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	request := &DeviceVerifyRequest{
		DeviceID:  device.DeviceID,
		Timestamp: time.Now().Add(-10 * time.Minute),
	}

	response, err := svc.VerifyDevice(ctx, request)

	require.NoError(t, err)
	assert.False(t, response.Valid)
	assert.Equal(t, "medium", response.RiskLevel)
}

func TestGetDeviceFingerprint(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType:   "gateway",
		DeviceName:   "Home Gateway",
		Manufacturer: "IoT Corp",
		Model:        "GW-200",
		Firmware:     "2.0.0",
		MACAddress:   "AA:BB:CC:DD:EE:04",
		Capabilities: []string{"wifi", "bluetooth", "zigbee"},
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	fp, err := svc.GetDeviceFingerprint(ctx, device.DeviceID)

	require.NoError(t, err)
	assert.NotEmpty(t, fp.HardwareHash)
	assert.NotEmpty(t, fp.SoftwareHash)
	assert.NotEmpty(t, fp.NetworkPattern)
	assert.Greater(t, fp.BehavioralScore, float64(0))
}

func TestGetDeviceFingerprint_NotFound(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	fp, err := svc.GetDeviceFingerprint(ctx, "non-existent")

	assert.Error(t, err)
	assert.Nil(t, fp)
}

func TestAuthenticateSmartHome_Thermostat(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType:   "thermostat",
		DeviceName:   "Smart Thermostat",
		MACAddress:   "AA:BB:CC:DD:EE:05",
		Location:     "Bedroom",
		Capabilities: []string{"temperature", "humidity", "schedule"},
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	request := &SmartHomeAuthRequest{
		DeviceID:   device.DeviceID,
		DeviceType: "thermostat",
		Location:   "Bedroom",
		Sensors: map[string]float64{
			"temperature": 22.5,
			"humidity":    45.0,
		},
	}

	response, err := svc.AuthenticateSmartHome(ctx, request)

	require.NoError(t, err)
	assert.True(t, response.Authenticated)
	assert.NotEmpty(t, response.AccessToken)
	assert.Contains(t, response.Permissions, "read_temperature")
	assert.Contains(t, response.Permissions, "write_temperature")
}

func TestAuthenticateSmartHome_InvalidSensors(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType: "camera",
		MACAddress: "AA:BB:CC:DD:EE:06",
		Location:   "Hallway",
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	request := &SmartHomeAuthRequest{
		DeviceID:   device.DeviceID,
		DeviceType: "camera",
		Location:   "Hallway",
		Sensors: map[string]float64{
			"temperature": 150.0,
		},
	}

	response, err := svc.AuthenticateSmartHome(ctx, request)

	require.NoError(t, err)
	assert.False(t, response.Authenticated)
}

func TestAuthenticateSmartHome_NotRegistered(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	request := &SmartHomeAuthRequest{
		DeviceID:   "non-existent",
		DeviceType: "lock",
		Location:   "Front Door",
		Sensors:    map[string]float64{"motion": 1},
	}

	response, err := svc.AuthenticateSmartHome(ctx, request)

	require.NoError(t, err)
	assert.False(t, response.Authenticated)
}

func TestAuthenticateVehicle_ValidVIN(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType:   "connected_car",
		DeviceName:   "Tesla Model 3",
		Manufacturer: "Tesla",
		Model:        "Model 3",
		MACAddress:   "AA:BB:CC:DD:EE:07",
		Capabilities: []string{"autopilot", "remote_start", "location_tracking"},
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	validVIN := "1HGBH41JXMN109186"

	request := &VehicleAuthRequest{
		VehicleID: device.DeviceID,
		VIN:       validVIN,
		Odometer:  15000,
		Location:  "San Francisco, CA",
	}

	response, err := svc.AuthenticateVehicle(ctx, request)

	require.NoError(t, err)
	assert.True(t, response.Authorized)
	assert.NotEmpty(t, response.Token)
	assert.Contains(t, response.Features, "autopilot")
}

func TestAuthenticateVehicle_InvalidVIN(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType: "ev",
		DeviceName: "Electric Vehicle",
		MACAddress: "AA:BB:CC:DD:EE:08",
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	request := &VehicleAuthRequest{
		VehicleID: device.DeviceID,
		VIN:       "INVALID",
		Odometer:  5000,
	}

	response, err := svc.AuthenticateVehicle(ctx, request)

	require.NoError(t, err)
	assert.False(t, response.Authorized)
}

func TestAuthenticateVehicle_NotRegistered(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	request := &VehicleAuthRequest{
		VehicleID: "non-existent",
		VIN:       "1HGBH41JXMN109186",
		Odometer:  10000,
	}

	response, err := svc.AuthenticateVehicle(ctx, request)

	require.NoError(t, err)
	assert.False(t, response.Authorized)
}

func TestAuthenticateIIoT_Valid(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType:   "plc",
		DeviceName:   "Industrial PLC",
		Manufacturer: "Siemens",
		Model:        "S7-1500",
		MACAddress:   "AA:BB:CC:DD:EE:09",
		Capabilities: []string{"modbus", "ethernet", "digital_io"},
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	request := &IIoTAuthRequest{
		DeviceID:       device.DeviceID,
		DeviceType:     "plc",
		PlantID:        "plant-001",
		ProductionLine: "assembly-line-1",
		Certificates:   []string{"iso9001", "iec62443"},
		Measurements: map[string]float64{
			"temperature": 45.0,
			"pressure":     2.5,
		},
		Timestamp: time.Now(),
	}

	response, err := svc.AuthenticateIIoT(ctx, request)

	require.NoError(t, err)
	assert.True(t, response.Authorized)
	assert.NotEmpty(t, response.SessionToken)
	assert.Equal(t, "compliant", response.Compliance)
	assert.Equal(t, "full_control", response.AccessLevel)
}

func TestAuthenticateIIoT_MissingPlantInfo(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType: "sensor",
		MACAddress: "AA:BB:CC:DD:EE:10",
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	request := &IIoTAuthRequest{
		DeviceID:       device.DeviceID,
		PlantID:        "",
		ProductionLine: "assembly-1",
		Timestamp:      time.Now(),
	}

	response, err := svc.AuthenticateIIoT(ctx, request)

	require.NoError(t, err)
	assert.False(t, response.Authorized)
}

func TestAuthenticateIIoT_ExpiredTimestamp(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType: "gateway",
		MACAddress: "AA:BB:CC:DD:EE:11",
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	request := &IIoTAuthRequest{
		DeviceID:       device.DeviceID,
		PlantID:        "plant-001",
		ProductionLine: "line-1",
		Timestamp:      time.Now().Add(-10 * time.Minute),
	}

	response, err := svc.AuthenticateIIoT(ctx, request)

	require.NoError(t, err)
	assert.False(t, response.Authorized)
}

func TestGetDeviceHistory(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType: "test_device",
		MACAddress: "AA:BB:CC:DD:EE:12",
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	_, err = svc.AuthenticateSmartHome(ctx, &SmartHomeAuthRequest{
		DeviceID: device.DeviceID,
		Sensors:  map[string]float64{"temperature": 25},
	})
	require.NoError(t, err)

	_, err = svc.VerifyDevice(ctx, &DeviceVerifyRequest{
		DeviceID:  device.DeviceID,
		Timestamp: time.Now(),
	})
	require.NoError(t, err)

	history, err := svc.GetDeviceHistory(ctx, device.DeviceID, 10, 0)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(history), 2)
}

func TestGetDeviceHistory_Empty(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	history, err := svc.GetDeviceHistory(ctx, "non-existent", 10, 0)

	require.NoError(t, err)
	assert.Len(t, history, 0)
}

func TestGetDeviceHistory_Pagination(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType: "multi_event_device",
		MACAddress: "AA:BB:CC:DD:EE:13",
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, _ = svc.VerifyDevice(ctx, &DeviceVerifyRequest{
			DeviceID:  device.DeviceID,
			Timestamp: time.Now(),
		})
	}

	page1, err := svc.GetDeviceHistory(ctx, device.DeviceID, 2, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	page2, err := svc.GetDeviceHistory(ctx, device.DeviceID, 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	page3, err := svc.GetDeviceHistory(ctx, device.DeviceID, 2, 4)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(page3), 1)
}

func TestSmartHomeAuth_AllDeviceTypes(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	deviceTypes := []string{"thermostat", "camera", "lock", "light", "sensor"}

	for _, deviceType := range deviceTypes {
		device := &IoTDevice{
			DeviceType: deviceType,
			MACAddress: "AA:BB:CC:DD:EE:" + deviceType[:2],
			Location:   "Test Location",
		}

		err := svc.RegisterDevice(ctx, device)
		require.NoError(t, err)

		request := &SmartHomeAuthRequest{
			DeviceID:   device.DeviceID,
			DeviceType: deviceType,
			Location:   "Test Location",
			Sensors: map[string]float64{
				"temperature": 22.0,
				"humidity":    50.0,
			},
		}

		response, err := svc.AuthenticateSmartHome(ctx, request)
		require.NoError(t, err)
		assert.True(t, response.Authenticated, "Device type %s should authenticate", deviceType)
		assert.NotEmpty(t, response.Permissions)
	}
}

func TestVehicleAuth_AllSecurityLevels(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	vins := []string{
		"1HGBH41JXMN109186",
		"1HGCM82633A004352",
		"5YJSA1DG9DFP14705",
	}

	for i, vin := range vins {
		device := &IoTDevice{
			DeviceType: "vehicle_" + string(rune('0'+i)),
			MACAddress: "AA:BB:CC:DD:EE:" + string(rune('F'-i)),
			TrustScore: float64(50 + i*20),
		}

		err := svc.RegisterDevice(ctx, device)
		require.NoError(t, err)

		request := &VehicleAuthRequest{
			VehicleID: device.DeviceID,
			VIN:       vin,
			Odometer:  10000 * (i + 1),
		}

		response, err := svc.AuthenticateVehicle(ctx, request)
		require.NoError(t, err)
		assert.True(t, response.Authorized)
		assert.NotEmpty(t, response.SecurityLevel)
		assert.NotEmpty(t, response.Features)
	}
}

func TestIIoTAuth_AllDeviceTypes(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	deviceTypes := []string{"plc", "sensor", "actuator", "gateway"}

	for _, deviceType := range deviceTypes {
		device := &IoTDevice{
			DeviceType: deviceType,
			MACAddress: "AA:BB:CC:DD:EE:" + deviceType[:2],
		}

		err := svc.RegisterDevice(ctx, device)
		require.NoError(t, err)

		request := &IIoTAuthRequest{
			DeviceID:       device.DeviceID,
			DeviceType:     deviceType,
			PlantID:        "test-plant",
			ProductionLine: "test-line",
			Certificates:   []string{"iso9001"},
			Measurements: map[string]float64{
				"temperature": 40.0,
				"pressure":     2.0,
			},
			Timestamp: time.Now(),
		}

		response, err := svc.AuthenticateIIoT(ctx, request)
		require.NoError(t, err)
		assert.True(t, response.Authorized, "Device type %s should authenticate", deviceType)
		assert.NotEmpty(t, response.AccessLevel)
	}
}

func TestFingerprintGeneration(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType:   "test_device",
		DeviceName:   "Fingerprint Test",
		Manufacturer: "TestCo",
		Model:        "FP-100",
		Firmware:     "1.0.0",
		MACAddress:   "AA:BB:CC:DD:EE:FF",
		Capabilities: []string{"feature1", "feature2"},
		Metadata:     map[string]string{"os": "linux"},
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	fp, err := svc.GetDeviceFingerprint(ctx, device.DeviceID)
	require.NoError(t, err)

	assert.NotEmpty(t, fp.HardwareHash)
	assert.NotEmpty(t, fp.SoftwareHash)
	assert.NotEmpty(t, fp.NetworkPattern)
	assert.Equal(t, device.DeviceID, fp.DeviceID)

	fp2, err := svc.GetDeviceFingerprint(ctx, device.DeviceID)
	require.NoError(t, err)
	assert.Equal(t, fp.HardwareHash, fp2.HardwareHash)
}

func TestDeviceExpiration(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	device := &IoTDevice{
		DeviceType:     "temporary_device",
		MACAddress:     "AA:BB:CC:DD:EE:14",
		ExpirationDate: time.Now().Add(-1 * time.Hour),
	}

	err := svc.RegisterDevice(ctx, device)
	require.NoError(t, err)

	response, err := svc.VerifyDevice(ctx, &DeviceVerifyRequest{
		DeviceID:  device.DeviceID,
		Timestamp: time.Now(),
	})

	require.NoError(t, err)
	assert.False(t, response.Valid)
	assert.Equal(t, "critical", response.RiskLevel)
	assert.Contains(t, response.Message, "expired")
}

func TestMultipleDeviceRegistrations(t *testing.T) {
	svc := NewIoTDeviceService()
	ctx := context.Background()

	deviceIDs := []string{}

	for i := 0; i < 10; i++ {
		device := &IoTDevice{
			DeviceType: "batch_device",
			MACAddress: "AA:BB:CC:DD:EE:" + string(rune('0'+i)),
		}

		err := svc.RegisterDevice(ctx, device)
		require.NoError(t, err)
		deviceIDs = append(deviceIDs, device.DeviceID)
	}

	assert.Len(t, deviceIDs, 10)

	for _, id := range deviceIDs {
		device, err := svc.GetDevice(ctx, id)
		require.NoError(t, err)
		assert.NotNil(t, device)
	}
}
