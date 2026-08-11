package version

// Version is the current fallback version of git-user.
// It can be overridden at build time using:
//	-ldflags="-X github.com/divyo-argha/git-user/internal/version.Version=vX.Y.Z"
var Version = "v4.7.3"

var BuildVersion = ""

func GetVersion() string {
	if BuildVersion != "" && BuildVersion != "dev" {
		return BuildVersion
	}
	return Version
}
