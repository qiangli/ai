// https://github.com/mariotoffia/goannoy
// https://github.com/spotify/annoy

// https://github.com/blevesearch/bleve/blob/master/docs/vectors.md
package db

// import (
// 	"github.com/mariotoffia/goannoy/builder"
// 	"github.com/mariotoffia/goannoy/interfaces"
// )

// type Index struct {
// 	idx interfaces.AnnoyIndex[float32, uint32]
// }

// func NewIndex(f int /*vectorLength*/) *Index {
// 	idx := builder.Index[float32, uint32]().
// 		AngularDistance(f).
// 		UseMultiWorkerPolicy().
// 		MmapIndexAllocator().Build()

// 	return &Index{
// 		idx: idx,
// 	}
// }

// func (r *Index) Build(ntrees int) {
// 	r.idx.Build(ntrees, -1)
// }

// func (r *Index) Save(filename string) error {
// 	return r.idx.Save(filename)
// }

// func (r *Index) Load(filename string) error {
// 	return r.idx.Load(filename)
// }

// func (r *Index) GetByVector(query []float32, n int) ([]uint32, []float32) {
// 	ctx := r.idx.CreateContext()
// 	return r.idx.GetNnsByVector(query, n, -1, ctx)
// }

// func (r *Index) GetByItem(item uint32, n int) ([]uint32, []float32) {
// 	ctx := r.idx.CreateContext()
// 	return r.idx.GetNnsByItem(item, n, -1, ctx)
// }

// func (r *Index) AddItem(id uint32, vector []float32) {
// 	r.idx.AddItem(id, vector)
// }

// func (r *Index) GetItem(id uint32) []float32 {
// 	return r.idx.GetItem(id)
// }

// func (r *Index) Close() error {
// 	return r.idx.Close()
// }
