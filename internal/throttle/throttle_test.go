package throttle

import "testing"

func TestApplyCriticalThrottle(t *testing.T) {
	c := New(250, 5, 200, 8)
	delay, depth, pages, workers := c.apply(Critical)
	if depth != 1 {
		t.Fatalf("expected depth 1, got %d", depth)
	}
	if workers != 1 {
		t.Fatalf("expected 1 worker, got %d", workers)
	}
	if delay > 2000 {
		t.Fatalf("expected delay capped at 2000ms, got %d", delay)
	}
	if pages < 10 {
		t.Fatalf("expected minimum pages 10, got %d", pages)
	}
}

func TestClassifyLevels(t *testing.T) {
	if classify(50, 50, 100) != Normal {
		t.Fatal("expected NORMAL")
	}
	if classify(95, 50, 100) != Critical {
		t.Fatal("expected CRITICAL for high CPU")
	}
	if classify(50, 50, MaxGoroutines) != Critical {
		t.Fatal("expected CRITICAL for goroutine cap")
	}
}
