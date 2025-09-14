package storage

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type UserState struct {
	FileID       string
	Stage        string
	InputFormat  string
	OutputFormat string
	FrameOption  string
	Size         string
}

func SaveState(ctx context.Context, rdb *redis.Client, userID int64, state UserState) error {
	key := fmt.Sprintf("user_state:%d", userID)
	return rdb.HSet(ctx, key, map[string]any{
		"file_id":       state.FileID,
		"stage":         state.Stage,
		"input_format":  state.InputFormat,
		"output_format": state.OutputFormat,
		"frame_option":  state.FrameOption,
		"size":          state.Size,
	}).Err()
}

func LoadState(ctx context.Context, rdb *redis.Client, userID int64) (*UserState, error) {
	key := fmt.Sprintf("user_state:%d", userID)
	res, err := rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, redis.Nil
	}
	return &UserState{
		FileID:       res["file_id"],
		Stage:        res["stage"],
		InputFormat:  res["input_format"],
		OutputFormat: res["output_format"],
		FrameOption:  res["frame_option"],
		Size:         res["size"],
	}, nil
}
