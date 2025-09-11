package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/oapi-codegen/runtime"
	converter "github.com/yazmeyaa/telegram_sticker_converter"
	"github.com/yazmeyaa/telegram_sticker_converter/tgs"
	"github.com/yazmeyaa/telegram_sticker_converter/webm"
	"github.com/yazmeyaa/telegram_sticker_converter/webp"
	api "github.com/yazmeyaa/telegram_sticker_converter_backend/internal/api/gen"
	"gopkg.in/yaml.v3"
)

type Server struct{}

func requestFrameToConverterFrame(frame api.ConvertStickerMultipartBodyFrame) (converter.FrameSelector, error) {
	switch frame {
	case api.All:
		return converter.FrameAll, nil
	case api.First:
		return converter.FrameFirst, nil
	case api.N:
		return converter.FrameN, nil
	case api.Range:
		return converter.FrameRange, nil
	default:
		return converter.FrameAll, errors.New("unexpected frame type")
	}
}

func (s *Server) GetSwaggerSpec(w http.ResponseWriter, r *http.Request) {
	spec, err := api.GetSwagger()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := spec.MarshalYAML()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/yaml")
	yaml.NewEncoder(w).Encode(data)
}

// ConvertSticker implements api.ServerInterface.
func (s Server) ConvertSticker(w http.ResponseWriter, r *http.Request) {
	var body api.ConvertStickerMultipartBody
	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to bind multipart: %v", err), http.StatusBadRequest)
		return
	}
	if err := runtime.BindMultipart(&body, *reader); err != nil {
		http.Error(w, fmt.Sprintf("Failed to bind multipart: %v", err), http.StatusBadRequest)
		return
	}

	switch body.InputFormat {
	case api.ConvertStickerMultipartBodyInputFormatTgs:
		conv := tgs.NewConverter()

		var frame converter.FrameSelector
		if body.Frame != nil {
			f, err := requestFrameToConverterFrame(*body.Frame)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			frame = f
		}

		opts := converter.TGSTransformOptions{
			Format: converter.OutputFormat(body.OutputFormat),
			Frame:  frame,
		}
		if body.FrameIndex != nil {
			opts.FrameIndex = *body.FrameIndex
		}
		if body.FrameOffset != nil {
			opts.FrameOffset = *body.FrameOffset
		}
		if body.ResizeWidth != nil {
			opts.ResizeWidth = *body.ResizeWidth
		}
		if body.ResizeHeight != nil {
			opts.ResizeHeight = *body.ResizeHeight
		}

		file, err := body.File.Reader()
		if err != nil {
			http.Error(w, "Cannot read file", http.StatusInternalServerError)
			fmt.Fprintf(os.Stdout, "err: %s", err.Error())
			return
		}

		if err := conv.Transform(r.Context(), file, w, opts); err != nil {
			http.Error(w, "Error while transform sticker", http.StatusInternalServerError)
			fmt.Fprintf(os.Stdout, "err: %s", err.Error())
			return
		}
	case api.ConvertStickerMultipartBodyInputFormatWebm:
		conv := webm.NewConverter()
		var frame converter.FrameSelector
		if body.Frame != nil {
			f, err := requestFrameToConverterFrame(*body.Frame)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			frame = f
		}

		opts := converter.WEBMTransformOptions{
			Format: converter.OutputFormat(body.OutputFormat),
			Frame:  frame,
		}
		if body.FrameIndex != nil {
			opts.FrameIndex = uint(*body.FrameIndex)
		}
		if body.FrameOffset != nil {
			opts.FrameOffset = uint(*body.FrameOffset)
		}
		if body.ResizeWidth != nil {
			opts.Width = uint(*body.ResizeWidth)
		}
		if body.ResizeHeight != nil {
			opts.Height = uint(*body.ResizeHeight)
		}

		file, err := body.File.Reader()
		if err != nil {
			http.Error(w, "Cannot read file", http.StatusInternalServerError)
			fmt.Fprintf(os.Stdout, "err: %s", err.Error())
			return
		}

		if err := conv.Transform(r.Context(), file, w, opts); err != nil {
			http.Error(w, "Error while transform sticker", http.StatusInternalServerError)
			fmt.Fprintf(os.Stdout, "err: %s", err.Error())
			return
		}

	case api.ConvertStickerMultipartBodyInputFormatWebp:
		conv := webp.NewConverter()
		opts := converter.WebpTransformOptions{
			Format: converter.OutputFormat(body.OutputFormat),
		}

		file, err := body.File.Reader()
		if err != nil {
			http.Error(w, "Cannot read file", http.StatusInternalServerError)
			fmt.Fprintf(os.Stdout, "err: %s", err.Error())
			return
		}

		if err := conv.Transform(r.Context(), file, w, opts); err != nil {
			http.Error(w, "Error while transform sticker", http.StatusInternalServerError)
			fmt.Fprintf(os.Stdout, "err: %s", err.Error())
			return
		}
	default:
		http.Error(w, "Unexpected input type", http.StatusBadRequest)
	}
}

func main() {
	server := Server{}
	mux := http.NewServeMux()

	h := api.HandlerWithOptions(server, api.StdHTTPServerOptions{
		BaseRouter: mux,
	})
	mux.HandleFunc("GET /docs/openapi.yaml", server.GetSwaggerSpec)

	fmt.Fprint(os.Stdout, "Starting server...\n")
	if err := http.ListenAndServe("0.0.0.0:8000", h); err != nil {
		panic(err)
	}
}
