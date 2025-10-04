package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CommandHistory представляет историю выполненных команд
type CommandHistory struct {
	Command string
	Result  string
}

// OllamaRequest структура для запроса к Ollama API
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaResponse структура для ответа от Ollama API
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Config структура конфигурации агента
type Config struct {
	OllamaURL  string
	Model      string
	Timeout    time.Duration
	MaxHistory int
}

// ShellAgent основная структура агента
type ShellAgent struct {
	commandHistory []CommandHistory
	config         Config
	client         *http.Client
}

// NewShellAgent создает новый экземпляр агента
func NewShellAgent() *ShellAgent {
	config := Config{
		OllamaURL:  "http://localhost:11434/api/generate",
		Model:      "qwen2.5-coder:3b",
		Timeout:    60 * time.Second,
		MaxHistory: 30,
	}
	
	return &ShellAgent{
		commandHistory: make([]CommandHistory, 0),
		config:         config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// naturalLanguageToCommand преобразует естественный язык в команду Linux
func (sa *ShellAgent) naturalLanguageToCommand(query string) (string, error) {
	// Формируем контекст из истории команд
	contextInfo := ""
	if len(sa.commandHistory) > 0 {
		contextInfo = "\n\nКонтекст предыдущих команд:\n"
		start := len(sa.commandHistory) - 3
		if start < 0 {
			start = 0
		}
		
		for i, cmd := range sa.commandHistory[start:] {
			contextInfo += fmt.Sprintf("%d. %s\n", i+1, cmd.Command)
		}
	}

	prompt := fmt.Sprintf(`Преобразуй следующий запрос на естественном языке в команду Linux.
Учитывай контекст предыдущих команд для понимания, о каких файлах или объектах идет речь.
Отвечай ТОЛЬКО командой, без объяснений и дополнительного текста.

Примеры:
- "покажи файлы" -> ls -la
- "кто я" -> whoami  
- "текущая папка" -> pwd
- "информация о системе" -> uname -a
- "свободная память" -> free -h
- "использование диска" -> df -h
- "запущенные процессы" -> ps aux
- "дата и время" -> date
- "содержимое файла test.txt" -> cat test.txt
- "найти файлы с расширением .py" -> find . -name "*.py"
- "создай файл example.txt" -> touch example.txt
- "покажи содержимое файла" (после создания example.txt) -> cat example.txt

%s

Запрос: %s

Команда:`, contextInfo, query)

	// Создаем запрос к Ollama
	reqBody := OllamaRequest{
		Model:  sa.config.Model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ошибка создания JSON: %v", err)
	}

	// Отправляем запрос
	resp, err := sa.client.Post(sa.config.OllamaURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка запроса к Ollama: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama вернул статус %d", resp.StatusCode)
	}

	// Парсим ответ
	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	command := strings.TrimSpace(ollamaResp.Response)
	
	// Убираем возможные лишние символы
	if strings.HasPrefix(command, "```") {
		lines := strings.Split(command, "\n")
		if len(lines) > 1 {
			command = lines[1]
		}
	}
	command = strings.ReplaceAll(command, "```", "")
	
	return strings.TrimSpace(command), nil
}

// runShell выполняет команду Linux
func (sa *ShellAgent) runShell(command string) string {
	if strings.TrimSpace(command) == "" {
		return "Пустая команда"
	}

	ctx, cancel := context.WithTimeout(context.Background(), sa.config.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	
	var output []string
	
	if stdout.Len() > 0 {
		output = append(output, strings.TrimSpace(stdout.String()))
	}
	
	if stderr.Len() > 0 {
		if err != nil {
			output = append(output, fmt.Sprintf("Ошибка: %s", strings.TrimSpace(stderr.String())))
		} else {
			output = append(output, fmt.Sprintf("Предупреждения: %s", strings.TrimSpace(stderr.String())))
		}
	}

	if len(output) > 0 {
		return strings.Join(output, "\n")
	}
	
	if err != nil {
		return fmt.Sprintf("Ошибка выполнения: %v", err)
	}
	
	return "Команда выполнена успешно (нет вывода)"
}

// showCommandHistory показывает историю команд
func (sa *ShellAgent) showCommandHistory() {
	if len(sa.commandHistory) == 0 {
		fmt.Println("\n📚 История команд пуста")
		return
	}

	fmt.Printf("\n📚 История команд (%d команд):\n", len(sa.commandHistory))
	fmt.Println(strings.Repeat("=", 60))

	for i, cmd := range sa.commandHistory {
		fmt.Printf("%d. %s\n", i+1, cmd.Command)
		
		// Показываем краткий результат (первые 100 символов)
		shortResult := cmd.Result
		if len(shortResult) > 100 {
			shortResult = shortResult[:100] + "..."
		}
		fmt.Printf("   Результат: %s\n", shortResult)
		fmt.Println(strings.Repeat("-", 40))
	}
}

// clearCommandHistory очищает историю команд
func (sa *ShellAgent) clearCommandHistory() {
	sa.commandHistory = make([]CommandHistory, 0)
	fmt.Println("\n🗑️ История команд очищена")
}

// intelligentShellAgent основная функция агента
func (sa *ShellAgent) intelligentShellAgent(query string) string {
	fmt.Printf("\n🔍 Обрабатываю запрос: %s\n", query)

	// Показываем контекст если есть
	if len(sa.commandHistory) > 0 {
		contextSize := len(sa.commandHistory)
		if contextSize > 3 {
			contextSize = 3
		}
		fmt.Printf("📚 Контекст: последние %d команд в памяти\n", contextSize)
	}

	// Преобразуем запрос в команду с учетом контекста
	command, err := sa.naturalLanguageToCommand(query)
	if err != nil {
		return fmt.Sprintf("Ошибка преобразования: %v", err)
	}

	fmt.Printf("\n💡 Предлагаемая команда: %s\n", command)

	// Спрашиваем разрешение
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n❓ Выполнить эту команду? (да/нет/d/n): ")
		permission, _ := reader.ReadString('\n')
		permission = strings.TrimSpace(strings.ToLower(permission))

		if permission == "да" || permission == "yes" || permission == "y" || permission == "d" || permission == "1" {
			fmt.Printf("\n⚡ Выполняю команду: %s\n", command)
			fmt.Println(strings.Repeat("-", 50))

			// Выполняем команду
			result := sa.runShell(command)

			// Добавляем в историю команд
			sa.commandHistory = append(sa.commandHistory, CommandHistory{
				Command: command,
				Result:  result,
			})

			// Ограничиваем историю согласно конфигурации
			if len(sa.commandHistory) > sa.config.MaxHistory {
				sa.commandHistory = sa.commandHistory[len(sa.commandHistory)-sa.config.MaxHistory:]
			}

			fmt.Printf("\n📋 Результат:\n%s\n", result)
			return result

		} else if permission == "нет" || permission == "no" || permission == "n" || permission == "0" {
			fmt.Println("\n❌ Выполнение отменено пользователем")
			return "Выполнение команды отменено пользователем"
		} else {
			fmt.Println("Пожалуйста, введите 'да' или 'нет' (d/n)")
		}
	}
}

// interactiveMode интерактивный режим работы с агентом
func (sa *ShellAgent) interactiveMode() {
	fmt.Println("🤖 Умный Linux Shell Агент")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Агент преобразует ваши запросы на естественном языке в команды Linux")
	fmt.Println("и спрашивает разрешение перед выполнением.")
	fmt.Println("\nВведите ваш запрос или 'exit' для выхода")
	fmt.Println("\nПримеры запросов:")
	fmt.Println("- 'покажи файлы в текущей папке'")
	fmt.Println("- 'создай файл test.txt'")
	fmt.Println("- 'покажи содержимое файла' (после создания файла)")
	fmt.Println("- 'кто я в системе'")
	fmt.Println("- 'информация о системе'")
	fmt.Println("- 'сколько свободной памяти'")
	fmt.Println("\nСпециальные команды:")
	fmt.Println("- 'история' - показать историю команд")
	fmt.Println("- 'очистить историю' - очистить контекст")
	fmt.Println(strings.Repeat("=", 60))

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\n👤 Ваш запрос: ")
		query, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("❌ Ошибка чтения ввода: %v\n", err)
			continue
		}

		query = strings.TrimSpace(query)

		if query == "" {
			fmt.Println("Пожалуйста, введите запрос")
			continue
		}

		lowerQuery := strings.ToLower(query)
		if lowerQuery == "exit" || lowerQuery == "quit" || lowerQuery == "выход" || lowerQuery == "выйти" {
			fmt.Println("👋 До свидания!")
			break
		}

		// Специальные команды для управления контекстом
		if lowerQuery == "история" || lowerQuery == "history" || lowerQuery == "контекст" {
			sa.showCommandHistory()
			continue
		} else if lowerQuery == "очистить историю" || lowerQuery == "clear history" || lowerQuery == "очистить контекст" {
			sa.clearCommandHistory()
			continue
		}

		sa.intelligentShellAgent(query)
	}
}



// main основная функция
func main() {
	agent := NewShellAgent()
	agent.interactiveMode()
}