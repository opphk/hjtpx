package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type IoTDeviceService interface {
	RegisterDevice(ctx context.Context, device *IoTDevice) error
	GetDevice(ctx context.Context, deviceID string) (*IoTDevice, error)
	VerifyDevice(ctx context.Context, request *DeviceVerifyRequest) (*DeviceVerifyResponse, error)
	GetDeviceFingerprint(ctx context.Context, deviceID string) (*DeviceFingerprint, error)
	AuthenticateSmartHome(ctx context.Context, request *SmartHomeAuthRequest) (*SmartHomeAuthResponse, error)
	AuthenticateVehicle(ctx context.Context, request *VehicleAuthRequest) (*VehicleAuthResponse, error)
	AuthenticateIIoT(ctx context.Context, request *IIoTAuthRequest) (*IIoTAuthResponse, error)
	GetDeviceHistory(ctx context.Context, deviceID string, limit, offset int) ([]*DeviceEvent, error)
}

type IoTDevice struct {
	DeviceID       string            `json:"device_id"`
	DeviceType     string            `json:"device_type"`
	DeviceName     string            `json:"device_name"`
	Manufacturer   string            `json:"manufacturer"`
	Model          string            `json:"model"`
	Firmware       string            `json:"firmware"`
	MACAddress     string            `json:"mac_address"`
	IPAddress      string            `json:"ip_address"`
	Location       string            `json:"location"`
	OwnerID        string            `json:"owner_id"`
	Status         string            `json:"status"`
	TrustScore     float64           `json:"trust_score"`
	Capabilities   []string          `json:"capabilities"`
	Metadata       map[string]string `json:"metadata"`
	RegisteredAt   time.Time         `json:"registered_at"`
	LastSeenAt     time.Time         `json:"last_seen_at"`
	ExpirationDate time.Time         `json:"expiration_date"`
}

type DeviceVerifyRequest struct {
	DeviceID       string            `json:"device_id"`
	Fingerprint    *DeviceFingerprint `json:"fingerprint"`
	Challenge      string            `json:"challenge"`
	Signature      string            `json:"signature"`
	Timestamp      time.Time         `json:"timestamp"`
	ConnectionInfo map[string]string `json:"connection_info"`
}

type DeviceVerifyResponse struct {
	Valid       bool      `json:"valid"`
	DeviceID    string    `json:"device_id"`
	TrustScore  float64   `json:"trust_score"`
	RiskLevel   string    `json:"risk_level"`
	Message     string    `json:"message"`
	VerifiedAt  time.Time `json:"verified_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type DeviceFingerprint struct {
	DeviceID        string    `json:"device_id"`
	HardwareHash    string    `json:"hardware_hash"`
	SoftwareHash    string    `json:"software_hash"`
	NetworkPattern  string    `json:"network_pattern"`
	BehavioralScore float64   `json:"behavioral_score"`
	Components      map[string]string `json:"components"`
	LastUpdated     time.Time `json:"last_updated"`
}

type SmartHomeAuthRequest struct {
	DeviceID   string            `json:"device_id"`
	DeviceType string            `json:"device_type"`
	Location   string            `json:"location"`
	Sensors    map[string]float64 `json:"sensors"`
	AuthMethod string            `json:"auth_method"`
	SessionKey string            `json:"session_key,omitempty"`
}

type SmartHomeAuthResponse struct {
	Authenticated bool      `json:"authenticated"`
	DeviceID      string    `json:"device_id"`
	AccessToken   string    `json:"access_token,omitempty"`
	ExpiresIn     int       `json:"expires_in"`
	Message       string    `json:"message"`
	Permissions   []string  `json:"permissions"`
	ZoneID        string    `json:"zone_id,omitempty"`
}

type VehicleAuthRequest struct {
	VehicleID     string `json:"vehicle_id"`
	VIN           string `json:"vin"`
	OBDeviceID    string `json:"obd_device_id"`
	Odometer      int    `json:"odometer"`
	Location      string `json:"location"`
	AuthChallenge string `json:"auth_challenge"`
	Signature     string `json:"signature"`
}

type VehicleAuthResponse struct {
	Authorized   bool      `json:"authorized"`
	VehicleID    string    `json:"vehicle_id"`
	Token        string    `json:"token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	Message      string    `json:"message"`
	SecurityLevel string   `json:"security_level"`
	Features     []string  `json:"features"`
}

