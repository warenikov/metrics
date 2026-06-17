# Имена бинарников
SERVER_BIN=cmd/server/server
AGENT_BIN=cmd/agent/agent
TEST_BIN=autotest
FILE_STORAGE=storage.txt

# Номер инкретемента для тестов через бинарник (можно переопределить: make test_ya NUM=7)
NUM ?= 1
make test_ya NUM=12 DATABASE_DSN="postgres://userusernamename:password@localhost:5432/dbname?sslmode=disable" ?=

# Пути к исходникам
SERVER_SRC=./cmd/server
AGENT_SRC=./cmd/agent

.PHONY: build test_ya test_local clean cover_html

#Сборка обоих бинарников
build:
	go build -o $(SERVER_BIN) $(SERVER_SRC)
	go build -o $(AGENT_BIN) $(AGENT_SRC)

#Запуск автотестов от YANDEX (сначала вызывается build) - можно передать флаг clear для удаления бинарников приложения после тестов
test_ya: build
	$(TEST_BIN) -test.v -test.run=^TestIteration$(NUM)$$ -binary-path=$(SERVER_BIN) \
		-agent-binary-path=$(AGENT_BIN) -server-port=8080 -source-path=./ \
		-file-storage-path=$(FILE_STORAGE) \
		$(if $(DATABASE_DSN),-database-dsn=$(DATABASE_DSN)) \
	$(if $(KEY),-key=$(KEY))

	$(if $(filter --clean,$(MAKECMDGOALS)), $(MAKE) clean)

#запуск локальных тестов при помощи встроенных в go инструментов
test_local:
	go test -v -race $(if $(filter cover,$(MAKECMDGOALS)),-cover) ./...

# Создает HTML-отчет по покрытию кода тестами и открывает его в браузере
cover_html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
	rm coverage.out

# Запуск бенчмарков по всем пакетам
bench:
	go test -bench=. -benchmem ./internal/...

# Профилирование памяти: base (до оптимизаций) и result (после)
bench_base:
	mkdir -p profiles
	go test -bench=. -benchmem -memprofile=profiles/base.pprof ./internal/middleware/

bench_result:
	mkdir -p profiles
	go test -bench=. -benchmem -memprofile=profiles/result.pprof ./internal/middleware/

# Сравнение профилей до и после оптимизации
bench_diff:
	go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof

#Очистка бинарников
clean:
	rm -f $(SERVER_BIN) $(AGENT_BIN)
