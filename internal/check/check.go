package check

// Result contains the evaluated outcome of a single health check.
type Result struct {
	Name    string
	Status  Status
	Message string
	Err     error
}

// Checker is the interface that all health checks must implement.
// In Go, any struct implementing these two methods implicitly satisfies Checker.
type Checker interface {
	Name() string
	Check() Result
}

// Runner manages and executes a collection of health checks.
type Runner struct {
	checkers []Checker
}

// NewRunner creates a new Runner with the provided slice of checkers.
func NewRunner(checkers ...Checker) *Runner {
	return &Runner{
		checkers: checkers,
	}
}

// Add appends an additional checker to the runner.
func (r *Runner) Add(c Checker) {
	r.checkers = append(r.checkers, c)
}

// RunAll executes every registered checker in order and returns their results.
func (r *Runner) RunAll() []Result {
	results := make([]Result, 0, len(r.checkers))
	for _, c := range r.checkers {
		results = append(results, c.Check())
	}
	return results
}

// OverallStatus returns the highest severity status among a slice of results.
// If the results slice is empty, it returns StatusOK.
func OverallStatus(results []Result) Status {
	if len(results) == 0 {
		return StatusOK
	}

	highest := StatusOK
	for _, res := range results {
		if res.Status.Severity() > highest.Severity() {
			highest = res.Status
		}
	}

	return highest
}
