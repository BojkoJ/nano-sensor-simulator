# Stage 1: Builder
# Používáme oficální Go image s Alpine jako base pro build stage.
# Verze Go musí přesně odpovídat verzi v go.mod.
# "AS builder" pojmenuje stage pro pozdějí COPY --from=builder
FROM golang:1.25-alpine AS builder

# Nainstalujeme ca-certificates pro HTTPS volání a git pro go get
# Děláme to v jednom RUN příkazu = jedna vrstva. Čistíme cache hned po instalaci.
RUN apk add --no-cache ca-certificates git

# Nastavíme pracovní adresář uvnitř kontejneru.
WORKDIR /build

# Důležité: Nejprve zkopírujeme POUZE go.mod a go.sum a stáhneme závislosti.
# Proč: Závislosti se mění velmí zřídka. Docker tuto vrstu cachuje.
# Pokud změníme jen kód aplikace, Docker nemusí znovu stahovat závislosti.
# Kdybychom zkopírovali všechny najednou (COPY . .), cache by se invalidovala při každé změně souboru
COPY go.mod ./
RUN go mod download

# Teprve teď zkopírujeme zbytek zdrojového kódu.
# Tato vrstva se invaliduje při každé změně kódu - ale závislosti jsou stále v cache.
COPY . .

# Kompilace binárky.
# CGO_ENABLED=0    - zakáže CGO, výsledek je statická binárka bez závislosti na libc.
#                    NUTNÉ pro FROM scratch a FROM distroless/static
# GOOS=linux       - cross-compile pro linux (důležité pro buildu na macOS/Windows)
# GOARCH=amd64     - target architektura (pro ARM servery použijeme arm64)
# -ldflags="-w -s" - odstraní lokální cesty ze stack traces (bezpečnost + reproducibility)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -trimpath \
    -o /build/simulator \
    ./main.go

# Stage 2: Runtime
# Začínáme znovu od nuly s minimálním base image.
# distroless/static obsahuje: SSL certifikáty, timezone data, /etc/password
# Neobsahuje: shell, package manager, žádné systémové utility
FROM gcr.io/distroless/static-debian12

# Zkopírujeme SSL certifikáty z builder stage (záložní - distroless je již má)
# Pokud používáme FROM scratch, tento řádek je NUTNÝ pro HTTPS volání:
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs

# Zkopírujeme POUZE zkompilovanou binárky z builder stage.
# Z gigabajtového builder image vezmeme jen ~8MB binárku.
COPY --from=builder /build/simulator /simulator

# Nastavíme enviroment variables pro produkci.
# Tyto hodnoty budou výchozí; lze přepsat při docker run -e nebo v Compose.
ENV APP_ENV=production \
    LOG_FORMAT=json

# Dokumentujeme port, na kterém aplikace naslouchá.
# EXPOSE pouze dokumentuje - skutečné mapování portů je při docker run -p
EXPOSE 8080

# Health check - Docker bude pravidelně volat tento příkaz.
# Pokud příkaz selže interval-krát za sebou, kontejner je označen jako "unhealthy".
# Pro distroless/scratch potřebujeme buď vlastní health endpoint nebo wget ze statfs.
# Zde předpokládáme, že simulátor má HTTP /health endpoint.
# HEALTHCHECK --intervals=30s --timeout=5s --start-period=10s --retries=3 \
#     CMD ["/simulator", "--health-check"]

# Bezpečnost: spouštíme jako non-root uživatel.
# distroless/static má předpřipraveného uživatele "nonroot" (UID 65532).
# Pro FROM scratch bychom museli zkopírovat /etc/passwd a /etc/group z builder stage.
USER nonroot:nonroot

# language= Hlavní příkaz kontejneru.
# Exec forma (JSON array) zajistí, že /simulator bude PID 1 a dostane SIGTERM správně.
ENTRYPOINT ["/simulator"]


# Sestavení image:
# Základní build - tečka na konci je build context (aktuální adresář)
# docker build -t simulator:latest .

# S kontrétním tagem (správná praxe)
# docker build -t simulator:1.0.0 .

# S více tagy najednou
# docker build -t simulator:1.0.0 -t simulator:latest .

# Specifický Dockerfile (pokud není v root adresáři)
# docker build -f docker/simulator.Dockerfile -t simulator:1.0.0 .

# Build s build argumentem (pro ARG instrukce v Dockerfile)
# docker build --build-arg VERSION=1.0.0 -t simulator:1.0.0 .

# Build bez cachce (pokud potřebujeme čistý build)
# docker build --no-cache -t simulator:1.0.0 .

# Spuštění kontejneru:
# docker run simulator:1.0.0

# --rm: automaticky smaže kontejner po zastavení (ideální pro vývoj)
# docker run --rm simulator:1.0.0

# -d: spustí na pozadí (detached mode)
# docker run -d --name simulator simulator:1.0.0

# -p: mapování portů HOST_PORT:CONTAINER_PORT
# docker run -p 8080:8080 simulator:1.0.0

# -e: nastavení enviroment variables
# docker run -e NATS_URL=nats://localhost:4222 -e LOG_LEVEL=debug simulator:1.0.0

# -v: připojení volume HOST_PATH:CONTAINER_PATH
# docker run -v /tmp/data:/data simulator:1.0.0
# nebo named volume:
# docker run -v simulator-data:/data simulator:1.0.0

# Kombinace všeho pro lokální vývoj:
# docker run --rm \
# --name simulator \
# -p 8080:8080 \
# -e NATS_URL=nats://host.docker.internal:4222 \
# -v "$(pwd)/config:/etc/simulator" \
# simulator:1.0.0


# Tento spouštět: docker build -f backend/Dockerfile -t sts-simulator:dev .
# pak: docker images sts-simulator:dev