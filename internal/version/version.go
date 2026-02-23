package version

// Build-time variables injected via ldflags:
//
//	-X github.com/MathTrail/mentor-api/internal/version.Version=<semver>
//	-X github.com/MathTrail/mentor-api/internal/version.Commit=<git-sha>
//	-X github.com/MathTrail/mentor-api/internal/version.Date=<build-date>
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)
