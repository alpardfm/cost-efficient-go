package json_processing

import (
	"encoding/json"
	"testing"
)

// ============================================================
// Benchmarks: JSON Marshal/Unmarshal Performance
// ============================================================

// --- Marshal Benchmarks ---

func BenchmarkMarshalBadResponse(b *testing.B) {
	resp := BadResponse{
		Status:  "success",
		Message: "",
		Data:    map[string]string{"id": "user-123", "name": "John Doe"},
		Error:   "",
		Meta: BadMeta{
			Page:       1,
			PageSize:   20,
			Total:      150,
			TotalPages: 8,
			RequestID:  "req-abc-123",
			Version:    "1.0.0",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		json.Marshal(resp)
	}
}

func BenchmarkMarshalGoodResponse(b *testing.B) {
	resp := GoodResponse{
		Status: "success",
		Data:   map[string]string{"id": "user-123", "name": "John Doe"},
		Meta: &GoodMeta{
			Page:       1,
			PageSize:   20,
			Total:      150,
			TotalPages: 8,
			RequestID:  "req-abc-123",
			Version:    "1.0.0",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		json.Marshal(resp)
	}
}

func BenchmarkMarshalGoodResponseMinimal(b *testing.B) {
	// Common case: success response without meta (e.g., create/update)
	resp := GoodResponse{
		Status: "success",
		Data:   map[string]string{"id": "user-123"},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		json.Marshal(resp)
	}
}

// --- Unmarshal Benchmarks ---

func BenchmarkUnmarshalBadResponse(b *testing.B) {
	data := []byte(`{"status":"success","message":"","data":{"id":"user-123","name":"John Doe"},"error":"","meta":{"page":1,"page_size":20,"total":150,"total_pages":8,"request_id":"req-abc-123","version":"1.0.0"}}`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var resp BadResponse
		json.Unmarshal(data, &resp)
	}
}

func BenchmarkUnmarshalGoodResponse(b *testing.B) {
	data := []byte(`{"status":"success","data":{"id":"user-123","name":"John Doe"},"meta":{"page":1,"page_size":20,"total":150,"total_pages":8,"request_id":"req-abc-123","version":"1.0.0"}}`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var resp GoodResponse
		json.Unmarshal(data, &resp)
	}
}

// --- Map vs Struct Benchmarks ---

func BenchmarkMarshalMapProduct(b *testing.B) {
	product := BadProduct{
		ID:    1,
		Name:  "T-Shirt Premium Cotton",
		Price: 29.99,
		Attributes: map[string]interface{}{
			"color":  "blue",
			"size":   "M",
			"weight": 200,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		json.Marshal(product)
	}
}

func BenchmarkMarshalStructProduct(b *testing.B) {
	product := GoodProduct{
		ID:    1,
		Name:  "T-Shirt Premium Cotton",
		Price: 29.99,
		Attrs: &ProductAttrs{
			Color:  "blue",
			Size:   "M",
			Weight: 200,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		json.Marshal(product)
	}
}

func BenchmarkUnmarshalMapProduct(b *testing.B) {
	data := []byte(`{"id":1,"name":"T-Shirt Premium Cotton","price":29.99,"attributes":{"color":"blue","size":"M","weight":200}}`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var product BadProduct
		json.Unmarshal(data, &product)
	}
}

func BenchmarkUnmarshalStructProduct(b *testing.B) {
	data := []byte(`{"id":1,"name":"T-Shirt Premium Cotton","price":29.99,"attributes":{"color":"blue","size":"M","weight":200}}`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var product GoodProduct
		json.Unmarshal(data, &product)
	}
}

// --- Batch Processing Benchmarks ---

func BenchmarkMarshalBatch1000BadProducts(b *testing.B) {
	products := make([]BadProduct, 1000)
	for i := range products {
		products[i] = BadProduct{
			ID:    i,
			Name:  "Product",
			Price: 9.99,
			Attributes: map[string]interface{}{
				"color": "red",
				"size":  "L",
			},
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		json.Marshal(products)
	}
}

func BenchmarkMarshalBatch1000GoodProducts(b *testing.B) {
	products := make([]GoodProduct, 1000)
	for i := range products {
		products[i] = GoodProduct{
			ID:    i,
			Name:  "Product",
			Price: 9.99,
			Attrs: &ProductAttrs{
				Color: "red",
				Size:  "L",
			},
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		json.Marshal(products)
	}
}
