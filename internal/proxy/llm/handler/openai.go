package handler

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/hertz-contrib/sse"

	"github.com/qiangli/ai/internal/log"
)

// https://github.com/litongjava/go-llm-proxy/blob/main/handlers/openai_handler.go
var OpenAiApiPrefix = "https://api.openai.com/v1"
var OpenAiChatCompletionUrl = OpenAiApiPrefix + "/chat/completions"
var OpenAiModelUrl = OpenAiApiPrefix + "/models"

// OPENAI_API_KEY
var ApiKey = ""

// TODO login/auth
// internal secret
var ApiSecret = ""

func updateAuth(headers map[string]string) {
	if v, ok := headers["Authorization"]; ok {
		if v == ApiSecret {
			headers["Authorization"] = fmt.Sprintf("Bearer %s", ApiKey)
		}
	}
}

func OpenAiModels(ctx context.Context, reqCtx *app.RequestContext) {
	uri := reqCtx.URI()
	log.Infof("uri: %s\n", uri.String())

	headers := make(map[string]string)
	reqCtx.Request.Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})
	//
	headers["Host"] = "api.openai.com"

	updateAuth(headers)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}

	httpClient, err := client.NewClient(client.WithTLSConfig(tlsConfig))
	if err != nil {
		log.Errorf("Failed to create HTTP client: %v", err)
		Fail500(reqCtx, err)
		return
	}

	request := &protocol.Request{}
	response := &protocol.Response{}
	request.SetRequestURI(OpenAiModelUrl)
	request.SetMethod("GET")
	request.SetHeaders(headers)

	log.Debugf("headers: %+v", headers)

	err = httpClient.Do(context.Background(), request, response)
	defer response.CloseBodyStream()
	if err != nil {
		log.Errorf("url: %s error: %v\n", OpenAiApiPrefix, err)
		Fail500(reqCtx, err)
		return
	}

	response.Header.VisitAll(func(key, value []byte) {
		reqCtx.Response.Header.Set(string(key), string(value))
	})

	reqCtx.Response.SetBody(response.Body())
}
func OpenAiV1ChatCompletions(ctx context.Context, reqCtx *app.RequestContext) {
	// Read request body and headers
	body, _ := reqCtx.Body()
	// Decode the body into a map
	var requestMap map[string]interface{}
	sonic.ConfigDefault.Unmarshal(body, &requestMap)

	headers := make(map[string]string)
	reqCtx.Request.Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	updateAuth(headers)

	//headers.put("host", "api.openai.com");
	headers["Host"] = "api.openai.com"

	log.Debugf("body: %v\n", string(body))
	log.Debugf("headers: %+v\n", headers)

	if requestMap["stream"] == true {
		// Setup client to connect to the remote server
		client := sse.NewClient(OpenAiChatCompletionUrl)
		client.SetMethod("POST")
		client.SetHeaders(headers)
		client.SetBody(body)

		// Setup the stream for the original client
		var sEvent *sse.Stream = sse.NewStream(reqCtx)
		errChan := make(chan error)
		var completeContent strings.Builder
		go func() {
			err := client.Subscribe(func(msg *sse.Event) {
				if msg.Data != nil {
					// Forwarding the received event back to the original client
					event := &sse.Event{
						Data: msg.Data,
					}
					err := sEvent.Publish(event)
					if err != nil {
						log.Errorf("failed to send event to client: %+v error: %s", ctx, err)
						return
					}
					printResponseContent(msg, &completeContent)
				}
			})
			errChan <- err
		}()

		select {
		case err := <-errChan:
			if err != nil {
				log.Errorf("error from remote server: %+v error: %v\n", ctx, err)
				Fail500(reqCtx, err)
				return
			}
		}
	} else {
		httpClient, _ := client.NewClient(client.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true, //InsecureSkipVerify: true
		}))
		request := &protocol.Request{}
		response := &protocol.Response{}
		request.SetRequestURI(OpenAiChatCompletionUrl)
		request.SetMethod("POST")
		request.SetHeaders(headers)
		request.SetBody(body)

		var err = httpClient.Do(context.Background(), request, response)
		defer response.CloseBodyStream()
		if err != nil {
			log.Errorf("error: %v", err)
			Fail500(reqCtx, err)
			return
		}

		response.Header.VisitAll(func(key, value []byte) {
			reqCtx.Response.Header.Set(string(key), string(value))
		})

		reqCtx.Response.SetBody(response.Body())
	}
}
