package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run generate_test_data.go <output_directory>")
		fmt.Println("Example: go run generate_test_data.go test_data")
		return
	}

	outputDir := os.Args[1]

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Failed to create directory: %v\n", err)
		return
	}

	rand.Seed(time.Now().UnixNano())

	// Generate different types of sensor data
	generators := []Generator{
		{"temperature_hourly.csv", generateTemperatureData},
		{"humidity_realtime.csv", generateHumidityData},
		{"pressure_daily.csv", generatePressureData},
		{"light_sensors.csv", generateLightData},
		{"vibration_sensors.csv", generateVibrationData},
	}

	var wg sync.WaitGroup
	for i, gen := range generators {
		wg.Add(1)
		go generateMockData(i, outputDir, gen, &wg)
	}
	wg.Wait()
	fmt.Println("All mocked data generated.")
}

type Generator struct {
	filename  string
	generator func() []SensorReading
}

type SensorReading struct {
	Timestamp  time.Time
	SensorName string
	Value      float64
}

const numberOfDays = 5 * 365

func generateMockData(id int, outputDir string, generator Generator, wg *sync.WaitGroup) {
	defer wg.Done()
	csvFilepath := filepath.Join(outputDir, generator.filename)
	data := generator.generator()

	if err := writeCSV(csvFilepath, data); err != nil {
		fmt.Printf("Failed to write %s: %v\n", generator.filename, err)
		return
	}

	fmt.Printf("Generated %s with %d records\n", generator.filename, len(data))
}

func generateTemperatureData() []SensorReading {
	var readings []SensorReading
	sensors := []string{"temp_sensor_01", "temp_sensor_02", "temp_sensor_03", "temp_sensor_04"}

	start := time.Now().UTC().AddDate(0, -1, -numberOfDays)

	for day := 0; day < numberOfDays; day++ {
		for i := 0; i < 288; i++ { // 24 hours * 12 (every 5 minutes)
			timestamp := start.Add(time.Duration(i) * 5 * time.Minute)

			for j, sensor := range sensors {
				// Simulate temperature with daily cycle + noise
				hourAngle := float64(timestamp.Hour()) * math.Pi / 12
				baseTemp := 20.0 + 8.0*math.Sin(hourAngle-math.Pi/2) // Daily temperature cycle
				noise := rand.Float64()*2 - 1                        // ±1 degree noise
				sensorOffset := float64(j) * 0.5                     // Each sensor slightly different

				reading := SensorReading{
					Timestamp:  timestamp,
					SensorName: sensor,
					Value:      baseTemp + noise + sensorOffset,
				}
				readings = append(readings, reading)
			}
		}
		start = start.AddDate(0, 0, 1)
	}

	return readings
}

func generateHumidityData() []SensorReading {
	var readings []SensorReading
	sensors := []string{"humidity_sensor_01", "humidity_sensor_02"}

	start := time.Now().UTC().AddDate(0, -1, -numberOfDays)

	for day := 0; day < numberOfDays; day++ {
		for i := 0; i < 480; i++ { // 8 hours * 60 (every minute)
			timestamp := start.Add(time.Duration(i) * time.Minute)

			for j, sensor := range sensors {
				// Simulate humidity decreasing during day
				baseHumidity := 70.0 - float64(i)/480*15 // Decrease from 70% to 55%
				noise := rand.Float64()*4 - 2            // ±2% noise
				sensorOffset := float64(j) * 2.0         // Different sensor readings

				reading := SensorReading{
					Timestamp:  timestamp,
					SensorName: sensor,
					Value:      math.Max(30, math.Min(95, baseHumidity+noise+sensorOffset)),
				}
				readings = append(readings, reading)
			}
		}
		start = start.AddDate(0, 0, 1)
	}

	return readings
}

