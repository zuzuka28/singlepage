package page

const (
	maxSaltBytes       = 64
	maxCapabilityBytes = 256
)

type Config struct {
	MaxPages int64
}

type Service struct {
	repo   Repository
	config Config
	clock  Clock
}
