package struct_alignment

// BadUser demonstrates a struct with suboptimal field ordering that causes
// excessive padding due to memory alignment requirements.
type BadUser struct {
	ID     int32
	Active bool
	Name   string
	Age    int8
}

// GoodUser demonstrates the same struct with fields reordered to minimize
// padding bytes, reducing memory usage per instance.
type GoodUser struct {
	ID     int32
	Age    int8
	Active bool
	Name   string
}
