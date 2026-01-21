package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
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

func ProcessSticker(ctx context.Context, bot *telego.Bot, rdb *redis.Client, userID int64, chatID int64, logger Logger) error {
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

	f, err := bot.GetFile(ctx, &telego.GetFileParams{FileID: state.FileID})
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

	// Получаем reader для файла
	// При локальном Bot API file_path может быть абсолютным путём к файлу
	var fileReader io.ReadCloser
	if strings.HasPrefix(f.FilePath, "/") {
		// Локальный Bot API - читаем файл напрямую с диска
		logger.Info("Reading file from local path",
			"user_id", userID,
			"chat_id", chatID,
			"file_path", f.FilePath,
			"action", "process_sticker",
		)
		localFile, err := os.Open(f.FilePath)
		if err != nil {
			logger.Error("Failed to open local file",
				"user_id", userID,
				"chat_id", chatID,
				"file_path", f.FilePath,
				"error", err,
				"action", "process_sticker",
			)
			return fmt.Errorf("open local file: %w", err)
		}
		fileReader = localFile
	} else {
		// Обычный Bot API - скачиваем по HTTP
		dl := bot.FileDownloadURL(f.FilePath)
		logger.Info("Downloading file via HTTP",
			"user_id", userID,
			"chat_id", chatID,
			"download_link", dl,
			"action", "process_sticker",
		)

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
		fileReader = res.Body
	}
	defer fileReader.Close()

	logger.Info("File ready, starting conversion",
		"user_id", userID,
		"chat_id", chatID,
		"file_id", state.FileID,
		"action", "process_sticker",
	)

	// Создаем временный файл для результата
	tmpFile, err := os.CreateTemp("", "sticker_result_*")
	if err != nil {
		logger.Error("Failed to create temp file",
			"user_id", userID,
			"chat_id", chatID,
			"error", err,
			"action", "process_sticker",
		)
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpFileName := tmpFile.Name()
	defer os.Remove(tmpFileName)
	defer tmpFile.Close()

	if err := converter.ConvertSticker(ctx, fileReader, tmpFile, *state); err != nil {
		logger.Error("Conversion error",
			"user_id", userID,
			"chat_id", chatID,
			"file_id", state.FileID,
			"input_format", state.InputFormat,
			"output_format", state.OutputFormat,
			"error", err,
			"action", "process_sticker",
		)
		return fmt.Errorf("convert sticker: %w", err)
	}

	// Закрываем файл и открываем заново для чтения
	tmpFile.Close()
	tmpFile, err = os.Open(tmpFileName)
	if err != nil {
		return fmt.Errorf("reopen temp file: %w", err)
	}
	defer tmpFile.Close()

	// Получаем размер файла
	stat, err := tmpFile.Stat()
	if err != nil {
		return fmt.Errorf("stat temp file: %w", err)
	}

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
		"file_size", stat.Size(),
		"action", "process_sticker",
	)

	// Читаем файл в память для отправки
	data, err := io.ReadAll(tmpFile)
	if err != nil {
		return fmt.Errorf("read temp file: %w", err)
	}

	doc := tu.Document(tu.ID(chatID), telego.InputFile{
		File: tu.NameReader(
			&bytesReaderCloser{data: data},
			filename+"."+ext,
		),
	})

	_, err = bot.SendDocument(ctx, doc)
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

// bytesReaderCloser реализует io.ReadCloser для []byte
type bytesReaderCloser struct {
	data   []byte
	offset int
}

func (b *bytesReaderCloser) Read(p []byte) (n int, err error) {
	if b.offset >= len(b.data) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.offset:])
	b.offset += n
	return n, nil
}

func (b *bytesReaderCloser) Close() error {
	return nil
}
