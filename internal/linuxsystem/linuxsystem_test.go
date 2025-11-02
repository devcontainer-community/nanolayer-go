package linuxsystem

import (
	"bufio"
	"os"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestGetArchitectureMatchesUname(t *testing.T) {
	want := expectedArchitectureFromUname(t)
	got := GetArchitecture()
	if got != want {
		t.Fatalf("GetArchitecture() = %q, want %q", got, want)
	}
}

func TestIsLinuxMatchesGOOS(t *testing.T) {
	got := IsLinux()
	want := runtime.GOOS == "linux"
	if got != want {
		t.Fatalf("IsLinux() = %v, want %v for GOOS=%s", got, want, runtime.GOOS)
	}
}

func TestHasRootPrivileges(t *testing.T) {
	t.Setenv("SUDO_UID", "")

	if os.Geteuid() == 0 {
		if !HasRootPrivileges() {
			t.Fatalf("expected HasRootPrivileges() to be true when running as root")
		}
	} else {
		if HasRootPrivileges() {
			t.Fatalf("expected HasRootPrivileges() to be false without root privileges")
		}

		t.Setenv("SUDO_UID", "1000")
		if !HasRootPrivileges() {
			t.Fatalf("expected HasRootPrivileges() to be true when SUDO_UID is set")
		}
	}
}

func TestGetDistributionMatchesOSRelease(t *testing.T) {
	want, err := distributionFromOSRelease()
	got := GetDistribution()

	if err != nil {
		if got != Unknown {
			t.Fatalf("GetDistribution() = %q, want Unknown when %s missing", got, OS_RELEASE_FILE)
		}
		return
	}

	if got != want {
		t.Fatalf("GetDistribution() = %q, want %q", got, want)
	}
}

func expectedArchitectureFromUname(t *testing.T) Architecture {
	var uts unix.Utsname
	if err := unix.Uname(&uts); err != nil {
		t.Fatalf("unix.Uname failed: %v", err)
	}

	machine := make([]byte, 0, len(uts.Machine))
	for _, b := range uts.Machine {
		if b == 0 {
			break
		}
		machine = append(machine, byte(b))
	}

	switch string(machine) {
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

func distributionFromOSRelease() (LinuxReleaseID, error) {
	file, err := os.Open(OS_RELEASE_FILE)
	if err != nil {
		return Unknown, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "ID=") {
			continue
		}

		id := strings.TrimPrefix(line, "ID=")
		id = strings.Trim(id, "\"'")
		id = strings.ToLower(id)

		switch id {
		case "ubuntu":
			return Ubuntu, nil
		case "debian":
			return Debian, nil
		case "alpine":
			return Alpine, nil
		case "rhel":
			return RHEL, nil
		case "fedora":
			return Fedora, nil
		case "opensuse", "opensuse-leap", "opensuse-tumbleweed":
			return OpenSUSE, nil
		case "raspbian":
			return Raspbian, nil
		case "manjaro":
			return Manjaro, nil
		case "arch":
			return Arch, nil
		default:
			return Unknown, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return Unknown, err
	}

	return Unknown, nil
}
