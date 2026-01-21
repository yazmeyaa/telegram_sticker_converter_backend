package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"time"

	"github.com/go-telegram/bot"
	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/handlers"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/logger"
)

// debugTransport обёртка для отладки HTTP запросов
type debugTransport struct {
	transport http.RoundTripper
}

func (d *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Логируем запрос
	reqDump, _ := httputil.DumpRequestOut(req, true)
	fmt.Printf("=== REQUEST ===\n%s\n", string(reqDump))

	// Выполняем запрос
	resp, err := d.transport.RoundTrip(req)
	if err != nil {
		fmt.Printf("=== ERROR ===\n%v\n", err)
		return nil, err
	}

	// Читаем тело ответа для логирования
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	fmt.Printf("=== RESPONSE ===\nStatus: %s\nContent-Length: %d\nBody: %s\n", resp.Status, resp.ContentLength, string(bodyBytes))

	// Восстанавливаем тело ответа
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	return resp, nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Инициализация логгера
	appLogger := logger.New("telegram_bot", []io.Writer{os.Stdout})

	rdb := redis.NewClient(&redis.Options{})
	if err := rdb.Ping(ctx).Err(); err != nil {
		appLogger.Error("Redis connection failed", "error", err)
		log.Fatal("Redis connection failed:", err)
	}

	appLogger.Info("Redis connection established")

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		appLogger.Error("BOT_TOKEN not set")
		log.Fatal("BOT_TOKEN not set")
	}

	// Определение URL Telegram API
	telegramAPIURL := os.Getenv("TELEGRAM_API_URL")
	if telegramAPIURL == "" {
		telegramAPIURL = "https://api.telegram.org"
	}

	// Настройка HTTP клиента с увеличенными таймаутами для больших файлов
	httpClient := &http.Client{
		Timeout: 10 * time.Minute, // 10 минут для загрузки больших файлов
		Transport: &debugTransport{
			transport: &http.Transport{
				ResponseHeaderTimeout: 5 * time.Minute,
				TLSHandshakeTimeout:   30 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		},
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(handlers.MessageHandler(rdb, appLogger)),
		bot.WithCallbackQueryDataHandler("action=format_select", bot.MatchTypePrefix, handlers.FormatSelectHandler(rdb, appLogger)),
		bot.WithCallbackQueryDataHandler("action=frame_select", bot.MatchTypePrefix, handlers.FrameSelectHandler(rdb, appLogger)),
		bot.WithCallbackQueryDataHandler("action=size_select", bot.MatchTypePrefix, handlers.SizeSelectHandler(rdb, appLogger)),
		bot.WithServerURL(telegramAPIURL),
		bot.WithHTTPClient(30*time.Second, httpClient), // pollTimeout 30 секунд, HTTP клиент с увеличенными таймаутами
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		appLogger.Error("Failed to create bot", "error", err)
		log.Fatal(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, handlers.HandleStartCommand(appLogger))

	appLogger.Info("Telegram bot started", "telegram_api_url", telegramAPIURL)
	b.Start(ctx)
}
