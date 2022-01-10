package version

import (
	"fmt"
	"runtime"
)

var (
	// NOTE: The $Format strings are replaced during 'git archive' thanks to the
	// companion .gitattributes file containing 'export-subst' in this same
	// directory.  See also https://git-scm.com/docs/gitattributes
	GitVersion = "v0.0.0-master+unknown"
	GitCommit  = "unknown" // sha1 from git, output of $(git rev-parse HEAD)

	BuildDate = "1970-01-01T00:00:00Z" // build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
)

// Info holds build information
type Info struct {
	GitCommit  string `json:"gitCommit"`
	GitVersion string `json:"gitVersion"`
	BuildDate  string `json:"buildDate"`
	GoVersion  string `json:"goVersion"`
	Compiler   string `json:"compiler"`
	Platform   string `json:"platform"`
}

// Get creates and initialized Info object
func Get() Info {
	return Info{
		GitCommit:  GitCommit,
		GitVersion: GitVersion,
		BuildDate:  BuildDate,
		GoVersion:  runtime.Version(),
		Compiler:   runtime.Compiler,
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
