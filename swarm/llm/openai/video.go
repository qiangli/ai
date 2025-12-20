package openai

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func Video(ctx context.Context, req *api.Request) (*api.Response, error) {
	log.GetLogger(ctx).Debugf(">OPENAI:\n video req: %+v\n", req)

	var err error
	var resp *api.Response

	resp, err = genVideo(ctx, req)

	log.GetLogger(ctx).Debugf(">OPENAI:\n video resp: %+v err: %v\n", resp, err)
	return resp, err
}

func genVideo(ctx context.Context, req *api.Request) (*api.Response, error) {
	client, err := NewClient(req.Model, req.Token())
	if err != nil {
		return nil, err
	}

	messages := make([]string, 0)
	for _, v := range req.Messages {
		messages = append(messages, v.Content)
	}
	prompt := strings.Join(messages, "\n")

	params := openai.VideoNewParams{
		Model:   openai.VideoModel(req.Model.Model),
		Prompt:  prompt,
		Seconds: openai.VideoSeconds4,
		Size:    openai.VideoSize720x1280,
	}

	getStrArg := func(key string, args api.Arguments, val string) string {
		v := args.GetString(key)
		if v != "" {
			return v
		}
		return val
	}
	getIntArg := func(key string, args api.Arguments, val int) int {
		v := args.GetInt(key)
		if v != 0 {
			return v
		}
		return val
	}

	if v := getStrArg("input_reference", req.Arguments, ""); v != "" {
		ref, err := fetchContent(v)
		if err != nil {
			return nil, err
		}
		params.InputReference = openai.File(ref, "reference.jpg", "image/jpeg")
	}
	if v := getStrArg("seconds", req.Arguments, "4"); v != "" {
		params.Seconds = openai.VideoSeconds(v)
	}
	var sizeMap = map[string]openai.VideoSize{
		"720x1280":  openai.VideoSize720x1280,
		"1280x720":  openai.VideoSize1280x720,
		"1024x1792": openai.VideoSize1024x1792,
		"1792x1024": openai.VideoSize1792x1024,
	}
	if size := getStrArg("size", req.Arguments, "720x1280"); size != "" {
		if v, ok := sizeMap[size]; ok {
			params.Size = v
		}
	}
	var pollIntervalMs = getIntArg("poll_interval", req.Arguments, 1000)

	video, err := client.Videos.NewAndPoll(ctx, params, pollIntervalMs)
	if err != nil {
		return nil, err
	}

	var result = ""
	if video.Status == openai.VideoStatusCompleted {
		result = fmt.Sprintf("Video successfully completed: %v", video)
	} else {
		result = fmt.Sprintf("Video creation failed. Status: %s", video.Status)
	}

	return &api.Response{
		Result: &api.Result{
			Value: result,
		},
	}, nil
}
