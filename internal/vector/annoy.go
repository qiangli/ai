// https://github.com/mariotoffia/goannoy
// https://github.com/spotify/annoy

package vector

import (
	"github.com/mariotoffia/goannoy/builder"
	"github.com/mariotoffia/goannoy/interfaces"
)

type VectorStore struct {
	idx interfaces.AnnoyIndex[float32, uint32]
}

func New(f int /*vectorLength*/) *VectorStore {
	idx := builder.Index[float32, uint32]().
		AngularDistance(f).
		UseMultiWorkerPolicy().
		MmapIndexAllocator().Build()

	return &VectorStore{
		idx: idx,
	}
}

func (r *VectorStore) Build(ntrees int) {
	r.idx.Build(ntrees, -1)
}

func (r *VectorStore) Save(filename string) error {
	return r.idx.Save(filename)
}

func (r *VectorStore) Load(filename string) error {
	return r.idx.Load(filename)
}

func (r *VectorStore) GetByVector(query []float32, n int) ([]uint32, []float32) {
	ctx := r.idx.CreateContext()
	return r.idx.GetNnsByVector(query, n, -1, ctx)
}

func (r *VectorStore) GetByItem(item uint32, n int) ([]uint32, []float32) {
	ctx := r.idx.CreateContext()
	return r.idx.GetNnsByItem(item, n, -1, ctx)
}

func (vs *VectorStore) AddItem(id uint32, vector []float32) {
	vs.idx.AddItem(id, vector)
}
