package embed

import "testing"

func TestLocalRandomEmbedder(t *testing.T) {
	p := &LocalRandomEmbedder{}
	embs, err := p.Embed([]string{"test text", "another"})
	if err != nil {
		t.Fatal(err)
	}
	if len(embs) != 2 {
		t.Fatal("wrong batch size")
	}
	if len(embs[0]) != dim {
		t.Errorf("wrong dim %d", len(embs[0]))
	}
}