func generatePressureData() []SensorReading {
	var readings []SensorReading
	sensors := []string{"pressure_sensor_01", "pressure_sensor_02", "pressure_sensor_03"}

	start := time.Now().UTC().AddDate(0, -1, -numberOfDays)

	for day := 0; day < numberOfDays; day++ {
		for i := 0; i < 24; i++ { // 24 hours (every hour)
			timestamp := start.Add(time.Duration(i) * time.Hour)

			for j, sensor := range sensors {
				// Simulate atmospheric pressure with small variations
				basePressure := 1013.25
				variation := math.Sin(float64(i)*math.Pi/12) * 2 // ±2 hPa variation
				noise := rand.Float64()*0.5 - 0.25               // ±0.25 hPa noise
				sensorOffset := float64(j) * 0.1                 // Small sensor differences

				reading := SensorReading{
					Timestamp:  timestamp,
					SensorName: sensor,
					Value:      basePressure + variation + noise + sensorOffset,
				}
				readings = append(readings, reading)
			}
		}
		start = start.AddDate(0, 0, 1)
	}

	return readings
}

func generateLightData() []SensorReading {
	var readings []SensorReading
	sensors := []string{"light_sensor_01", "light_sensor_02", "light_sensor_03", "light_sensor_04", "light_sensor_05"}

	start := time.Now().UTC().AddDate(0, -1, -numberOfDays)

	for day := 0; day < numberOfDays; day++ {
		for i := 0; i < 720; i++ { // 12 hours * 60 (every minute)
			timestamp := start.Add(time.Duration(i) * time.Minute)

			for j, sensor := range sensors {
				// Simulate light intensity (sunrise to sunset)
				hour := float64(timestamp.Hour()) + float64(timestamp.Minute())/60
				var lightLevel float64

				if hour < 6 || hour > 18 {
					lightLevel = rand.Float64() * 10 // Night: 0-10 lux
				} else {
					// Day: simulate sun arc
					sunAngle := (hour - 6) * math.Pi / 12
					lightLevel = 1000 * math.Sin(sunAngle) * (0.8 + rand.Float64()*0.4)
				}

				sensorOffset := float64(j) * 20 // Different locations
				noise := rand.Float64()*50 - 25

				reading := SensorReading{
					Timestamp:  timestamp,
					SensorName: sensor,
					Value:      math.Max(0, lightLevel+sensorOffset+noise),
				}
				readings = append(readings, reading)
			}
		}
		start = start.AddDate(0, 0, 1)
	}

	return readings
}

func generateVibrationData() []SensorReading {
	var readings []SensorReading
	sensors := []string{"vibration_sensor_01", "vibration_sensor_02"}

	start := time.Now().UTC().AddDate(0, -1, -numberOfDays)

	for day := 0; day < numberOfDays; day++ {
		for i := 0; i < 3600; i++ { // 1 hour * 3600 (every second)
			timestamp := start.Add(time.Duration(i) * time.Second)

			for j, sensor := range sensors {
				// Simulate vibration with periodic spikes
				baseVibration := rand.Float64() * 0.5 // Base noise

				// Add periodic machinery vibration
				if i%300 == 0 { // Every 5 minutes
					baseVibration += rand.Float64()*2 + 3 // Spike: 3-5 units
				}

				sensorOffset := float64(j) * 0.2

				reading := SensorReading{
					Timestamp:  timestamp,
					SensorName: sensor,
					Value:      baseVibration + sensorOffset,
				}
				readings = append(readings, reading)
			}
		}
		start = start.AddDate(0, 0, 1)
	}

	return readings
}

func writeCSV(filename string, readings []SensorReading) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header
	if _, err := file.WriteString("timestamp,sensor_name,value\n"); err != nil {
		return err
	}

	// Write data
	for _, reading := range readings {
		line := fmt.Sprintf("%s,%s,%.2f\n",
			reading.Timestamp.Format("2006-01-02T15:04:05Z"),
			reading.SensorName,
			reading.Value)
		if _, err := file.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}
