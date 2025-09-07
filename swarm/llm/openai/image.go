package openai

import (
	"context"
	"strings"

	"github.com/openai/openai-go"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
)

func ImageGen(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	log.Debugf(">>>OPENAI:\n image-gen req: %+v\n\n", req)

	var err error
	var resp *llm.Response

	resp, err = generateImage(ctx, req)

	log.Debugf("<<<OPENAI:\n image-gen resp: %+v err: %v\n\n", resp, err)
	return resp, err
}

func generateImage(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	messages := make([]string, 0)
	for _, v := range req.Messages {
		messages = append(messages, v.Content)
	}

	client := NewClient(req.Model, req.Vars)
	prompt := strings.Join(messages, "\n")
	model := req.Model.Model

	resp := &llm.Response{
		ContentType: api.ContentTypeB64JSON,
	}

	log.Infof("@%s %s %s\n", req.Agent, req.Model, req.Model.BaseUrl)

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

	if q, ok := qualityMap[req.Vars.Extra["quality"]]; ok {
		imageQuality = q
	}
	if s, ok := sizeMap[req.Vars.Extra["size"]]; ok {
		imageSize = s
	}
	if s, ok := styleMap[req.Vars.Extra["style"]]; ok {
		imageStyle = s
	}

	image, err := client.Images.Generate(ctx, openai.ImageGenerateParams{
		Prompt:         prompt,
		Model:          model,
		ResponseFormat: imageFormat,
		Quality:        imageQuality,
		Size:           imageSize,
		Style:          imageStyle,
		N:              openai.Int(1),
	})
	if err != nil {
		return nil, err
	}
	log.Infof("✨ %v %v %v\n", imageQuality, imageSize, imageStyle)

	resp.Content = image.Data[0].B64JSON

	return resp, nil
}
