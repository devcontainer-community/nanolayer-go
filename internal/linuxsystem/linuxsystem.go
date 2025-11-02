package linuxsystem

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

const OS_RELEASE_FILE = "/etc/os-release"

// Architecture represents the CPU architecture
type Architecture string

const (
	ARM64  Architecture = "arm64"
	X86_64 Architecture = "x86_64"
	ARMV5  Architecture = "armv5"
	ARMV6  Architecture = "armv6"
	ARMV7  Architecture = "armv7"
	ARMHF  Architecture = "armhf"
	ARM32  Architecture = "arm32"
	I386   Architecture = "i386"
	I686   Architecture = "i686"
	PPC64  Architecture = "ppc64"
	S390   Architecture = "s390"
	OTHER  Architecture = "other"
)

// LinuxReleaseID represents the Linux distribution
type LinuxReleaseID string

const (
	Ubuntu   LinuxReleaseID = "ubuntu"
	Debian   LinuxReleaseID = "debian"
	Alpine   LinuxReleaseID = "alpine"
	RHEL     LinuxReleaseID = "rhel"
	Fedora   LinuxReleaseID = "fedora"
	OpenSUSE LinuxReleaseID = "opensuse"
	Raspbian LinuxReleaseID = "raspbian"
	Manjaro  LinuxReleaseID = "manjaro"
	Arch     LinuxReleaseID = "arch"
	Unknown  LinuxReleaseID = "unknown"
)

func GetArchitecture() Architecture {
	var utsname unix.Utsname
	if err := unix.Uname(&utsname); err != nil {
		return OTHER
	}

	// Convert the byte array to string
	machine := make([]byte, 0, len(utsname.Machine))
	for _, b := range utsname.Machine {
		if b == 0 {
			break
		}
		machine = append(machine, byte(b))
	}

	arch := string(machine)

	// Map the architecture string to the enum
	switch arch {
	case "arm64", "aarch64":
		return ARM64
	case "x86_64", "amd64":
		return X86_64
	case "armv5":
		return ARMV5
	case "armv6":
		return ARMV6
	case "armv7", "armv7l":
		return ARMV7
	case "armhf":
		return ARMHF
	case "arm32":
		return ARM32
	case "i386":
		return I386
	case "i686":
		return I686
	case "ppc64", "ppc64le":
		return PPC64
	case "s390", "s390x":
		return S390
	default:
		return OTHER
	}
}

func GetDistribution() LinuxReleaseID {
	file, err := os.Open(OS_RELEASE_FILE)
	if err != nil {
		return Unknown
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Look for ID= line
		if strings.HasPrefix(line, "ID=") {
			// Remove ID= prefix and quotes
			id := strings.TrimPrefix(line, "ID=")
			id = strings.Trim(id, "\"'")
			id = strings.ToLower(id)

			// Map to enum
			switch id {
			case "ubuntu":
				return Ubuntu
			case "debian":
				return Debian
			case "alpine":
				return Alpine
			case "rhel":
				return RHEL
			case "fedora":
				return Fedora
			case "opensuse", "opensuse-leap", "opensuse-tumbleweed":
				return OpenSUSE
			case "raspbian":
				return Raspbian
			case "manjaro":
				return Manjaro
			case "arch":
				return Arch
			default:
				return Unknown
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return Unknown
	}

	return Unknown
}

func IsLinux() bool {
	var utsname unix.Utsname
	if err := unix.Uname(&utsname); err != nil {
		return false
	}

	// Convert the sysname to string
	sysname := make([]byte, 0, len(utsname.Sysname))
	for _, b := range utsname.Sysname {
		if b == 0 {
			break
		}
		sysname = append(sysname, byte(b))
	}
	fmt.Printf("sysname: %s\n", string(sysname))
	return string(sysname) == "Linux"
}

func HasRootPrivileges() bool {
	return os.Geteuid() == 0 || os.Getenv("SUDO_UID") != ""
}
