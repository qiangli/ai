package openai

import (
	"context"
	"strings"

	"github.com/openai/openai-go/v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func Image(ctx context.Context, req *api.Request) (*api.Response, error) {
	log.GetLogger(ctx).Debugf(">OPENAI:\n image-gen req: %+v\n", req)

	var err error
	var resp *api.Response

	resp, err = genImage(ctx, req)

	log.GetLogger(ctx).Debugf(">OPENAI:\n image-gen resp: %+v err: %v\n", resp, err)
	return resp, err
}

func genImage(ctx context.Context, req *api.Request) (*api.Response, error) {
	client, err := NewClient(req.Model, req.Token())
	if err != nil {
		return nil, err
	}

	messages := make([]string, 0)
	for _, v := range req.Messages {
		messages = append(messages, v.Content)
	}
	prompt := strings.Join(messages, "\n")

	log.GetLogger(ctx).Infof("Ⓞ @%s/%s %s [%v/%v] image %s/%s\n", req.Agent.Pack, req.Agent.Name, req.Agent.Display, req.Model.Provider, req.Model.Model)

	var imageFormat = openai.ImageGenerateParamsResponseFormatB64JSON

	var qualityMap = map[string]openai.ImageGenerateParamsQuality{
		"standard": openai.ImageGenerateParamsQualityStandard,
		"hd":       openai.ImageGenerateParamsQualityHD,
	}
	var sizeMap = map[string]openai.ImageGenerateParamsSize{
		// no longer supported?
		// "256x256":   openai.ImageGenerateParamsSize256x256,
		// "512x512":   openai.ImageGenerateParamsSize512x512,
		"1024x1024": openai.ImageGenerateParamsSize1024x1024,
		"1792x1024": openai.ImageGenerateParamsSize1792x1024,
		"1024x1792": openai.ImageGenerateParamsSize1024x1792,
	}
	var styleMap = map[string]openai.ImageGenerateParamsStyle{
		"vivid":   openai.ImageGenerateParamsStyleVivid,
		"natural": openai.ImageGenerateParamsStyleNatural,
	}

	var imageQuality = openai.ImageGenerateParamsQualityStandard
	var imageSize = openai.ImageGenerateParamsSize1024x1024
	var imageStyle = openai.ImageGenerateParamsStyleNatural

	if v := req.Arguments; v != nil {
		if key := v.GetString("quality"); key != "" {
			if q, ok := qualityMap[key]; ok {
				imageQuality = q
			}
		}
		if key := v.GetString("size"); key != "" {
			if s, ok := sizeMap[key]; ok {
				imageSize = s
			}
		}
		if key := v.GetString("style"); key != "" {
			if s, ok := styleMap[key]; ok {
				imageStyle = s
			}
		}
	}

	image, err := client.Images.Generate(ctx, openai.ImageGenerateParams{
		Prompt:         prompt,
		Model:          req.Model.Model,
		ResponseFormat: imageFormat,
		Quality:        imageQuality,
		Size:           imageSize,
		Style:          imageStyle,
		N:              openai.Int(1),
	})
	if err != nil {
		return nil, err
	}
	log.GetLogger(ctx).Infof("✨ %v %v %v\n", imageQuality, imageSize, imageStyle)

	return &api.Response{
		Result: &api.Result{
			// State:    api.StateExit,
			MimeType: api.ContentTypeImageB64,
			Value:    image.Data[0].B64JSON,
		},
	}, nil
}
