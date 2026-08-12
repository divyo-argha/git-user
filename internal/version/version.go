package version

// Version is the current fallback version of git-user.
// Release builds override the build version at compile time via
//
//	-ldflags="-X main.buildVersion=vX.Y.Z"  (see cmd/git-user/main.go)
var Version = "v4.7.6"

var BuildVersion = ""

func GetVersion() string {
	if BuildVersion != "" && BuildVersion != "dev" {
		return BuildVersion
	}
	return Version
}
