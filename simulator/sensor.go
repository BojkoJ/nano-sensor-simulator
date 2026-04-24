package simulator

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"os"
	"time"
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

// Config obsahuje konfiguraci senzoru
type Config struct {
	ID       string        // size: 16B
	MinTemp  float64       // size: 8B
	MaxTemp  float64       // size: 8B
	Interval time.Duration // size: 8B, reprezentuje uběhlý čas mezi ...
}

// sum size: 40B, při předávání Configu do funkcí KOPÍROVAT místo předávání pointeru:
// kopírování malého structu je RYCHLEJŠÍ než předání jako pointeru, pokud by struct byl >128B tak předávat jako pointer
// (nebo pokud by struct obsahoval Mutex lock - Mutexy se nesmí NIKDY kopírovat)

// Reading je jedno naměřené čtení senzory
type Reading struct {
	// `json:"..."` jsou struct tags, které říkají, jak pojmenovat pole v JSONu
	SensorID    string    `json:"sensor_id"`   // ID senzoru, který provedl měření
	Temperature float64   `json:"temperature"` // Naměřená teplota senzoru v tomto jeho jednom čtení
	Timestamp   time.Time `json:"timestamp"`   // Čas, kdy bylo měření provedeno
	Unit        string    `json:"unit"`        // Jednotka měření (jako celsius: °C, kelvin: K, Fahrenheit: °F)
	IsAnomaly   bool      `json:"is_anomaly"`  // Detekce, zda naměřená teplota je teplotní anomálie
}

// GenerateTemperature generuje náhodnou teplotu v rozsahu [min, max]
// s malým Gaussovým šumem pro realismus
// Exportováno pro unit testing.
func GenerateTemperature(min, max float64, rng *rand.Rand) float64 {
	center := (min + max) / 2 // Medián
	spread := (max - min) / 4 // Rozptyl
	// Gaussovo rozdělení kolem středu rozsahu
	raw := center + rng.NormFloat64()*spread
	// Clamping do rozsahu (oříznutí)
	return math.Max(min, math.Min(max, raw))
	// Vysvětlení:
	// 1) math.Min(max, raw) - clamp do horní hranice - bereme menší hodnotu z horní hranice a vygenerované teploty
	//        - Pokud je vygenerovaná teplota Větší než horní hranice tak místo ní vezmeme horní hranici - to je ten clamp
	// 2) math.Max(min, ...) - clamp do spodní hranice - bereme menší hodnotu z dolní hranice a vygenerované teploty
	//        - Pokud je vygenerovaná teplota Menší než spodní hranice tak místo ní vezmeme spodní hranici - to je zase clamp
}

// IsOutOfRange zkontroluje, zda teplota překračuje meze
// pokud překračuje, tak vrací TRUE (ano)
func IsOutOfRange(temp, min, max float64) bool {
	return temp < min || temp > max
}

// Sensor similuje IoT senzor teploty - teplotní čidlo
type Sensor struct {
	config Config       // Sensor si nese svoji konfiguraci
	rng    *rand.Rand   // také si nese svůj zdroj náhodných čísel (ať neicinializujeme zbytečně nový ve funkcích)
	logger *slog.Logger // také si nese svoji instanci loggeru
}

// NewSensor vytvoří nový senzor s danou konfigurací
func NewSensor(conf Config, logger *slog.Logger) *Sensor {
	return &Sensor{
		config: conf,
		// Každý senzor má vlastní RNG seed pro různé sekvence:
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
		logger: logger,
	}
}

// Run spustí senzor jako goroutinu, vytvoří reading, vygeneruje mu teplotu a pošle do out channelu.
// Zastaví se při zrušení contextu nebo uzavření out channelu.
// Rate-limiting: maximálne 1 zpráva za config.Interval
func (sensor *Sensor) Run(ctx context.Context, out chan<- Reading) {
	ticker := time.NewTicker(sensor.config.Interval)
	defer ticker.Stop()
	defer func() {
		if _, err := fmt.Fprintf(os.Stdout, "[%s] sensor stopped\n", sensor.config.ID); err != nil {
			sensor.logger.Error("failed to write \"sensor stopped\" log", "error", err)
		}
	}()

	// Loopujeme dokud to manuálně nezastavíme
	for {
		select {
		case <-ticker.C: // Ticker tiknul, z kanálu ticker.C (kde se ukládají data o tikání) jsme úspěšně přečetli data
			temperature := GenerateTemperature(sensor.config.MinTemp, sensor.config.MaxTemp, sensor.rng)
			reading := Reading{
				SensorID:    sensor.config.ID,
				Temperature: temperature,
				Timestamp:   time.Now(),
				Unit:        "C",
				IsAnomaly:   IsOutOfRange(temperature, sensor.config.MinTemp, sensor.config.MaxTemp),
			}

			// Non-blocking send - drop pokud je buffer plný
			select {
			case out <- reading: // pokud output buffer není plný, dáme do něj reading
			default:
				if _, err := fmt.Fprintf(os.Stderr, "[%s] WARNING: output buffer full, current reading dropped...\n", sensor.config.ID); err != nil {
					sensor.logger.Error("failed to write \"output buffer full WARNING\" log", "error", err)
				}
			}
		case <-ctx.Done():
			// Ve chvíli když se někde v main zavolá cancel() (nebo vyprší časový limit), tento kanál "ctx" se uzavře
			//     1. Jakmile se kanál uzavře, select to zaregistruje - tento case
			//     2. Provede se tato větev case <-ctx.Done()
			//     3. Příkaz return ukončí funkci Run()
			//     4. Díky tomu, že funkce končí, spustí se defer bloku (zastaví se ticker a vypíše se log o zastavení)
			return
		}

	}
}
