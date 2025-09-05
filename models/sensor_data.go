package models

import (
	"time"
)

// SensorData represents sensor reading data
type SensorData struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Timestamp  time.Time `gorm:"uniqueIndex:idx_timestamp_sensor;not null" json:"timestamp"`
	SensorName string    `gorm:"uniqueIndex:idx_timestamp_sensor;not null;size:255" json:"sensor_name"`
	Value      float64   `gorm:"not null" json:"value"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName customizes the table name
func (SensorData) TableName() string {
	return "sensor_data"
}

// GetAllModels returns all models for migration
func GetAllModels() []interface{} {
	return []interface{}{
		&SensorData{},
	}
}
