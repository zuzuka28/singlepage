package page

// NewWithClock constructs a page service with an explicit clock.
func NewWithClock(repo Repository, config Config, clock Clock) *Service {
	if clock == nil {
		clock = realClock{}
	}

	return &Service{repo: repo, config: config, clock: clock}
}
