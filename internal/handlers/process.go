package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"

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

func ProcessSticker(ctx context.Context, b *bot.Bot, rdb *redis.Client, userID int64, chatID int64) error {
	state, err := storage.LoadState(ctx, rdb, userID)

	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	if state.FileID == "" {
		return fmt.Errorf("no FileID in state")
	}

	f, err := b.GetFile(ctx, &bot.GetFileParams{FileID: state.FileID})
	if err != nil {
		return fmt.Errorf("get file: %w", err)
	}

	dl := b.FileDownloadLink(f)

	res, err := http.Get(dl)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}
	defer res.Body.Close()

	pr, pw := io.Pipe()
	go func(r io.Reader, w io.Writer) {
		defer pw.Close()
		if err := converter.ConvertSticker(ctx, res.Body, pw, *state); err != nil {
			fmt.Fprintf(os.Stderr, "convert error: %s", err)
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
		return err
	}

	_, err = b.SendDocument(ctx, &bot.SendDocumentParams{
		ChatID: chatID,
		Document: &models.InputFileUpload{
			Filename: filename + "." + ext,
			Data:     pr,
		},
	})
	if err != nil {
		return err
	}

	return err
}