type IIoTAuthRequest struct {
	DeviceID       string            `json:"device_id"`
	DeviceType     string            `json:"device_type"`
	PlantID        string            `json:"plant_id"`
	ProductionLine string            `json:"production_line"`
	Certificates   []string          `json:"certificates"`
	Measurements   map[string]float64 `json:"measurements"`
	Timestamp      time.Time         `json:"timestamp"`
}

type IIoTAuthResponse struct {
	Authorized    bool      `json:"authorized"`
	DeviceID      string    `json:"device_id"`
	SessionToken  string    `json:"session_token,omitempty"`
	ExpiresAt     time.Time `json:"expires_at"`
	Message       string    `json:"message"`
	Compliance    string    `json:"compliance"`
	AccessLevel   string    `json:"access_level"`
}

type DeviceEvent struct {
	EventID    string    `json:"event_id"`
	DeviceID   string    `json:"device_id"`
	EventType  string    `json:"event_type"`
	EventData  string    `json:"event_data"`
	Timestamp  time.Time `json:"timestamp"`
	RiskLevel  string    `json:"risk_level"`
	Result     string    `json:"result"`
}

type iotDeviceService struct {
	devices      map[string]*IoTDevice
	fingerprints map[string]*DeviceFingerprint
	events       map[string][]*DeviceEvent
}

var (
	ErrDeviceNotFound      = errors.New("device not found")
	ErrInvalidFingerprint = errors.New("invalid device fingerprint")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrDeviceExpired      = errors.New("device certificate expired")
	ErrUnsupportedDevice  = errors.New("unsupported device type")
)

func NewIoTDeviceService() IoTDeviceService {
	return &iotDeviceService{
		devices:      make(map[string]*IoTDevice),
		fingerprints: make(map[string]*DeviceFingerprint),
		events:       make(map[string][]*DeviceEvent),
	}
}

func (s *iotDeviceService) RegisterDevice(ctx context.Context, device *IoTDevice) error {
	if device == nil {
		return errors.New("device cannot be nil")
	}

	if device.DeviceID == "" {
		device.DeviceID = uuid.New().String()
	}
	if device.RegisteredAt.IsZero() {
		device.RegisteredAt = time.Now()
	}
	device.LastSeenAt = time.Now()

	if device.Status == "" {
		device.Status = "active"
	}
	if device.TrustScore == 0 {
		device.TrustScore = 50.0
	}

	s.devices[device.DeviceID] = device

	fingerprint := s.generateFingerprint(device)
	s.fingerprints[device.DeviceID] = fingerprint

	s.addEvent(device.DeviceID, "registration", "Device registered successfully", "low", "success")

	return nil
}

func (s *iotDeviceService) GetDevice(ctx context.Context, deviceID string) (*IoTDevice, error) {
	device, exists := s.devices[deviceID]
	if !exists {
		return nil, ErrDeviceNotFound
	}
	return device, nil
}

