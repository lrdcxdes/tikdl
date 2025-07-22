// main.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"tikdl-web/enums"
	"tikdl-web/ext/tiktok" // Убедитесь, что путь импорта верный

	"tikdl-web/models"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// --- Глобальные переменные и структуры ---

// DownloadJob представляет задачу на скачивание
type DownloadJob struct {
	ID         string
	URL        string
	WantsVideo bool
	WantsAudio bool
	WantsHQ    bool
	Client     *Client // Ссылка на клиента для отправки результата
}

// DownloadResult представляет результат обработки
type DownloadResult struct {
	JobID      string `json:"jobId"`
	RequestURL string `json:"requestUrl"`
	Title      string `json:"title,omitempty"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	VideoURL   string `json:"videoUrl,omitempty"`
	AudioURL   string `json:"audioUrl,omitempty"`
}

// JobRequest представляет запрос от клиента
type JobRequest struct {
	URL        string `json:"url"`
	WantsVideo bool   `json:"video"`
	WantsAudio bool   `json:"audio"`
	WantsHQ    bool   `json:"hq"`
}

var (
	// Очередь задач
	jobQueue = make(chan DownloadJob, 100)
	// Для обновления HTTP до WebSocket
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Разрешаем все подключения для простоты
			return true
		},
	}
)

// --- Логика воркеров ---

// processTikTokURL - это обертка над вашим модулем
func processTikTokURL(ctx *models.DownloadContext) ([]*models.Media, error) {
	// Сначала пробуем через API
	mediaList, err := tiktok.MediaListFromAPI(ctx)
	if err == nil {
		log.Println("Successfully fetched media from API for:", ctx.MatchedContentURL)
		return mediaList, nil
	}
	log.Println("API method failed, falling back to Web. Error:", err)

	// Если API не сработал, пробуем через Web
	// mediaList, err = tiktok.MediaListFromWeb(ctx)
	// if err != nil {
	// 	log.Println("Web method also failed for:", ctx.MatchedContentURL, "Error:", err)
	// 	return nil, tiktok.ErrAllMethodsFailed
	// }

	// log.Println("Successfully fetched media from Web for:", ctx.MatchedContentURL)
	return mediaList, nil
}

// worker - функция, которая будет выполняться в отдельной горутине
func worker(id int, jobs <-chan DownloadJob) {
	for job := range jobs {
		log.Printf("Worker %d started job %s for URL: %s", id, job.ID, job.URL)

		// Создаем контекст для вашего модуля
		// Для этого нужно извлечь ID из URL с помощью регекспа из вашего модуля
		matches := tiktok.Extractor.URLPattern.FindStringSubmatch(job.URL)
		if len(matches) == 0 {
			// Добавляем RequestURL в ответ с ошибкой
			result := DownloadResult{JobID: job.ID, RequestURL: job.URL, Status: "error", Message: "Invalid TikTok URL"}
			job.Client.send <- result
			continue
		}

		idIndex := tiktok.Extractor.URLPattern.SubexpIndex("id")
		contentID := matches[idIndex]

		ctx := &models.DownloadContext{
			Extractor:         tiktok.Extractor,
			MatchedContentID:  contentID,
			MatchedContentURL: job.URL,
		}

		mediaList, err := processTikTokURL(ctx)
		if err != nil {
			result := DownloadResult{JobID: job.ID, Status: "error", Message: "Failed to fetch video data: " + err.Error()}
			job.Client.send <- result
			continue
		}

		media := mediaList[0] // Работаем с первым элементом
		var videoURL, audioURL string

		// Ищем нужные форматы
		for _, format := range media.Formats {
			if format.Type == enums.MediaTypeVideo && videoURL == "" {
				if len(format.URL) > 0 {
					videoURL = format.URL[0]
				}
			}
			if format.Type == enums.MediaTypeAudio && audioURL == "" {
				if len(format.URL) > 0 {
					audioURL = format.URL[0]
				}
			}
		}

		if videoURL == "" && audioURL == "" {
			result := DownloadResult{JobID: job.ID, RequestURL: job.URL, Status: "error", Message: "No downloadable links found."}
			job.Client.send <- result
			continue
		}

		if videoURL == "" {
			result := DownloadResult{JobID: job.ID, Status: "error", Message: "No downloadable links found."}
			job.Client.send <- result
			continue
		}

		result := DownloadResult{
			JobID:      job.ID,
			RequestURL: job.URL,
			Title:      media.Caption.String, // <--- ADD THIS LINE
			Status:     "completed",
			VideoURL:   videoURL,
			AudioURL:   audioURL,
		}
		job.Client.send <- result
	}
}

// --- Управление WebSocket клиентами ---

// Client - представляет одного WebSocket клиента
type Client struct {
	conn *websocket.Conn
	send chan DownloadResult
	mu   sync.Mutex
}

// readPump читает сообщения от клиента
func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		var req JobRequest
		if err := json.Unmarshal(message, &req); err != nil {
			log.Println("Invalid JSON received:", err)
			continue
		}

		jobID := uuid.New().String()

		// Отправляем задачу в очередь
		jobQueue <- DownloadJob{
			ID:         jobID,
			URL:        req.URL,
			WantsVideo: req.WantsVideo,
			WantsAudio: req.WantsAudio,
			WantsHQ:    req.WantsHQ,
			Client:     c,
		}

		// Отправляем подтверждение клиенту
		statusUpdate := DownloadResult{
			JobID:      jobID,
			RequestURL: req.URL, // <--- И ЗДЕСЬ
			Status:     "queued",
			Message:    "Your request is in the queue.",
		}
		c.send <- statusUpdate
	}
}

// writePump отправляет сообщения клиенту
func (c *Client) writePump() {
	defer c.conn.Close()
	for result := range c.send {
		c.mu.Lock()
		err := c.conn.WriteJSON(result)
		c.mu.Unlock()
		if err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}

// handleConnections - обработчик WebSocket подключений
func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan DownloadResult, 10),
	}

	go client.readPump()
	go client.writePump()

	log.Println("Client connected")
}

// --- Точка входа ---

func main() {
	// Запускаем воркеров
	const numWorkers = 4
	for i := 1; i <= numWorkers; i++ {
		go worker(i, jobQueue)
	}
	log.Printf("Started %d workers", numWorkers)

	// Сервер для статики (html, css)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Обработчик WebSocket
	http.HandleFunc("/ws", handleConnections)

	log.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
