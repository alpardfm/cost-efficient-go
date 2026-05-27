// Package main demonstrates query optimization patterns.
// This is the educational example code showing the cost-efficiency pattern.
//
// Run with: go run ./patterns/query-optimization/examples/
package main

import (
	"fmt"
	"time"

	query "github.com/alpardfm/cost-efficient-go/patterns/query-optimization"
)

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
	allUsers := query.SimulateSelectStar(users)
	starDuration := time.Since(start)

	start = time.Now()
	summaries := query.SimulateSelectSpecific(users)
	specificDuration := time.Since(start)

	fmt.Printf("SELECT * (15 columns):     %v, result size: %d bytes/row\n", starDuration, 400)
	fmt.Printf("SELECT id,email,name:      %v, result size: %d bytes/row\n", specificDuration, 48)
	fmt.Printf("Speedup: %.1fx faster\n", float64(starDuration)/float64(specificDuration))
	fmt.Println()
	_ = allUsers
	_ = summaries

	// 2. N+1 vs Batch (with simulated network latency)
	fmt.Println("--- N+1 vs Batch Query (100 users, 5 orders each) ---")
	networkLatency := 1 * time.Millisecond

	start = time.Now()
	query.SimulateNPlusOneWithLatency(userIDs, ordersByUser, networkLatency)
	nplusDuration := time.Since(start)

	start = time.Now()
	query.SimulateBatchQueryWithLatency(userIDs, ordersByUser, 2*networkLatency)
	batchDuration := time.Since(start)

	fmt.Printf("N+1 (100 queries × 1ms latency):  %v\n", nplusDuration)
	fmt.Printf("Batch (1 query × 2ms latency):    %v\n", batchDuration)
	fmt.Printf("Speedup: %.1fx faster with batch\n", float64(nplusDuration)/float64(batchDuration))
	fmt.Printf("Explanation: N+1 = 100 round trips × 1ms = ~100ms, Batch = 1 round trip × 2ms = ~2ms\n")
	fmt.Println()

	// 3. Offset vs Keyset pagination
	fmt.Println("--- Offset vs Keyset Pagination (page 500 of 10K rows, pageSize=20) ---")
	start = time.Now()
	for range 10000 {
		query.SimulateOffsetPagination(users, 500, 20)
	}
	offsetDuration := time.Since(start) / 10000

	start = time.Now()
	for range 10000 {
		query.SimulateKeysetPagination(users, 9980, 20)
	}
	keysetDuration := time.Since(start) / 10000

	fmt.Printf("OFFSET %d LIMIT 20:        %v (must skip %d rows)\n", 499*20, offsetDuration, 499*20)
	fmt.Printf("WHERE id > 9980 LIMIT 20:  %v (binary search index seek)\n", keysetDuration)
	fmt.Printf("In real DB: OFFSET scales O(n), keyset is O(log n) via B-tree index\n")
	fmt.Println()

	// 4. IN clause building
	fmt.Println("--- Parameterized IN Clause ---")
	clause, args := query.BuildINClause(userIDs[:5])
	fmt.Printf("SELECT * FROM orders WHERE user_id IN (%s)\n", clause)
	fmt.Printf("Args: %v\n", args)
}

// generateUsers creates N test users for the example.
func generateUsers(n int) []query.User {
	users := make([]query.User, n)
	now := time.Now()
	for i := range users {
		users[i] = query.User{
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
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	return users
}

// generateOrders creates test orders for the example.
func generateOrders(userIDs []int, ordersPerUser int) map[int][]query.Order {
	result := make(map[int][]query.Order, len(userIDs))
	orderID := 1
	for _, uid := range userIDs {
		orders := make([]query.Order, ordersPerUser)
		for i := range orders {
			orders[i] = query.Order{
				ID:     orderID,
				UserID: uid,
				Amount: float64(orderID) * 1.5,
				Status: "completed",
			}
			orderID++
		}
		result[uid] = orders
	}
	return result
}
