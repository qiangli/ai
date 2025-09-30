package conf

import (
	"context"

	"github.com/qiangli/ai/internal/bubble"
	"github.com/qiangli/ai/swarm/api"
)

// func (r *SystemKit) ReadStdin(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	return readStdin()
// }

// func (r *SystemKit) PasteFromClipboard(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	return readClipboard(ctx)
// }

// func (r *SystemKit) PasteFromClipboardWait(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	return readClipboardWait(ctx)
// }

// func (r *SystemKit) WriteStdout(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	content, err := r.getStr("content", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	return writeStdout(content)
// }

// func (r *SystemKit) CopyToClipboard(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	content, err := r.getStr("content", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	return writeClipboard(content)
// }

// func (r *SystemKit) CopyToClipboardAppend(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	content, err := r.getStr("content", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	return writeClipboardAppend(content)
// }

// func (r *SystemKit) GetUserTextInput(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	prompt, err := r.getStr("prompt", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	return bubble.Edit(prompt, "Enter here...", "")
// }

func (r *SystemKit) Confirm(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	prompt, err := r.getStr("prompt", args)
	if err != nil {
		return "", err
	}
	return bubble.Confirm(prompt)
}

// func (r *SystemKit) GetUserChoiceInput(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	prompt, err := r.getStr("prompt", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	choices, err := r.getArray("choices", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	return bubble.Choose(prompt, choices, false)
// }

// func (r *SystemKit) GetFileContentInput(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	prompt, err := r.getStr("prompt", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	pathname, err := r.getStr("pathname", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	return bubble.PickFile(prompt, pathname)
// }
