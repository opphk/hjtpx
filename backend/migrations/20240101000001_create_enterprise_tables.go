package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func init() {
	m := &gormigrate.Migration{
		ID: "20240101000001",
		Migrate: func(tx *gorm.DB) error {
			// SSO Provider table
			type SSOProvider struct {
				ID          uint   `gorm:"primaryKey"`
				Name        string `gorm:"size:100;not null"`
				Type        string `gorm:"size:50;not null"` // saml, oauth, oidc
				MetadataURL string `gorm:"size:500"`
				EntityID    string `gorm:"size:500"`
				SsoURL      string `gorm:"size:500"`
				CertData    string `gorm:"type:text"`
				CreatedAt   int64  `gorm:"autoCreateTime"`
				UpdatedAt   int64  `gorm:"autoUpdateTime"`
			}

			// SAML Provider table
			type SAMLProvider struct {
				ID               uint   `gorm:"primaryKey"`
				Name             string `gorm:"size:100;not null"`
				MetadataXML      string `gorm:"type:text"`
				EntityID         string `gorm:"size:500;unique"`
				SingleSignOnURL  string `gorm:"size:500"`
				SingleLogoutURL  string `gorm:"size:500"`
				X509Certificate  string `gorm:"type:text"`
				Active           bool   `gorm:"default:true"`
				CreatedAt        int64  `gorm:"autoCreateTime"`
				UpdatedAt        int64  `gorm:"autoUpdateTime"`
			}

			// OIDC Provider table
			type OIDCProvider struct {
				ID                uint   `gorm:"primaryKey"`
				Name              string `gorm:"size:100;not null"`
				Issuer            string `gorm:"size:500;unique"`
				ClientID          string `gorm:"size:255"`
				ClientSecret      string `gorm:"size:255"`
				AuthorizationURL  string `gorm:"size:500"`
				TokenURL          string `gorm:"size:500"`
				UserInfoURL       string `gorm:"size:500"`
				EndSessionURL     string `gorm:"size:500"`
				Scope             string `gorm:"size:500"`
				Active            bool   `gorm:"default:true"`
				CreatedAt         int64  `gorm:"autoCreateTime"`
				UpdatedAt         int64  `gorm:"autoUpdateTime"`
			}

			// OIDC User Session table
			type OIDCUserSession struct {
				ID            uint   `gorm:"primaryKey"`
				ProviderID    uint
				State         string `gorm:"size:255"`
				CodeVerifier  string `gorm:"size:255"`
				RedirectURI   string `gorm:"size:500"`
				Nonce         string `gorm:"size:255"`
				ExpiresAt     int64
				CreatedAt     int64 `gorm:"autoCreateTime"`
			}

			// SCIM Provider table
			type SCIMProvider struct {
				ID           uint   `gorm:"primaryKey"`
				TenantID     uint
				Name         string `gorm:"size:100;not null"`
				BaseURL      string `gorm:"size:500"`
				AuthToken    string `gorm:"size:500"`
				Active       bool   `gorm:"default:true"`
				LastSyncTime int64
				CreatedAt    int64 `gorm:"autoCreateTime"`
				UpdatedAt    int64 `gorm:"autoUpdateTime"`
			}

			// SCIM User Mapping table
			type SCIMUserMapping struct {
				ID            uint   `gorm:"primaryKey"`
				ProviderID    uint
				ExternalID    string `gorm:"size:255;unique"`
				InternalUserID uint
				LastSyncTime  int64
				CreatedAt     int64 `gorm:"autoCreateTime"`
				UpdatedAt     int64 `gorm:"autoUpdateTime"`
			}

			// SCIM Group table
			type SCIMGroup struct {
				ID           uint   `gorm:"primaryKey"`
				ProviderID   uint
				ExternalID   string `gorm:"size:255"`
				DisplayName  string `gorm:"size:255"`
				CreatedAt    int64  `gorm:"autoCreateTime"`
				UpdatedAt    int64  `gorm:"autoUpdateTime"`
			}

			// Audit Log table
			type AuditLog struct {
				ID            uint      `gorm:"primaryKey"`
				LogType       string    `gorm:"size:50;not null"` // api_request, authentication, authorization, security_event
				UserID        uint
				UserName      string    `gorm:"size:100"`
				ClientIP      string    `gorm:"size:50"`
				UserAgent     string    `gorm:"size:500"`
				Endpoint      string    `gorm:"size:500"`
				Method        string    `gorm:"size:10"`
				Status        string    `gorm:"size:20"` // success, failed, denied
				StatusCode    int
				RequestData   string    `gorm:"type:text"`
				ResponseData  string    `gorm:"type:text"`
				ErrorMessage  string    `gorm:"type:text"`
				DurationMs    int64
				Timestamp     int64     `gorm:"autoCreateTime"`
			}

			// Data Export Request table
			type DataExportRequest struct {
				ID           uint   `gorm:"primaryKey"`
				UserID       uint
				Status       string `gorm:"size:20;default:'pending'"` // pending, processing, completed, failed
				ExportType   string `gorm:"size:50"`
				DownloadURL  string `gorm:"size:500"`
				RequestedAt  int64  `gorm:"autoCreateTime"`
				CompletedAt  int64
			}

			// Data Deletion Request table
			type DataDeletionRequest struct {
				ID          uint   `gorm:"primaryKey"`
				UserID      uint
				Status      string `gorm:"size:20;default:'pending'"` // pending, processing, completed, failed
				RequestedAt int64  `gorm:"autoCreateTime"`
				CompletedAt int64
			}

			// User Consent table
			type UserConsent struct {
				ID                  uint   `gorm:"primaryKey"`
				UserID              uint   `gorm:"unique"`
				ConsentMarketing    bool   `gorm:"default:false"`
				ConsentAnalytics    bool   `gorm:"default:false"`
				ConsentPersonalization bool `gorm:"default:false"`
				ConsentDataSharing  bool   `gorm:"default:false"`
				ConsentUpdatedAt    int64
				CreatedAt           int64 `gorm:"autoCreateTime"`
				UpdatedAt           int64 `gorm:"autoUpdateTime"`
			}

			tables := []interface{}{
				&SSOProvider{},
				&SAMLProvider{},
				&OIDCProvider{},
				&OIDCUserSession{},
				&SCIMProvider{},
				&SCIMUserMapping{},
				&SCIMGroup{},
				&AuditLog{},
				&DataExportRequest{},
				&DataDeletionRequest{},
				&UserConsent{},
			}

			for _, table := range tables {
				if err := tx.AutoMigrate(table); err != nil {
					return err
				}
			}

			// Create indexes
			indexes := []struct {
				table  string
				column string
			}{
				{"sso_providers", "type"},
				{"saml_providers", "entity_id"},
				{"oidc_providers", "issuer"},
				{"scim_providers", "tenant_id"},
				{"scim_user_mappings", "external_id"},
				{"scim_user_mappings", "internal_user_id"},
				{"audit_logs", "log_type"},
				{"audit_logs", "user_id"},
				{"audit_logs", "timestamp"},
				{"data_export_requests", "user_id"},
				{"data_export_requests", "status"},
				{"data_deletion_requests", "user_id"},
				{"data_deletion_requests", "status"},
				{"user_consents", "user_id"},
			}

			for _, idx := range indexes {
				if err := tx.Exec(
					"CREATE INDEX IF NOT EXISTS idx_" + idx.table + "_" + idx.column + " ON " + idx.table + "(" + idx.column + ")",
				).Error; err != nil {
					return err
				}
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			tables := []string{
				"user_consents",
				"data_deletion_requests",
				"data_export_requests",
				"audit_logs",
				"scim_groups",
				"scim_user_mappings",
				"scim_providers",
				"oidc_user_sessions",
				"oidc_providers",
				"saml_providers",
				"sso_providers",
			}

			for _, table := range tables {
				if err := tx.Migrator().DropTable(table); err != nil {
					return err
				}
			}

			return nil
		},
	}

	RegisterMigration(m)
}