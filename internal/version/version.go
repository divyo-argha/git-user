package version

var Version = "v4.7.8"

var BuildVersion = ""

func GetVersion() string {
	if BuildVersion != "" && BuildVersion != "dev" {
		return BuildVersion
	}
	return Version
}
