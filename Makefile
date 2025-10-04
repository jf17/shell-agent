# Переменные
BINARY_NAME=shell-agent
GO_FILES=$(shell find . -name "*.go" -type f)
INSTALL_PATH=/usr/local/bin

# Основные команды
.PHONY: build clean run test help install uninstall

# Сборка проекта
build:
	go build -o $(BINARY_NAME) .

# Запуск проекта
run: build
	./$(BINARY_NAME)

# Очистка собранных файлов
clean:
	go clean
	rm -f $(BINARY_NAME)

# Запуск тестов
test:
	go test ./...

# Форматирование кода
fmt:
	go fmt ./...

# Проверка кода
vet:
	go vet ./...

# Установка зависимостей
deps:
	go mod tidy
	go mod download

# Сборка для разных платформ
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux .

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows.exe .

build-mac:
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-mac .

# Сборка для всех платформ
build-all: build-linux build-windows build-mac

# Установка в систему
install: build
	@echo "Установка $(BINARY_NAME) в $(INSTALL_PATH)..."
	sudo cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "$(BINARY_NAME) успешно установлен!"
	@echo "Теперь можно запускать командой: $(BINARY_NAME)"

# Удаление из системы
uninstall:
	@echo "Удаление $(BINARY_NAME) из $(INSTALL_PATH)..."
	sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "$(BINARY_NAME) успешно удален!"

# Помощь
help:
	@echo "Доступные команды:"
	@echo "  build       - Собрать проект"
	@echo "  run         - Собрать и запустить проект"
	@echo "  clean       - Очистить собранные файлы"
	@echo "  test        - Запустить тесты"
	@echo "  fmt         - Форматировать код"
	@echo "  vet         - Проверить код"
	@echo "  deps        - Установить зависимости"
	@echo "  build-all   - Собрать для всех платформ"
	@echo "  install     - Установить в систему (/usr/local/bin)"
	@echo "  uninstall   - Удалить из системы"
	@echo "  help        - Показать эту справку"