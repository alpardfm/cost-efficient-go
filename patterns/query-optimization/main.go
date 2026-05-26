package main

import (
	"fmt"
	"math/rand"
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

// SimulateOffsetPagination uses OFFSET (slow for deep pages)
func SimulateOffsetPagination(data []User, page, pageSize int) []User {
	offset := (page - 1) * pageSize
	if offset >= len(data) {
		return nil
	}
	end := offset + pageSize
	if end > len(data) {
		end = len(data)
	}
	// In real DB: must scan and discard `offset` rows first
	return data[offset:end]
}

// SimulateKeysetPagination uses WHERE id > lastID (fast for any page)
func SimulateKeysetPagination(data []User, lastID, pageSize int) []User {
	// In real DB: uses index to jump directly to lastID
	start := 0
	for i, u := range data {
		if u.ID > lastID {
			start = i
			break
		}
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

func main() {
	fmt.Println("=== Query Optimization & Indexing ===")
	fmt.Println()

	users := generateUsers(10000)
	userIDs := make([]int, 100)
	for i := range userIDs {
		userIDs[i] = i + 1
	}
	ordersByUser := generateOrders(userIDs, 5)

	// 1. SELECT * vs SELECT specific
	fmt.Println("--- SELECT * vs SELECT specific (10K rows) ---")
	start := time.Now()
	allUsers := SimulateSelectStar(users)
	starDuration := time.Since(start)

	start = time.Now()
	summaries := SimulateSelectSpecific(users)
	specificDuration := time.Since(start)

	fmt.Printf("SELECT * (15 columns):     %v, result size: %d bytes/row\n", starDuration, 400) // approximate
	fmt.Printf("SELECT id,email,name:      %v, result size: %d bytes/row\n", specificDuration, 48)
	fmt.Printf("Speedup: %.1fx faster\n", float64(starDuration)/float64(specificDuration))
	fmt.Println()
	_ = allUsers
	_ = summaries

	// 2. N+1 vs Batch
	fmt.Println("--- N+1 vs Batch Query (100 users, 5 orders each) ---")
	start = time.Now()
	for range 1000 {
		SimulateNPlusOne(userIDs, ordersByUser)
	}
	nplusDuration := time.Since(start) / 1000

	start = time.Now()
	for range 1000 {
		SimulateBatchQuery(userIDs, ordersByUser)
	}
	batchDuration := time.Since(start) / 1000

	fmt.Printf("N+1 (100 queries):  %v\n", nplusDuration)
	fmt.Printf("Batch (1 query):    %v\n", batchDuration)
	fmt.Printf("In real DB: N+1 = 100 round trips × ~1ms = ~100ms\n")
	fmt.Printf("In real DB: Batch = 1 round trip × ~2ms = ~2ms (50x faster)\n")
	fmt.Println()

	// 3. Offset vs Keyset pagination
	fmt.Println("--- Offset vs Keyset Pagination (page 500 of 10K rows, pageSize=20) ---")
	start = time.Now()
	for range 10000 {
		SimulateOffsetPagination(users, 500, 20)
	}
	offsetDuration := time.Since(start) / 10000

	start = time.Now()
	for range 10000 {
		SimulateKeysetPagination(users, 9980, 20)
	}
	keysetDuration := time.Since(start) / 10000

	fmt.Printf("OFFSET %d LIMIT 20:        %v (must skip %d rows)\n", 499*20, offsetDuration, 499*20)
	fmt.Printf("WHERE id > 9980 LIMIT 20:  %v (index seek)\n", keysetDuration)
	fmt.Printf("In real DB: OFFSET scales O(n), keyset is O(1) via index\n")
	fmt.Println()

	// 4. IN clause building
	fmt.Println("--- Parameterized IN Clause ---")
	clause, args := BuildINClause(userIDs[:5])
	fmt.Printf("SELECT * FROM orders WHERE user_id IN (%s)\n", clause)
	fmt.Printf("Args: %v\n", args)
}
