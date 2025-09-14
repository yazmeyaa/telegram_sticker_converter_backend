package converter

import (
	"context"
	"io"

	tc "github.com/yazmeyaa/telegram_sticker_converter"
	"github.com/yazmeyaa/telegram_sticker_converter/tgs"
	"github.com/yazmeyaa/telegram_sticker_converter/webm"
	"github.com/yazmeyaa/telegram_sticker_converter/webp"
	"github.com/yazmeyaa/telegram_sticker_converter_backend/internal/storage"
)

func ConvertSticker(ctx context.Context, in io.Reader, out io.Writer, state storage.UserState) error {
	switch state.InputFormat {
	case "webp":
		return webp.NewConverter().Transform(ctx, in, out, tc.WebpTransformOptions{
			Format: formatFromString(state.OutputFormat),
		})

	case "webm":
		return webm.NewConverter().Transform(ctx, in, out, tc.WEBMTransformOptions{
			Format: formatFromString(state.OutputFormat),
			Frame:  frameFromString(state.FrameOption),
		})

	case "tgs":
		return tgs.NewConverter().Transform(ctx, in, out, tc.TGSTransformOptions{
			Format: formatFromString(state.OutputFormat),
			Frame:  frameFromString(state.FrameOption),
		})
	}
	return nil
}

func formatFromString(f string) tc.OutputFormat {
	switch f {
	case "png":
		return tc.FormatPNG
	case "jpeg":
		return tc.FormatJPEG
	case "gif":
		return tc.FormatGIF
	case "webp":
		return tc.FormatWEBP
	case "webm":
		return tc.FormatWEBM
	case "mp4":
		return tc.FormatMP4
	case "lottie":
		return tc.FormatLottie
	default:
		return tc.FormatPNG
	}
}

func frameFromString(f string) tc.FrameSelector {
	switch f {
	case "first":
		return tc.FrameFirst
	case "nth":
		return tc.FrameN
	case "all":
		return tc.FrameAll
	default:
		return tc.FrameFirst
	}
}
