package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
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

func main() {
	// Inicializujeme 3 senzory
	sensors := []*Sensor{
		NewSensor("sensor-1"),
		NewSensor("sensor-2"),
		NewSensor("sensor-3"),
	}

	_, err := fmt.Fprintln(os.Stderr, "Nano Sensor Simulator Starting...")
	if err != nil {
		log.Fatal(err)
		return
	}
	_, err = fmt.Fprintf(os.Stderr, "Monitoring %d sensors. Press Ctrl+C to stop.\n", len(sensors))
	if err != nil {
		log.Fatal(err)
		return
	}

	// Ticker - "tiká" každou sekundu (analogie setInterval v JS)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Channel pro OS signály (Ctrl+C = SIGINT, kill = SIGTERM)
	// Vytvoříme kanál (channel) na signály - 1. parametr
	// 2. parametr: 1 znamená, že kanál si může zapamatovat 1 signál,
	// Protože chceme gracefully ukončit, což znamená:
	// - zachytit signál SIGINT (zkratka pro "interrupt", což je signál, který se posílá při stisknutí Ctrl+C)
	// - zachytit signál SIGTERM (zkratka pro "terminate", což je signál, který se posílá při ukončování procesu, například příkazem kill)
	// Takže "graceful shutdown" znamená, že když uživatel stiskne Ctrl+C nebo když se proces ukončuje, naše aplikace zachytí tento signál
	// a může provést nějaké úklidové operace (jako zavření souborů, uvolnění zdrojů, atd.) před tím, než se skutečně ukončí.
	sigChan := make(chan os.Signal, 1)

	// signal.Notify říká, že chceme dostávat notifikace o signálech SIGINT a SIGTERM do kanálu sigChan a ostatní signály tento kanál ignoruje
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// JSON encoder pro výstup do stdout
	encoder := json.NewEncoder(os.Stdout)
	// prefix: "" - nechceme žádný prefix před každým JSON objektem
	// indent: " " - chceme, aby JSON byl hezky formátovaný s odsazením (pretty print)
	encoder.SetIndent("", " ")

	// Hlavní smyčka - select čeká na první připravený channel
	for {
		select { // Select je jako Switch
		case <-ticker.C: // Pokud tiknul ticker (každou sekundu), tak se provede tento blok
			// Ticker "tiknul" - zpracuj všechny seznory
			for _, sensor := range sensors {
				reading := sensor.GenerateReading()

				// Zapíšeme JSON na stdout
				// funkce Encode zapíše JSON reprezentaci objektu reading do writeru (v našem případě os.Stdout)
				// automaticky na stdout, protože json.NewEncoder(os.Stdout)
				if err := encoder.Encode(reading); err != nil {
					_, err := fmt.Fprintf(os.Stderr, "ERROR: encode reading: %v\n", err)
					if err != nil {
						log.Fatal(err)
						return
					}
					continue
				}

				if sensor.IsWarning() {
					_, err = fmt.Fprintf(os.Stderr,
						"WARNNING: %s temperature %.2f°C exceeds threshold 40°C\n",
						reading.SensorID, reading.Temperature)
					if err != nil {
						log.Fatal(err)
						return
					}
				}
			}
		case sig := <-sigChan: // Pokud do kanálu sigChan přišel signál (například Ctrl+C), který kanál zachycuje (zachycuje jen SIGINT a SIGTERM), tak se provede tento blok
			_, err = fmt.Fprintf(os.Stderr, "\nReceived Signal: %v. Shutting down...\n", sig)
			return // ukončí main, defer ticker.Stop() se zavolá
		}
	}
}
