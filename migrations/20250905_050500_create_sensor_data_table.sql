-- Migration: Create sensor_data table
-- Created: 2025-09-05 05:05:00
-- Description: Create sensor_data table with composite primary key on timestamp and sensor_name

CREATE TABLE sensor_data (
    id INT AUTO_INCREMENT PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    sensor_name VARCHAR(255) NOT NULL,
    value DOUBLE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE INDEX idx_timestamp_sensor (timestamp, sensor_name),
    INDEX idx_sensor_name (sensor_name),
    INDEX idx_timestamp (timestamp),
    INDEX idx_created_at (created_at)
);
