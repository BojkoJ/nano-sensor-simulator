package simulator_test

import (
	"math/rand"
	"testing"

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
// nano-sensor-simulator/
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
// nano-sensor-simulator/
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
// nano-sensor-simulator/
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

func TestGenerateTemperature_WithinBounds(t *testing.T) {
	tests := []struct {
		name           string
		minTemperature float64
		maxTemperature float64
		samples        int
	}{
		{
			name:           "normal operating range",
			minTemperature: -10.0,
			maxTemperature: 50.0,
			samples:        10000,
		},
		{
			name:           "cold storage range",
			minTemperature: -30.0,
			maxTemperature: -20.0,
			samples:        10000,
		},
		{
			name:           "server room range",
			minTemperature: 18.0,
			maxTemperature: 27.0,
			samples:        10000,
		},
		{
			name:           "extreme range",
			minTemperature: -273.15,
			maxTemperature: 1000.0,
			samples:        10000,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(42)) // Deterministický seed pro reprodukovatelnost

			for i := 0; i < testCase.samples; i++ {
				temperature := simulator.GenerateTemperature(testCase.minTemperature, testCase.maxTemperature, rng)

				if temperature < testCase.minTemperature || temperature > testCase.maxTemperature {
					t.Errorf("sample %d: temperature %.4f out of range [%.1f, %.1f]",
						i, temperature, testCase.minTemperature, testCase.maxTemperature)
				}
			}
		})
	}
}

func TestIsOutOfRange(t *testing.T) {
	tests := []struct {
		name           string
		temperature    float64
		minTemperature float64
		maxTemperature float64
		wantOut        bool
	}{
		{
			name:           "within range",
			temperature:    25.0,
			minTemperature: 0.0,
			maxTemperature: 50.0,
			wantOut:        false,
		},
		{
			name:           "at min boundary",
			temperature:    0.0,
			minTemperature: 0.0,
			maxTemperature: 50.0,
			wantOut:        false,
		},
		{
			name:           "at max boundary",
			temperature:    50.0,
			minTemperature: 0.0,
			maxTemperature: 50.0,
			wantOut:        false,
		},
		{
			name:           "below min",
			temperature:    -5.0,
			minTemperature: 0.0,
			maxTemperature: 50.0,
			wantOut:        true,
		},
		{
			name:           "above max",
			temperature:    55.0,
			minTemperature: 0.0,
			maxTemperature: 50.0,
			wantOut:        true,
		},
		{
			name:           "far below min",
			temperature:    -50.0,
			minTemperature: 0.0,
			maxTemperature: 50.0,
			wantOut:        true,
		},
		{
			name:           "far above max",
			temperature:    100.0,
			minTemperature: 0.0,
			maxTemperature: 50.0,
			wantOut:        true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := simulator.IsOutOfRange(testCase.temperature, testCase.minTemperature, testCase.maxTemperature)
			if got != testCase.wantOut {
				t.Errorf("IsOutOfRange(%.1f, %.1f, %.1f) = %v, want %v",
					testCase.temperature, testCase.minTemperature, testCase.maxTemperature, got, testCase.wantOut)
			}
		})
	}
}

func BenchmarkGenerateTemperature(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = simulator.GenerateTemperature(-10.0, 50.0, rng)
	}
}
