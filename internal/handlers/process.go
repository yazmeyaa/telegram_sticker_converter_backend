package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/converter"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/storage"
)

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:n], nil
}

func ProcessSticker(ctx context.Context, b *bot.Bot, rdb *redis.Client, userID int64, chatID int64, logger *slog.Logger) error {
	logger.Info("Processing sticker started",
		"user_id", userID,
		"chat_id", chatID,
		"action", "process_sticker",
	)

	state, err := storage.LoadState(ctx, rdb, userID)
	if err != nil {
		logger.Error("Failed to load state",
			"user_id", userID,
			"chat_id", chatID,
			"error", err,
			"action", "process_sticker",
		)
		return fmt.Errorf("load state: %w", err)
	}
	if state.FileID == "" {
		logger.Error("No FileID in state",
			"user_id", userID,
			"chat_id", chatID,
			"action", "process_sticker",
		)
		return fmt.Errorf("no FileID in state")
	}

	logger.Info("State loaded",
		"user_id", userID,
		"chat_id", chatID,
		"file_id", state.FileID,
		"input_format", state.InputFormat,
		"output_format", state.OutputFormat,
		"frame_option", state.FrameOption,
		"size", state.Size,
		"action", "process_sticker",
	)

	f, err := b.GetFile(ctx, &bot.GetFileParams{FileID: state.FileID})
	if err != nil {
		logger.Error("Failed to get file from Telegram",
			"user_id", userID,
			"chat_id", chatID,
			"file_id", state.FileID,
			"error", err,
			"action", "process_sticker",
		)
		return fmt.Errorf("get file: %w", err)
	}

	logger.Info("File retrieved from Telegram",
		"user_id", userID,
		"chat_id", chatID,
		"file_id", state.FileID,
		"file_path", f.FilePath,
		"file_size", f.FileSize,
		"action", "process_sticker",
	)

	dl := b.FileDownloadLink(f)

	// HTTP клиент с таймаутом для загрузки файла
	httpClient := &http.Client{
		Timeout: 5 * time.Minute,
	}

	res, err := httpClient.Get(dl)
	if err != nil {
		logger.Error("Failed to download file",
			"user_id", userID,
			"chat_id", chatID,
			"file_id", state.FileID,
			"download_link", dl,
			"error", err,
			"action", "process_sticker",
		)
		return fmt.Errorf("download file: %w", err)
	}
	defer res.Body.Close()

	logger.Info("File downloaded, starting conversion",
		"user_id", userID,
		"chat_id", chatID,
		"file_id", state.FileID,
		"action", "process_sticker",
	)

	pr, pw := io.Pipe()
	go func(r io.Reader, w io.Writer) {
		defer pw.Close()
		if err := converter.ConvertSticker(ctx, res.Body, pw, *state); err != nil {
			logger.Error("Conversion error",
				"user_id", userID,
				"chat_id", chatID,
				"file_id", state.FileID,
				"input_format", state.InputFormat,
				"output_format", state.OutputFormat,
				"error", err,
				"action", "process_sticker",
			)
			pw.CloseWithError(err)
		}
	}(res.Body, pw)

	var ext string
	if (state.OutputFormat == "png" || state.OutputFormat == "jpeg" || state.OutputFormat == "webp") &&
		(state.FrameOption == "all" || state.FrameOption == "range") {
		ext = "zip"
	} else if state.OutputFormat == "lottie" {
		ext = "json"
	} else {
		ext = state.OutputFormat
	}

	filename, err := randomString(16)
	if err != nil {
		logger.Error("Failed to generate random filename",
			"user_id", userID,
			"chat_id", chatID,
			"error", err,
			"action", "process_sticker",
		)
		return err
	}

	logger.Info("Sending converted file",
		"user_id", userID,
		"chat_id", chatID,
		"file_id", state.FileID,
		"filename", filename+"."+ext,
		"action", "process_sticker",
	)

	// Создаем контекст с увеличенным таймаутом для отправки больших файлов
	// Особенно важно для "all frames" + "original size"
	sendCtx := ctx
	isLargeFile := (state.FrameOption == "all" || state.FrameOption == "range") && state.Size == "original"
	if isLargeFile {
		// Для больших файлов (all frames + original size) увеличиваем таймаут до 15 минут
		var sendCancel context.CancelFunc
		sendCtx, sendCancel = context.WithTimeout(ctx, 15*time.Minute)
		defer sendCancel()
		logger.Info("Using extended timeout for large file",
			"user_id", userID,
			"chat_id", chatID,
			"frame_option", state.FrameOption,
			"size", state.Size,
			"action", "process_sticker",
		)
	} else {
		// Для обычных файлов используем таймаут 5 минут
		var sendCancel context.CancelFunc
		sendCtx, sendCancel = context.WithTimeout(ctx, 5*time.Minute)
		defer sendCancel()
	}

	_, err = b.SendDocument(sendCtx, &bot.SendDocumentParams{
		ChatID: chatID,
		Document: &models.InputFileUpload{
			Filename: filename + "." + ext,
			Data:     pr,
		},
	})
	if err != nil {
		logger.Error("Failed to send document",
			"user_id", userID,
			"chat_id", chatID,
			"file_id", state.FileID,
			"filename", filename+"."+ext,
			"error", err,
			"action", "process_sticker",
		)
		return err
	}

	logger.Info("Sticker processing completed successfully",
		"user_id", userID,
		"chat_id", chatID,
		"file_id", state.FileID,
		"input_format", state.InputFormat,
		"output_format", state.OutputFormat,
		"filename", filename+"."+ext,
		"action", "process_sticker",
	)

	return nil
}
