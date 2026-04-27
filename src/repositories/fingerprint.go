package repositories

import (
	"blog_api/src/model"

	"gorm.io/gorm"
)

// GetFingerprintByValue retrieves a fingerprint by its hash value.
func GetFingerprintByValue(db *gorm.DB, fingerprint string) (*model.Fingerprint, error) {
	var record model.Fingerprint
	if err := db.Where("fingerprint = ?", fingerprint).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// GetFingerprintByID retrieves a fingerprint by its primary key.
func GetFingerprintByID(db *gorm.DB, id int) (*model.Fingerprint, error) {
	var record model.Fingerprint
	if err := db.First(&record, id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// CreateFingerprint inserts a new fingerprint record.
func CreateFingerprint(db *gorm.DB, fingerprint *model.Fingerprint) error {
	return db.Create(fingerprint).Error
}

// UpdateFingerprintIdentity updates the fingerprint hash and current request identity fields.
func UpdateFingerprintIdentity(db *gorm.DB, id int, fingerprint, userAgent, ip string) error {
	return db.Model(&model.Fingerprint{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"fingerprint": fingerprint,
			"user_agent":  userAgent,
			"ip":          ip,
		}).Error
}
