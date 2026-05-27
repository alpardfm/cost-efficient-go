package json_processing

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

// BadResponse is an unoptimized response struct that always marshals all fields,
// including empty ones, wasting bandwidth and CPU cycles.
type BadResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
	Meta    BadMeta     `json:"meta"`
}

// BadMeta is an unoptimized metadata struct without omitempty tags.
type BadMeta struct {
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	Total      int    `json:"total"`
	TotalPages int    `json:"total_pages"`
	RequestID  string `json:"request_id"`
	Version    string `json:"version"`
}

// GoodResponse is an optimized response struct using omitempty to skip empty fields.
type GoodResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
	Meta    *GoodMeta   `json:"meta,omitempty"`
}

// GoodMeta is an optimized metadata struct using omitempty and pointer receiver.
type GoodMeta struct {
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"page_size,omitempty"`
	Total      int    `json:"total,omitempty"`
	TotalPages int    `json:"total_pages,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	Version    string `json:"version,omitempty"`
}

// BadProduct uses map[string]interface{} for known structures, causing
// extra allocations and type assertions.
type BadProduct struct {
	ID         int                    `json:"id"`
	Name       string                 `json:"name"`
	Price      float64                `json:"price"`
	Attributes map[string]interface{} `json:"attributes"`
}

// GoodProduct uses typed struct for known fields, avoiding map overhead.
type GoodProduct struct {
	ID    int           `json:"id"`
	Name  string        `json:"name"`
	Price float64       `json:"price"`
	Attrs *ProductAttrs `json:"attributes,omitempty"`
}

// ProductAttrs holds typed product attributes instead of a generic map.
type ProductAttrs struct {
	Color  string `json:"color,omitempty"`
	Size   string `json:"size,omitempty"`
	Weight int    `json:"weight,omitempty"`
}
