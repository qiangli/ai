package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/hertz-contrib/sse"

	"github.com/qiangli/ai/internal/log"
)

func Fail400(reqCtx *app.RequestContext, err error) {
	reqCtx.JSON(http.StatusBadRequest, utils.H{
		"msg":  "Invalid request:" + err.Error(),
		"ok":   false,
		"code": 0,
	})
}

func Fail500(reqCtx *app.RequestContext, err error) {
	reqCtx.JSON(http.StatusInternalServerError, utils.H{
		"msg":  "Server Error:" + err.Error(),
		"ok":   false,
		"code": 0,
	})
}

func Ok(reqCtx *app.RequestContext, data interface{}) {
	reqCtx.JSON(http.StatusOK, utils.H{
		"code": 1,
		"data": data,
		"ok":   true,
	})
}

func printResponseContent(msg *sse.Event, completeContent *strings.Builder) {
	var responseData struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if len(msg.Data) > 6 {
		if err := json.Unmarshal(msg.Data, &responseData); err == nil {
			for _, choice := range responseData.Choices {
				//hlog.Info("content:", choice.Delta.Content)
				completeContent.WriteString(choice.Delta.Content)
			}
		} else {
			log.Errorf("json parsing error: %s\n", err)
		}
	}

	// Check if it's the end of the data
	if string(msg.Data) == "[DONE]" {
		log.Infof("Complete content: %s\n", completeContent.String())
		completeContent.Reset() // Clear the builder for the next message
	}
}
