package openai

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
)

func Video(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	log.GetLogger(ctx).Debugf(">OPENAI:\n video req: %+v\n", req)

	var err error
	var resp *llm.Response

	resp, err = genVideo(ctx, req)

	log.GetLogger(ctx).Debugf(">OPENAI:\n video resp: %+v err: %v\n", resp, err)
	return resp, err
}

func genVideo(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	client, err := NewClient(req.Model, req.Token(), req.Vars)
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

	if v := GetStrArg("input_reference", req.Arguments, ""); v != "" {
		ref, err := fetchContent(v)
		if err != nil {
			return nil, err
		}
		params.InputReference = openai.File(ref, "reference.jpg", "image/jpeg")
	}
	if v := GetStrArg("seconds", req.Arguments, "4"); v != "" {
		params.Seconds = openai.VideoSeconds(v)
	}
	var sizeMap = map[string]openai.VideoSize{
		"720x1280":  openai.VideoSize720x1280,
		"1280x720":  openai.VideoSize1280x720,
		"1024x1792": openai.VideoSize1024x1792,
		"1792x1024": openai.VideoSize1792x1024,
	}
	if size := GetStrArg("size", req.Arguments, "720x1280"); size != "" {
		if v, ok := sizeMap[size]; ok {
			params.Size = v
		}
	}
	var pollIntervalMs = GetIntArg("poll_interval", req.Arguments, 1000)

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

	return &llm.Response{
		Result: &api.Result{
			Value: result,
		},
	}, nil
}
