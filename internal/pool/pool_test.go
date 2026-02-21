package pool

import (
	"sync"
	"testing"
	"time"
)

type TestUser struct {
	ID         int
	Name       string
	Tags       []string
	resetCount int
}

func (u *TestUser) Reset() {
	u.ID = 0
	u.Name = ""
	u.Tags = u.Tags[:0]
	u.resetCount++
}

type TestConfig struct {
	Host    string
	Port    int
	Enabled bool
	calls   int
}

func (c *TestConfig) Reset() {
	c.Host = ""
	c.Port = 0
	c.Enabled = false
	c.calls++
}

func TestPoolBasic(t *testing.T) {
	pool := New(func() *TestUser {
		return &TestUser{}
	})

	user := pool.Get()
	if user.ID != 0 || user.Name != "" || len(user.Tags) != 0 {
		t.Error("Get() must return empty object")
	}

	user.ID = 42
	user.Name = "Alice"
	user.Tags = []string{"admin", "user"}

	resetCountBefore := user.resetCount

	pool.Put(user)

	user2 := pool.Get()

	if user2.resetCount != resetCountBefore+1 {
		t.Errorf("Reset() must be called on Put(), counter: %d -> %d", resetCountBefore, user2.resetCount)
	}

	if user2.ID != 0 || user2.Name != "" || len(user2.Tags) != 0 {
		t.Errorf("Object must be resetted: %+v", user2)
	}
}

func TestPoolReuse(t *testing.T) {
	creationCount := 0
	pool := New(func() *TestConfig {
		creationCount++
		return &TestConfig{}
	})

	config1 := pool.Get()
	config1.Host = "example.com"
	config1.Port = 9000
	config1.Enabled = true

	pool.Put(config1)

	config2 := pool.Get()

	if config2.Host != "" || config2.Port != 0 || config2.Enabled != false {
		t.Errorf("Object must be resetted: %+v", config2)
	}

	if creationCount != 1 {
		t.Errorf("Only 1 object must be instantiated, but instantiated: %d", creationCount)
	}
}

func TestPoolConcurrent(t *testing.T) {
	pool := New(func() *TestUser {
		return &TestUser{}
	})

	const goroutines = 10
	const iterations = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	errors := make(chan error, goroutines*iterations)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				user := pool.Get()

				if user.ID != 0 || user.Name != "" || len(user.Tags) != 0 {
					errors <- &testError{
						goroutine: id,
						iteration: j,
						message:   "object is not resetted",
						user:      *user,
					}
				}

				user.ID = id
				user.Name = "User"
				user.Tags = []string{"tag1", "tag2"}

				pool.Put(user)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

type testError struct {
	goroutine int
	iteration int
	message   string
	user      TestUser
}

func (e *testError) Error() string {
	return e.message
}

func TestPoolResetOnPut(t *testing.T) {
	pool := New(func() *TestConfig {
		return &TestConfig{}
	})

	config := pool.Get()
	config.Host = "test.com"
	config.Port = 1234
	config.Enabled = true

	callsBefore := config.calls

	pool.Put(config)

	config2 := pool.Get()

	if config2.calls != callsBefore+1 {
		t.Errorf("Reset() must be called on Put(), counter: %d -> %d", callsBefore, config2.calls)
	}

	if config2.Host != "" || config2.Port != 0 || config2.Enabled != false {
		t.Error("Values must be resetted")
	}
}

func TestPoolMultipleTypes(t *testing.T) {
	userPool := New(func() *TestUser {
		return &TestUser{}
	})

	configPool := New(func() *TestConfig {
		return &TestConfig{}
	})

	user := userPool.Get()
	if user.ID != 0 {
		t.Error("User must have ID = 0 after Get")
	}
	user.ID = 100
	userPool.Put(user)

	user2 := userPool.Get()
	if user2.ID != 0 {
		t.Error("User must be resetted")
	}

	// Тестируем configPool
	config := configPool.Get()
	if config.Port != 0 {
		t.Error("Config must have Port = 0 after Get")
	}
	config.Port = 80
	configPool.Put(config)

	config2 := configPool.Get()
	if config2.Port != 0 {
		t.Error("Config must be resetted")
	}
}

func TestPoolEmpty(t *testing.T) {
	creationCount := 0
	pool := New(func() *TestConfig {
		creationCount++
		return &TestConfig{Host: "new"}
	})

	configs := make([]*TestConfig, 10)
	for i := range configs {
		configs[i] = pool.Get()
		if configs[i].Host != "" {
			t.Errorf("Object must be resetted in Get, Host: %s", configs[i].Host)
		}
	}

	if creationCount != 10 {
		t.Errorf("10 ojbcets must be instantiated, but instantiated: %d", creationCount)
	}

	for _, config := range configs {
		pool.Put(config)
	}

	lastConfig := pool.Get()
	if creationCount != 10 {
		t.Errorf("No new object must be instantiated, but instantiated: %d", creationCount)
	}

	if lastConfig.Host != "" {
		t.Errorf("Object must be resetted, Host: %s", lastConfig.Host)
	}
}

func TestPoolStress(t *testing.T) {
	pool := New(func() *TestUser {
		return &TestUser{}
	})

	const operations = 1000
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < operations/10; j++ {
				user := pool.Get()
				if user.ID != 0 && user.ID != -1 {
					t.Errorf("Unexpected ID: %d", user.ID)
				}
				user.ID = j
				pool.Put(user)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestPoolWithInitialValues(t *testing.T) {
	pool := New(func() *TestConfig {
		return &TestConfig{
			Host:    "default",
			Port:    8080,
			Enabled: true,
		}
	})

	config := pool.Get()

	if config.Host != "" || config.Port != 0 || config.Enabled != false {
		t.Errorf("Inizial values must be resetted: %+v", config)
	}

	config.Host = "custom"
	config.Port = 9000

	pool.Put(config)

	config2 := pool.Get()

	if config2.Host != "" || config2.Port != 0 || config2.Enabled != false {
		t.Errorf("Values must be resetted: %+v", config2)
	}
}

func TestPoolInterfaceConstraint(t *testing.T) {
	_ = New(func() *TestUser {
		return &TestUser{}
	})

	_ = New(func() *TestConfig {
		return &TestConfig{}
	})

	t.Log("Interface restrictions are working properly")
}

func TestPoolRaceConditions(t *testing.T) {
	pool := New(func() *TestUser {
		return &TestUser{}
	})

	const workers = 5
	const iterations = 100

	results := make(chan int, workers*iterations)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				user := pool.Get()
				results <- user.ID
				user.ID = workerID
				pool.Put(user)
			}
		}(w)
	}

	wg.Wait()
	close(results)

	nonZeroCount := 0
	for id := range results {
		if id != 0 {
			nonZeroCount++
		}
	}

	if nonZeroCount > 0 {
		t.Errorf("Found %d not resetted ID", nonZeroCount)
	}
}

func TestPoolPerformance(t *testing.T) {
	pool := New(func() *TestUser {
		return &TestUser{}
	})

	start := time.Now()
	const operations = 10000

	for i := 0; i < operations; i++ {
		user := pool.Get()
		user.ID = i
		pool.Put(user)
	}

	elapsed := time.Since(start)
	t.Logf("Done %d operations %v (%.0f ops/sec)",
		operations, elapsed, float64(operations)/elapsed.Seconds())
}
