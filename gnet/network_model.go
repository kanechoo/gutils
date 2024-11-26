package gnet

type MTRResult struct {
	Sequence int
	Host     string
	Loss     string
	Snt      int
	Last     float64
	Avg      float64
	Best     float64
	Wrst     float64
}
