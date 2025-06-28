package core

import (
	"slices"
	"strconv"
)

type Platform int8

const (
	Unspecify  Platform = 0
	Android    Platform = 1
	AndroidPad Platform = 2
	IPhone     Platform = 3
	Mac        Platform = 4
	IPad       Platform = 5
	Windows    Platform = 6
	Linux      Platform = 7
	Web        Platform = 8
	Harmony    Platform = 9
)

var platforms = []Platform{Android, AndroidPad, IPhone, IPad, Mac, IPad, Windows, Linux, Web, Harmony}

func (p Platform) IsValid() bool {
	return slices.Contains(platforms, p)
}

func (p Platform) String() string {
	switch p {
	case Android:
		return "android"
	case AndroidPad:
		return "androidpad"
	case IPhone:
		return "iphone"
	case IPad:
		return "ipad"
	case Mac:
		return "mac"
	case Windows:
		return "windows"
	case Linux:
		return "linux"
	case Web:
		return "web"
	case Harmony:
		return "harmony"
	}
	return "unspecify"
}

func PlatformFromString(pstr string) Platform {
	if len(pstr) == 0 {
		return Unspecify
	}

	p, err := strconv.Atoi(pstr)
	if err != nil {
		return Unspecify
	}
	pla := Platform(p)
	if pla.IsValid() {
		return pla
	}
	return Unspecify
}
