package query_optimization

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"
)

// ============================================================
// PATTERN 7: Query Optimization & Indexing
// ============================================================
// Problem: Unoptimized database queries are the #1 cause of
// API latency in backend systems. Common mistakes:
// - SELECT * when only 2 columns needed
// - N+1 queries in loops
// - Missing indexes on WHERE/JOIN columns
// - Loading entire tables for pagination
//
// This pattern demonstrates the Go-side optimizations:
// 1. SELECT specific columns vs SELECT *
// 2. Batch queries vs N+1
// 3. Efficient pagination (keyset vs offset)
// 4. Query result scanning overhead
// ============================================================

// --- Simulated Data ---

// User represents a database row with many columns.
type User struct {
	ID        int
	Email     string
	Username  string
	FullName  string
	Password  string
	Bio       string
	Avatar    string
	Phone     string
	Address   string
	City      string
	Country   string
	Timezone  string
	Language  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserSummary represents only the columns we actually need.
type UserSummary struct {
	ID       int
	Email    string
	FullName string
}

// Order represents a related entity for N+1 demonstration.
type Order struct {
	ID     int
	UserID int
	Amount float64
	Status string
}

// --- Simulation Functions ---

// SimulateSelectStar scans all columns (simulates SELECT *)
func SimulateSelectStar(users []User) []User {
	result := make([]User, len(users))
	for i, u := range users {
		// Simulate copying all fields (like scanning from DB rows)
		result[i] = User{
			ID:        u.ID,
			Email:     u.Email,
			Username:  u.Username,
			FullName:  u.FullName,
			Password:  u.Password,
			Bio:       u.Bio,
			Avatar:    u.Avatar,
			Phone:     u.Phone,
			Address:   u.Address,
			City:      u.City,
			Country:   u.Country,
			Timezone:  u.Timezone,
			Language:  u.Language,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		}
	}
	return result
}

// SimulateSelectSpecific scans only needed columns (simulates SELECT id, email, full_name)
func SimulateSelectSpecific(users []User) []UserSummary {
	result := make([]UserSummary, len(users))
	for i, u := range users {
		result[i] = UserSummary{
			ID:       u.ID,
			Email:    u.Email,
			FullName: u.FullName,
		}
	}
	return result
}

// SimulateNPlusOne fetches orders one user at a time (N+1 pattern)
func SimulateNPlusOne(userIDs []int, ordersByUser map[int][]Order) [][]Order {
	result := make([][]Order, len(userIDs))
	for i, uid := range userIDs {
		// Simulates: SELECT * FROM orders WHERE user_id = ?
		result[i] = ordersByUser[uid]
	}
	return result
}

// SimulateNPlusOneWithLatency fetches orders one user at a time with simulated network round-trip.
// Each query incurs a network round-trip (time.Sleep), making N+1 realistically expensive.
func SimulateNPlusOneWithLatency(userIDs []int, ordersByUser map[int][]Order, latency time.Duration) [][]Order {
	result := make([][]Order, len(userIDs))
	for i, uid := range userIDs {
		// Simulates: network round-trip per query
		time.Sleep(latency)
		// Simulates: SELECT * FROM orders WHERE user_id = ?
		result[i] = ordersByUser[uid]
	}
	return result
}

// SimulateBatchQuery fetches all orders in one query (WHERE user_id IN (...))
func SimulateBatchQuery(userIDs []int, ordersByUser map[int][]Order) map[int][]Order {
	// Simulates: SELECT * FROM orders WHERE user_id IN (?, ?, ?, ...)
	result := make(map[int][]Order, len(userIDs))
	for _, uid := range userIDs {
		if orders, ok := ordersByUser[uid]; ok {
			result[uid] = orders
		}
	}
	return result
}

// SimulateBatchQueryWithLatency fetches all orders in one query with simulated network round-trip.
// Only one round-trip is needed regardless of how many user IDs are queried.
func SimulateBatchQueryWithLatency(userIDs []int, ordersByUser map[int][]Order, latency time.Duration) map[int][]Order {
	// Simulates: single network round-trip for batch query
	time.Sleep(latency)
	// Simulates: SELECT * FROM orders WHERE user_id IN (?, ?, ?, ...)
	result := make(map[int][]Order, len(userIDs))
	for _, uid := range userIDs {
		if orders, ok := ordersByUser[uid]; ok {
			result[uid] = orders
		}
	}
	return result
}

// SimulateOffsetPagination uses OFFSET (slow for deep pages)
// In real PostgreSQL, OFFSET must sequentially scan and discard rows,
// making it O(n) where n is the offset value.
func SimulateOffsetPagination(data []User, page, pageSize int) []User {
	offset := (page - 1) * pageSize
	if offset >= len(data) {
		return nil
	}
	// Simulate real DB behavior: must scan through all rows up to offset
	// PostgreSQL cannot skip rows — it must read and discard them sequentially
	found := 0
	start := 0
	for i := range data {
		if found == offset {
			start = i
			break
		}
		found++
	}
	end := start + pageSize
	if end > len(data) {
		end = len(data)
	}
	return data[start:end]
}

// SimulateKeysetPagination uses WHERE id > lastID (fast for any page)
// Uses binary search (sort.Search) to simulate B-tree index lookup O(log n),
// which is how PostgreSQL actually executes keyset pagination via index seek.
func SimulateKeysetPagination(data []User, lastID, pageSize int) []User {
	// Binary search: simulates B-tree index seek in PostgreSQL
	// In real DB: WHERE id > lastID uses index to jump directly (O(log n))
	start := sort.Search(len(data), func(i int) bool {
		return data[i].ID > lastID
	})
	if start >= len(data) {
		return nil
	}
	end := start + pageSize
	if end > len(data) {
		end = len(data)
	}
	return data[start:end]
}

// BuildINClause builds a parameterized IN clause
func BuildINClause(ids []int) (string, []interface{}) {
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	return strings.Join(placeholders, ", "), args
}

// --- Data Generation ---

func generateUsers(n int) []User {
	users := make([]User, n)
	now := time.Now()
	for i := range users {
		users[i] = User{
			ID:        i + 1,
			Email:     fmt.Sprintf("user%d@example.com", i),
			Username:  fmt.Sprintf("user%d", i),
			FullName:  fmt.Sprintf("User Number %d", i),
			Password:  "$2a$10$hashedpasswordhere",
			Bio:       "This is a bio that takes up some space in memory",
			Avatar:    fmt.Sprintf("https://cdn.example.com/avatars/%d.jpg", i),
			Phone:     fmt.Sprintf("+628%010d", i),
			Address:   fmt.Sprintf("%d Main Street, Building %d", i*10, i),
			City:      "Jakarta",
			Country:   "Indonesia",
			Timezone:  "Asia/Jakarta",
			Language:  "id",
			CreatedAt: now.Add(-time.Duration(rand.Intn(365*24)) * time.Hour),
			UpdatedAt: now,
		}
	}
	return users
}

func generateOrders(userIDs []int, ordersPerUser int) map[int][]Order {
	result := make(map[int][]Order, len(userIDs))
	orderID := 1
	for _, uid := range userIDs {
		orders := make([]Order, ordersPerUser)
		for i := range orders {
			orders[i] = Order{
				ID:     orderID,
				UserID: uid,
				Amount: float64(rand.Intn(10000)) / 100,
				Status: "completed",
			}
			orderID++
		}
		result[uid] = orders
	}
	return result
}
