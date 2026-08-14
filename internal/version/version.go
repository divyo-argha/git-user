package version

var Version = "v4.7.9"

var BuildVersion = ""

func GetVersion() string {
	if BuildVersion != "" && BuildVersion != "dev" {
		return BuildVersion
	}
	return Version
}
