package handlers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/storage"
)

func MessageHandler(rdb *redis.Client, logger Logger) th.Handler {
	return func(c *th.Context, update telego.Update) error {
		msg := update.Message
		if msg == nil {
			return nil
		}

		bot := c.Bot()
		ctx := c.Context()

		userID := msg.From.ID
		chatID := msg.Chat.ID
		username := msg.From.Username

		var stickerFileID string
		var isAnimated, isVideo bool

		if msg.Sticker != nil {
			sticker := msg.Sticker
			stickerFileID = sticker.FileID
			isAnimated = sticker.IsAnimated
			isVideo = sticker.IsVideo

			logger.Info("Received sticker",
				"user_id", userID,
				"chat_id", chatID,
				"username", username,
				"sticker_file_id", stickerFileID,
				"sticker_is_animated", isAnimated,
				"sticker_is_video", isVideo,
				"action", "sticker_received",
			)
		} else if msg.Entities != nil {
			// Проверяем custom emoji
			for _, e := range msg.Entities {
				if e.Type != telego.EntityTypeCustomEmoji || e.CustomEmojiID == "" {
					continue
				}

				logger.Info("Received custom emoji",
					"user_id", userID,
					"chat_id", chatID,
					"username", username,
					"custom_emoji_id", e.CustomEmojiID,
					"action", "custom_emoji_received",
				)

				// Получаем стикер через API
				stickers, err := bot.GetCustomEmojiStickers(ctx, &telego.GetCustomEmojiStickersParams{
					CustomEmojiIDs: []string{e.CustomEmojiID},
				})
				if err != nil {
					logger.Error("Failed to get custom emoji stickers",
						"user_id", userID,
						"chat_id", chatID,
						"custom_emoji_id", e.CustomEmojiID,
						"error", err,
						"action", "custom_emoji_received",
					)
					return err
				}
				if len(stickers) < 1 {
					logger.Warn("No stickers returned for custom emoji",
						"user_id", userID,
						"chat_id", chatID,
						"custom_emoji_id", e.CustomEmojiID,
						"action", "custom_emoji_received",
					)
					return nil
				}
				sticker := stickers[0]
				stickerFileID = sticker.FileID
				isAnimated = sticker.IsAnimated
				isVideo = sticker.IsVideo

				logger.Info("Custom emoji sticker retrieved",
					"user_id", userID,
					"chat_id", chatID,
					"sticker_file_id", stickerFileID,
					"sticker_is_animated", isAnimated,
					"sticker_is_video", isVideo,
					"action", "custom_emoji_received",
				)
				break
			}
		}

		if stickerFileID == "" {
			return nil
		}

		var inf string
		var kb *telego.InlineKeyboardMarkup

		switch {
		case isAnimated:
			inf = "tgs"
			kb = tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("GIF").WithCallbackData(formatToKey(inf, "gif")),
					tu.InlineKeyboardButton("WEBM").WithCallbackData(formatToKey(inf, "webm")),
					tu.InlineKeyboardButton("MP4").WithCallbackData(formatToKey(inf, "mp4")),
				),
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("PNG").WithCallbackData(formatToKey(inf, "png")),
					tu.InlineKeyboardButton("JPEG").WithCallbackData(formatToKey(inf, "jpeg")),
					tu.InlineKeyboardButton("WEBP").WithCallbackData(formatToKey(inf, "webp")),
				),
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("Lottie.JSON").WithCallbackData(formatToKey(inf, "lottie")),
				),
			)
		case isVideo:
			inf = "webm"
			kb = tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("PNG").WithCallbackData(formatToKey(inf, "png")),
					tu.InlineKeyboardButton("JPEG").WithCallbackData(formatToKey(inf, "jpeg")),
					tu.InlineKeyboardButton("WEBP").WithCallbackData(formatToKey(inf, "webp")),
				),
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("GIF").WithCallbackData(formatToKey(inf, "gif")),
				),
			)
		default:
			inf = "webp"
			kb = tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("PNG").WithCallbackData(formatToKey(inf, "png")),
					tu.InlineKeyboardButton("JPEG").WithCallbackData(formatToKey(inf, "jpeg")),
				),
			)
		}

		rdb.Del(ctx, fmt.Sprintf("user_state:%d", userID))

		state := storage.UserState{
			Stage:  "init",
			FileID: stickerFileID,
		}

		if err := storage.SaveState(ctx, rdb, userID, state); err != nil {
			logger.Error("Failed to save user state",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "sticker_received",
			)
		}

		logger.Info("User state initialized",
			"user_id", userID,
			"chat_id", chatID,
			"input_format", inf,
			"sticker_file_id", stickerFileID,
			"action", "sticker_received",
		)

		formatDisplayName := map[string]string{
			"tgs":  "TGS (Animated)",
			"webm": "WEBM (Video)",
			"webp": "WEBP (Static)",
		}
		inputFormatName := formatDisplayName[inf]
		if inputFormatName == "" {
			inputFormatName = strings.ToUpper(inf)
		}

		message := fmt.Sprintf("🎨 *Sticker received\\!*\n\n📥 Input format: *%s*\n\n📤 Select output format:",
			escapeMarkdownV2(inputFormatName))

		_, err := bot.SendMessage(ctx, tu.Message(tu.ID(chatID), message).
			WithParseMode(telego.ModeMarkdownV2).
			WithReplyMarkup(kb))
		if err != nil {
			logger.Error("Failed to send format selection message",
				"user_id", userID,
				"chat_id", chatID,
				"error", err,
				"action", "sticker_received",
			)
			return err
		}
		logger.Info("Format selection message sent",
			"user_id", userID,
			"chat_id", chatID,
			"input_format", inf,
			"action", "sticker_received",
		)
		return nil
	}
}

func formatToKey(inf, outf string) string {
	v := url.Values{}
	v.Set("action", "format_select")
	v.Set("in", inf)
	v.Set("out", outf)
	return v.Encode()
}
