package gitkit

import (
	"context"
	"sort"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Tags returns a sorted list of tag names in the repository.
func Tags(ctx context.Context, r *git.Repository) ([]string, error) {
	var out []string
	refs, err := r.Tags()
	if err != nil {
		return nil, err
	}
	_ = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref == nil {
			return nil
		}
		out = append(out, ref.Name().Short())
		return nil
	})
	sort.Strings(out)
	return out, nil
}
