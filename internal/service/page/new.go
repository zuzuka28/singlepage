package page

// New constructs a page service that uses the system clock.
func New(repo Repository, config Config) *Service {
	return NewWithClock(repo, config, realClock{})
}
