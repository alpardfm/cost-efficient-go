package main

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

// ============================================================
// PATTERN 4: JSON Processing Efficiency
// ============================================================
// Problem: encoding/json uses reflection heavily, causing
// significant CPU and memory overhead at scale.
//
// This pattern demonstrates:
// 1. Struct tag optimization (omitempty, string)
// 2. Pre-allocated encoder/decoder buffers
// 3. Avoiding interface{}/any in hot paths
// 4. Pointer vs value receiver impact on marshaling
// ============================================================

// ❌ BAD: Unoptimized response struct
type BadResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
	Meta    BadMeta     `json:"meta"`
}

type BadMeta struct {
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	Total      int    `json:"total"`
	TotalPages int    `json:"total_pages"`
	RequestID  string `json:"request_id"`
	Version    string `json:"version"`
}

// ✅ GOOD: Optimized response struct
type GoodResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
	Meta    *GoodMeta   `json:"meta,omitempty"`
}

type GoodMeta struct {
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"page_size,omitempty"`
	Total      int    `json:"total,omitempty"`
	TotalPages int    `json:"total_pages,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	Version    string `json:"version,omitempty"`
}

// ❌ BAD: Using map[string]interface{} for known structures
type BadProduct struct {
	ID         int                    `json:"id"`
	Name       string                 `json:"name"`
	Price      float64                `json:"price"`
	Attributes map[string]interface{} `json:"attributes"`
}

// ✅ GOOD: Using typed struct for known fields
type GoodProduct struct {
	ID    int           `json:"id"`
	Name  string        `json:"name"`
	Price float64       `json:"price"`
	Attrs *ProductAttrs `json:"attributes,omitempty"`
}

type ProductAttrs struct {
	Color  string `json:"color,omitempty"`
	Size   string `json:"size,omitempty"`
	Weight int    `json:"weight,omitempty"`
}

// ============================================================
// Demonstration
// ============================================================

func main() {
	fmt.Println("=== JSON Processing Efficiency ===")
	fmt.Println()

	// Show size difference
	fmt.Printf("BadResponse size:  %d bytes\n", unsafe.Sizeof(BadResponse{}))
	fmt.Printf("GoodResponse size: %d bytes\n", unsafe.Sizeof(GoodResponse{}))
	fmt.Println()

	// Demonstrate omitempty savings
	good := GoodResponse{
		Status: "success",
		Data:   map[string]string{"id": "123"},
	}
	bad := BadResponse{
		Status:  "success",
		Message: "",
		Data:    map[string]string{"id": "123"},
		Error:   "",
		Meta:    BadMeta{},
	}

	goodJSON, _ := json.Marshal(good)
	badJSON, _ := json.Marshal(bad)

	fmt.Printf("Bad JSON output:  %d bytes → %s\n", len(badJSON), string(badJSON))
	fmt.Printf("Good JSON output: %d bytes → %s\n", len(goodJSON), string(goodJSON))
	fmt.Printf("Savings per response: %d bytes (%.1f%%)\n",
		len(badJSON)-len(goodJSON),
		float64(len(badJSON)-len(goodJSON))/float64(len(badJSON))*100,
	)
	fmt.Println()

	// Demonstrate typed vs map overhead
	badProduct := BadProduct{
		ID:    1,
		Name:  "T-Shirt",
		Price: 29.99,
		Attributes: map[string]interface{}{
			"color":  "blue",
			"size":   "M",
			"weight": 200,
		},
	}
	goodProduct := GoodProduct{
		ID:    1,
		Name:  "T-Shirt",
		Price: 29.99,
		Attrs: &ProductAttrs{
			Color:  "blue",
			Size:   "M",
			Weight: 200,
		},
	}

	fmt.Printf("BadProduct size:  %d bytes (struct) + map overhead\n", unsafe.Sizeof(badProduct))
	fmt.Printf("GoodProduct size: %d bytes (struct)\n", unsafe.Sizeof(goodProduct))

	badProdJSON, _ := json.Marshal(badProduct)
	goodProdJSON, _ := json.Marshal(goodProduct)
	fmt.Printf("BadProduct JSON:  %d bytes\n", len(badProdJSON))
	fmt.Printf("GoodProduct JSON: %d bytes\n", len(goodProdJSON))
}
