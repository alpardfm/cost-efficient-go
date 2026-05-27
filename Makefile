.PHONY: test bench-all property-test

# Run all tests in all pattern directories
test:
	@for dir in patterns/*/; do \
		echo "Testing $$dir..."; \
		go test -v ./$$dir; \
	done

# Run all benchmarks with -benchmem -benchtime=3s
bench-all:
	@for dir in patterns/*/; do \
		echo "Benchmarking $$dir..."; \
		go test -bench=. -benchmem -benchtime=3s ./$$dir; \
	done

# Run only property tests (TestProperty prefix)
property-test:
	@for dir in patterns/*/; do \
		echo "Property testing $$dir..."; \
		go test -v -run TestProperty ./$$dir; \
	done

# Run benchmark for a specific pattern (e.g., make bench-sync-pool)
bench-%:
	go test -bench=. -benchmem -benchtime=3s ./patterns/$*/