func (s *iotDeviceService) VerifyDevice(ctx context.Context, request *DeviceVerifyRequest) (*DeviceVerifyResponse, error) {
	device, exists := s.devices[request.DeviceID]
	if !exists {
		return nil, ErrDeviceNotFound
	}

	if !device.ExpirationDate.IsZero() && time.Now().After(device.ExpirationDate) {
		return &DeviceVerifyResponse{
			Valid:      false,
			DeviceID:   request.DeviceID,
			TrustScore: 0,
			RiskLevel:  "critical",
			Message:    "Device certificate expired",
		}, nil
	}

	valid := true
	riskLevel := "low"
	message := "Device verified successfully"

	if request.Fingerprint != nil {
		storedFP := s.fingerprints[request.DeviceID]
		if storedFP != nil {
			if request.Fingerprint.HardwareHash != storedFP.HardwareHash {
				valid = false
				riskLevel = "high"
				message = "Hardware fingerprint mismatch"
			}
		}
	}

	if request.Timestamp.IsZero() {
		request.Timestamp = time.Now()
	}

	if time.Since(request.Timestamp) > 5*time.Minute {
		valid = false
		riskLevel = "medium"
		message = "Request timestamp expired"
	}

	if valid {
		device.LastSeenAt = time.Now()
		device.TrustScore = s.calculateTrustScore(device)
	}

	response := &DeviceVerifyResponse{
		Valid:      valid,
		DeviceID:   request.DeviceID,
		TrustScore: device.TrustScore,
		RiskLevel:  riskLevel,
		Message:    message,
		VerifiedAt: time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	s.addEvent(request.DeviceID, "verification", message, riskLevel, map[bool]string{true: "success", false: "failed"}[valid])

	return response, nil
}

func (s *iotDeviceService) GetDeviceFingerprint(ctx context.Context, deviceID string) (*DeviceFingerprint, error) {
	fp, exists := s.fingerprints[deviceID]
	if !exists {
		return nil, ErrDeviceNotFound
	}
	return fp, nil
}

func (s *iotDeviceService) generateFingerprint(device *IoTDevice) *DeviceFingerprint {
	hwData := fmt.Sprintf("%s:%s:%s:%s", device.MACAddress, device.Manufacturer, device.Model, device.Firmware)
	hwHash := sha256.Sum256([]byte(hwData))

	swData := fmt.Sprintf("%s:%s:%v", device.DeviceType, strings.Join(device.Capabilities, ","), device.Metadata)
	swHash := sha256.Sum256([]byte(swData))

	netData := fmt.Sprintf("%s:%s", device.IPAddress, device.Location)
	netHash := sha256.Sum256([]byte(netData))

	return &DeviceFingerprint{
		DeviceID:       device.DeviceID,
		HardwareHash:   hex.EncodeToString(hwHash[:]),
		SoftwareHash:   hex.EncodeToString(swHash[:]),
		NetworkPattern: hex.EncodeToString(netHash[:]),
		BehavioralScore: 75.0,
		Components: map[string]string{
			"os":       device.Metadata["os"],
			"platform": device.DeviceType,
			"vendor":   device.Manufacturer,
		},
		LastUpdated: time.Now(),
	}
}

func (s *iotDeviceService) calculateTrustScore(device *IoTDevice) float64 {
	score := 50.0

	if device.Status == "active" {
		score += 10.0
	}

	age := time.Since(device.RegisteredAt)
	if age > 365*24*time.Hour {
		score += 20.0
	} else if age > 180*24*time.Hour {
		score += 15.0
	} else if age > 30*24*time.Hour {
		score += 10.0
	}

	if len(device.Capabilities) > 5 {
		score += 5.0
	}

	daysSinceLastSeen := time.Since(device.LastSeenAt).Hours() / 24
	if daysSinceLastSeen < 1 {
		score += 10.0
	} else if daysSinceLastSeen < 7 {
		score += 5.0
	} else if daysSinceLastSeen > 30 {
		score -= 10.0
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

func (s *iotDeviceService) AuthenticateSmartHome(ctx context.Context, request *SmartHomeAuthRequest) (*SmartHomeAuthResponse, error) {
	device, exists := s.devices[request.DeviceID]
	if !exists {
		return &SmartHomeAuthResponse{
			Authenticated: false,
			Message:       "Device not registered",
		}, nil
	}

	validSensors := s.validateSmartHomeSensors(request.Sensors)
	locationValid := s.validateSmartHomeLocation(request.Location)

	if !validSensors || !locationValid {
		s.addEvent(request.DeviceID, "smart_home_auth", "Authentication failed: validation error", "medium", "failed")
		return &SmartHomeAuthResponse{
			Authenticated: false,
			DeviceID:      request.DeviceID,
			Message:       "Sensor data or location validation failed",
		}, nil
	}

	permissions := s.getSmartHomePermissions(request.DeviceType)
	zoneID := s.assignZone(device.Location)

	device.LastSeenAt = time.Now()

	s.addEvent(request.DeviceID, "smart_home_auth", "Authentication successful", "low", "success")

	return &SmartHomeAuthResponse{
		Authenticated: true,
		DeviceID:      request.DeviceID,
		AccessToken:   uuid.New().String(),
		ExpiresIn:     3600,
		Message:       "Authentication successful",
		Permissions:   permissions,
		ZoneID:        zoneID,
	}, nil
}

func (s *iotDeviceService) validateSmartHomeSensors(sensors map[string]float64) bool {
	if len(sensors) == 0 {
		return false
	}

	for name, value := range sensors {
		switch strings.ToLower(name) {
		case "temperature":
			if value < -50 || value > 100 {
				return false
			}
		case "humidity":
			if value < 0 || value > 100 {
				return false
			}
		case "motion":
			if value != 0 && value != 1 {
				return false
			}
		}
	}

	return true
}

func (s *iotDeviceService) validateSmartHomeLocation(location string) bool {
	return len(location) > 0 && len(location) < 200
}

func (s *iotDeviceService) getSmartHomePermissions(deviceType string) []string {
	switch strings.ToLower(deviceType) {
	case "thermostat":
		return []string{"read_temperature", "write_temperature", "read_schedule"}
	case "camera":
		return []string{"read_video", "read_motion", "write_motion_alert"}
	case "lock":
		return []string{"read_status", "write_lock", "write_unlock"}
	case "light":
		return []string{"read_brightness", "write_brightness", "write_onoff"}
	default:
		return []string{"read_status", "write_settings"}
	}
}

func (s *iotDeviceService) assignZone(location string) string {
	return "zone_" + uuid.New().String()[:8]
}

func (s *iotDeviceService) AuthenticateVehicle(ctx context.Context, request *VehicleAuthRequest) (*VehicleAuthResponse, error) {
	device, exists := s.devices[request.VehicleID]
	if !exists {
		return &VehicleAuthResponse{
			Authorized: false,
			Message:   "Vehicle not registered",
		}, nil
	}

	if request.VIN == "" || len(request.VIN) != 17 {
		return &VehicleAuthResponse{
			Authorized: false,
			VehicleID:  request.VehicleID,
			Message:    "Invalid VIN format",
		}, nil
	}

	vinValid := s.validateVIN(request.VIN)
	if !vinValid {
		return &VehicleAuthResponse{
			Authorized: false,
			VehicleID:  request.VehicleID,
			Message:    "VIN validation failed",
		}, nil
	}

	securityLevel := s.calculateVehicleSecurityLevel(device)
	features := s.getVehicleFeatures(device)

	device.LastSeenAt = time.Now()

	s.addEvent(request.VehicleID, "vehicle_auth", "Vehicle authentication successful", "low", "success")

	return &VehicleAuthResponse{
		Authorized:    true,
		VehicleID:     request.VehicleID,
		Token:         uuid.New().String(),
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		Message:       "Vehicle authorized",
		SecurityLevel: securityLevel,
		Features:      features,
	}, nil
}

func (s *iotDeviceService) validateVIN(vin string) bool {
	if len(vin) != 17 {
		return false
	}

	for _, c := range vin {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}

	weights := []int{8, 7, 6, 5, 4, 3, 2, 10, 0, 9, 8, 7, 6, 5, 4, 3, 2}
	transliteration := map[rune]int{
		'A': 1, 'B': 2, 'C': 3, 'D': 4, 'E': 5, 'F': 6, 'G': 7, 'H': 8,
		'J': 1, 'K': 2, 'L': 3, 'M': 4, 'N': 5, 'P': 7, 'R': 9,
		'S': 2, 'T': 3, 'U': 4, 'V': 5, 'W': 6, 'X': 7, 'Y': 8, 'Z': 9,
	}

	sum := 0
	for i, c := range strings.ToUpper(vin) {
		var value int
		if c >= '0' && c <= '9' {
			value = int(c - '0')
		} else {
			value = transliteration[c]
		}
		sum += value * weights[i]
	}

	checkDigit := sum % 11
	if checkDigit == 10 {
		return string(vin[8]) == "X"
	}
	return string(vin[8]) == string(rune('0'+checkDigit))
}

func (s *iotDeviceService) calculateVehicleSecurityLevel(device *IoTDevice) string {
	if device.TrustScore >= 80 {
		return "level_4"
	} else if device.TrustScore >= 60 {
		return "level_3"
	} else if device.TrustScore >= 40 {
		return "level_2"
	}
	return "level_1"
}

func (s *iotDeviceService) getVehicleFeatures(device *IoTDevice) []string {
	features := []string{"remote_start", "location_tracking", "diagnostics"}

	if device.Capabilities != nil {
		for _, cap := range device.Capabilities {
			switch strings.ToLower(cap) {
			case "autopilot":
				features = append(features, "autopilot")
			case "remote_climate":
				features = append(features, "remote_climate_control")
			case "security":
				features = append(features, "enhanced_security", "geofencing")
			}
		}
	}

	return features
}

func (s *iotDeviceService) AuthenticateIIoT(ctx context.Context, request *IIoTAuthRequest) (*IIoTAuthResponse, error) {
	device, exists := s.devices[request.DeviceID]
	if !exists {
		return &IIoTAuthResponse{
			Authorized: false,
			DeviceID:   request.DeviceID,
			Message:    "IIoT device not registered",
		}, nil
	}

	if request.PlantID == "" || request.ProductionLine == "" {
		return &IIoTAuthResponse{
			Authorized:  false,
			DeviceID:    request.DeviceID,
			Message:     "Missing plant or production line information",
		}, nil
	}

	if request.Timestamp.IsZero() {
		request.Timestamp = time.Now()
	}

	if time.Since(request.Timestamp) > 5*time.Minute {
		return &IIoTAuthResponse{
			Authorized: false,
			DeviceID:   request.DeviceID,
			Message:    "Request timestamp expired",
		}, nil
	}

	if !s.validateIIoTCertificates(request.Certificates) {
		return &IIoTAuthResponse{
			Authorized:  false,
			DeviceID:    request.DeviceID,
			Message:     "Certificate validation failed",
		}, nil
	}

	compliance := s.checkCompliance(request)
	accessLevel := s.determineIIoTAccessLevel(device)

	device.LastSeenAt = time.Now()

	s.addEvent(request.DeviceID, "iiot_auth", "IIoT authentication successful", "low", "success")

	return &IIoTAuthResponse{
		Authorized:   true,
		DeviceID:     request.DeviceID,
		SessionToken: uuid.New().String(),
		ExpiresAt:    time.Now().Add(8 * time.Hour),
		Message:      "IIoT device authorized",
		Compliance:   compliance,
		AccessLevel:  accessLevel,
	}, nil
}

func (s *iotDeviceService) validateIIoTCertificates(certificates []string) bool {
	if len(certificates) == 0 {
		return true
	}

	for _, cert := range certificates {
		if len(cert) < 5 {
			return false
		}
	}

	return true
}

func (s *iotDeviceService) checkCompliance(request *IIoTAuthRequest) string {
	if len(request.Certificates) >= 2 {
		return "compliant"
	} else if len(request.Certificates) == 1 {
		return "partial"
	}
	return "pending"
}

func (s *iotDeviceService) determineIIoTAccessLevel(device *IoTDevice) string {
	switch strings.ToLower(device.DeviceType) {
	case "plc":
		return "full_control"
	case "sensor":
		return "read_only"
	case "actuator":
		return "write_limited"
	case "gateway":
		return "full_access"
	default:
		return "standard"
	}
}

func (s *iotDeviceService) GetDeviceHistory(ctx context.Context, deviceID string, limit, offset int) ([]*DeviceEvent, error) {
	events, exists := s.events[deviceID]
	if !exists {
		return []*DeviceEvent{}, nil
	}

	if offset >= len(events) {
		return []*DeviceEvent{}, nil
	}

	end := offset + limit
	if end > len(events) {
		end = len(events)
	}

	return events[offset:end], nil
}

func (s *iotDeviceService) addEvent(deviceID, eventType, eventData, riskLevel, result string) {
	event := &DeviceEvent{
		EventID:   uuid.New().String(),
		DeviceID:  deviceID,
		EventType: eventType,
		EventData: eventData,
		Timestamp: time.Now(),
		RiskLevel: riskLevel,
		Result:    result,
	}

	s.events[deviceID] = append(s.events[deviceID], event)

	if len(s.events[deviceID]) > 1000 {
		s.events[deviceID] = s.events[deviceID][len(s.events[deviceID])-1000:]
	}
}
