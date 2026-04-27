package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/BojkoJ/nano-sensor-simulator/internal/simulator"
)

// ---------------------------------------------------------------------------
// Mini projekt. Část 1a:
// Napsat GO CLI aplikaci "nano-sensor-simulator", která:
// 1. Inicializuje 3 "senzory" (ID: sensor-1, sensor-2, sensor-3)
// 2. Každou sekundu vygeneruje pro každý senzor:
//    - teplotu (float64, 15-45°C s náhodnout fluktuací),
//    - timestamp
// 3. Vytiskne výstup do stdout jako JSON (encoding/json)
// 4. Pokud teplota > 40°C, vytiskne WARNING na stderr
// 5. Gracefully se ukončí při Ctrl+C (os.Signal)
//
// Struktura projektu:
// go-learning-project-sensors/
// ├── main.go    (vstupní bod, signal handling, ticker loop)
// ├── sensor.go  (sensor logika, generování dat)
// └── go.mod
// ---------------------------------------------------------------------------
// Mini projekt. Část 1b:
// Rozšířit CLI simulátor ze sekce 1a.
// Každý senzor poběží jako samostatná goroutina, výsledky se agregují přes channel,
// Program správně reaguje na Ctrl+C a obsahuje unit testy (sensor_test.go).
//
// Nová struktura projektu:
// go-learning-project-sensors/
// ├── main.go
// ├── simulator/
// │   ├── sensor.go
// │   └── sensor_test.go (unit testing)
// └── go.mod
// ---------------------------------------------------------------------------
// Mini projekt. Část 2:
// Vytvořit multi-stage Dockerfile pro simulátor z Lekce 1.
// Výsledný image musí mít méně než 20MB.
// Aplikace běží jako non-root user
// Vytvořit docker-compose.yml, který spustí simulátor.
// Ověřit, že docker logs zobrazuje JSON výstup ze senzorů
// Uklidit folder/file strukturu projektu.
//
// Nová struktura projektu:
// ├── cmd/
// │   └── simulator/
// │       └──  main.go
// ├── internal
// │   └── simulator
// │       ├── sensor.go
// │       └── sensor_test.go (unit testing)
// ├── deploy/
// │   └── docker/
// │       ├── .dockerignore
// │       └── Dockerfile
// ├── docker-compose.yml
// ├── .gitignore
// └── go.mod
// ---------------------------------------------------------------------------

type Output struct {
	Sensor      string `json:"sensor"`
	Temperature string `json:"temperature"`
	Status      string `json:"status"`
	Time        string `json:"time"`
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Definice senzorů
	sensors := []simulator.Config{
		{
			ID:       "TEMP-001",
			MinTemp:  -10.0,
			MaxTemp:  50.0,
			Interval: time.Second,
		},
		{
			ID:       "TEMP-002",
			MinTemp:  15.0,
			MaxTemp:  30.0,
			Interval: time.Second,
		},
		{
			ID:       "TEMP-003",
			MinTemp:  -30.0,
			MaxTemp:  -10.0,
			Interval: time.Second,
		},
	}

	// Buffered channel pro agregaci výsledků ze všech senzorů
	// Kapacita = počet senzorů *10 pro absorpci krátkodobých špiček
	readings := make(chan simulator.Reading, len(sensors)*10)

	// Context pro graceful stutdown
	ctx, cancel := context.WithCancel(context.Background())

	// WaitGroup pro čekání na dokončení goroutin
	var wg sync.WaitGroup

	// JSON Encoder pro výstup na stdout
	encoder := json.NewEncoder(os.Stdout)

	// Spuštění goroutin pro každý senzor
	for _, cfg := range sensors {
		cfg := cfg // capture loop variable
		wg.Add(1)

		sensorLogger := logger.With("sensor_id", cfg.ID)

		go func() {
			defer wg.Done()
			sensor := simulator.NewSensor(cfg, sensorLogger)
			sensor.Run(ctx, readings) // Blokuje dokud ctx není zrušen (cancelled)
		}()
	}

	// goroutina pro uzavření readings channelu po dokončení všech senzorů
	go func() {
		wg.Wait()
		close(readings)
		logger.Info("all sensors stopped, readings channel closed")
	}()

	// Naslouchání na OS signály (Ctrl+C nebo kill)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Goroutina pro zpracování signálů
	go func() {
		sig := <-sigCh
		logger.Info("shutdown signal received", "signal", sig.String())
		cancel() // Zruší context -> ukončí všechny senzory
	}()

	logger.Info("simulator started",
		"sensors", len(sensors),
		"press", "Ctrl+C to stop")

	// Agregace výsledků v main goroutině
	var (
		totalReadings int
		anomalyCount  int
	)

	var output Output

	for reading := range readings {
		totalReadings++
		anomalyStatus := "OK"

		if reading.IsAnomaly {
			anomalyCount++
			anomalyStatus = "ANOMALY"
		}

		// Chceme toto logovat ve formátu JSON:

		// Takto by to bylo skrze slog logger na stdout
		// logger.Info("reading",
		// 	"sensor", reading.SensorID,
		//	"temperature", fmt.Sprintf("%.2f°C", reading.Temperature),
		//	"status", anomalyStatus,
		//	"time", reading.Timestamp.Format("15:04:05"),
		// )

		output = Output{
			Sensor:      reading.SensorID,
			Temperature: fmt.Sprintf("%.2f°C", reading.Temperature),
			Status:      anomalyStatus,
			Time:        reading.Timestamp.Format("15:04:05"),
		}

		// Zapíšeme JSON na stdout
		// funkce Encode zapíše JSON reprezentaci objektu output do writeru (v našem případě os.Stdout)
		// automaticky na stdout, protože json.NewEncoder(os.Stdout)
		if err := encoder.Encode(output); err != nil {
			_, err := fmt.Fprintf(os.Stderr, "ERROR: encode output: %v\n", err)
			if err != nil {
				log.Fatal(err)
				return
			}
			continue
		}
	}

	logger.Info("simulation complete",
		"total_readings", totalReadings,
		"anomalies", anomalyCount,
		"anomaly_rate", fmt.Sprintf("%.1f%%", float64(anomalyCount)/float64(totalReadings)*100),
	)

}

// $env:CGO_ENABLED = "1"
// nutné pro go run -race ./...
// + C kompilátor (pro windows např w64devkit)
