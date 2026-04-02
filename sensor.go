package main

import (
	"math"
	"math/rand"
	"time"
)

// ---------------------------------------------------------------------------
// Mini projekt. Část 1:
// Napsat GO CLI aplikaci "nano-sensor-simulator", která:
// 1. Inicializuje 3 "senzory" (ID: sensor-1, sensor-2, sensor-3)
// 2. Každou sekundu vygeneruje pro každý senzor:
//    - teplotu (float64, 15-45°C s náhodnout fluktuací),
//    - timestamp
// 3. Vytiskne výstup do stdout jako JSON (encoding/json)
// 4. Pokud teplota > 40°C, vytiskne WARNING na stderr
// 5. Gracefully se ukončí při Ctrl+C (os.Signal)

// Struktura projektu:
// mini-project/
// ├── main.go    (vstupní bod, signal handling, ticker loop)
// ├── sensor.go  (sensor logika, generování dat)
// └── go.mod
// ---------------------------------------------------------------------------

// Sensor struktura reprezentuje jeden IoT senzor s interním stavem
type Sensor struct {
	ID          string   // ID senzoru
	baseTemp    float64  // základní teplota pro simulaci, neexportujeme - interní detail implementace
	lastReading *Reading // pointer, protože chceme inicializovat bez hodnoty, kdyby to nebyl pointer, museli bychom ho inicializovat s nějakou defaultní hodnotou, což by mohlo být matoucí
}

// Reading je jedno naměřené čtení senzory
type Reading struct {
	// `json:"..."` jsou struct tags, které říkají, jak pojmenovat pole v JSONu
	SensorID    string    `json:"sensor_id"`   // ID senzoru, který provedl měření
	Temperature float64   `json:"temperature"` // Naměřená teplota senzoru v tomto jeho jednom čtení
	Timestamp   time.Time `json:"timestamp"`   // Čas, kdy bylo měření provedeno
	Unit        string    `json:"unit"`        // Jednotka měření (jako celsius: °C, kelvin: K, Fahrenheit: °F)
}

// NewSensor vytvoří nový seznor s daným ID a základní teplotou
// baseTemp simuluje "okolní teplotu" senzoru (15-30°C)
func NewSensor(id string) *Sensor {
	return &Sensor{
		ID:       id,
		baseTemp: 15 + rand.Float64()*15, // 15 + náhodná hodnota od 0 do 15 (čili náhodná hodnota v rozsahu 15-30)
	}
}

// GenerateReading vytvoří nové čtení senzoru se simulovanou fluktuací teploty
// Teplota fluktuuje +-5°C od základní teploty, občas "skočí" výše
func (s *Sensor) GenerateReading() Reading {
	// Základní náhodná hodnota fluktuace: +- 5°C
	// proč takto: vyber náhodně hodnotu 0-10 a odečti od ní 5:
	// vybere třeba 9, odečte 5 a máme: 4
	// vybere třeba 1, odečte 5 a máme: -4
	// tak se zajistí náhodný výběr od -5 do 5, protože rand.Float64 umí generovat jen od 0
	fluctuation := (rand.Float64() * 10.0) - 5.0

	// 10% šance na "teplotní skok" (simulace abnormálního stavu)
	if rand.Float64() < 0.10 {
		fluctuation += rand.Float64() * 20.0 // // Přidáme 0-20°C navíc
	}

	temp := s.baseTemp + fluctuation

	// Při určitých iteracích, by díky fluktuaci, teplotním skoku a baseTemp mohla teplota vystřelit nereálně vysoko:
	// max baseTemp: 30.0, max fluctuation: 25.0 -> 55.0°C - což je nerealistické
	// To stejné opačně:
	// min baseTemp: 15.0, min fluctuation: -5.0 -> 10.0, dost nízké na oceán, kde senzor měří
	// Proto omezíme temp na rozsah 15-45 (místo 10-55)
	if temp > 45.0 {
		temp = 45.0
	}
	if temp < 15.0 {
		temp = 15.0
	}

	// vytvoříme reading
	reading := Reading{
		SensorID:    s.ID,
		Temperature: math.Round(temp*100) / 100, // Zaokrouhlíme na 2 des. místa
		Timestamp:   time.Now().UTC(),
		Unit:        "celsius",
	}

	s.lastReading = &reading
	return reading
}

// IsWarning vrací true poud poslední čtení překročilo varovný práh 40°C
func (s *Sensor) IsWarning() bool {
	if s.lastReading == nil {
		return false
	}
	return s.lastReading.Temperature > 40
}
