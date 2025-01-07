package util

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
	// "github.com/qiangli/ai/internal/log"
	// "github.com/qiangli/ai/internal/tool"
)

// GetFileSystemInfo retrieves information about the file system.
func GetFileSystemInfo(path string) (total uint64, free uint64, err error) {
	var stat syscall.Statfs_t

	err = syscall.Statfs(path, &stat)
	if err != nil {
		return 0, 0, err
	}

	// Calculate total space in bytes
	total = stat.Blocks * uint64(stat.Bsize)

	// Calculate free space in bytes
	free = stat.Bfree * uint64(stat.Bsize)

	return total, free, nil
}

func GetRunningProcesses() ([]string, error) {
	var processInfo []string

	// Get a list of all processes
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("error retrieving processes: %v", err)
	}

	// Iterate over each process and collect its details
	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			fmt.Printf("error retrieving process name: %v\n", err)
			name = "unknown"
		}

		pid := p.Pid

		info := fmt.Sprintf("Process Name: %s, PID: %d", name, pid)
		processInfo = append(processInfo, info)
	}

	return processInfo, nil
}

func CollectNetworkConfig() (map[string][]net.IP, error) {
	networkConfig := make(map[string][]net.IP)

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("error fetching interfaces: %v", err)
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("error fetching addresses for interface %s: %v", iface.Name, err)
		}

		var ips []net.IP
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ips = append(ips, v.IP)
			case *net.IPAddr:
				ips = append(ips, v.IP)
			}
		}
		networkConfig[iface.Name] = ips
	}

	return networkConfig, nil
}

type DiskInfo struct {
	Device      string
	MountPoint  string
	Filesystem  string
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

// GetDiskInfo retrieves info about the disk usage.
func GetDiskInfo() ([]DiskInfo, error) {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return nil, err
	}

	var diskInfos []DiskInfo
	for _, partition := range partitions {
		usageStat, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			fmt.Printf("error getting usage for partition %s: %v\n", partition.Mountpoint, err)
			continue
		}

		info := DiskInfo{
			Device:      partition.Device,
			MountPoint:  partition.Mountpoint,
			Filesystem:  partition.Fstype,
			Total:       usageStat.Total,
			Used:        usageStat.Used,
			Free:        usageStat.Free,
			UsedPercent: usageStat.UsedPercent,
		}

		diskInfos = append(diskInfos, info)
	}

	return diskInfos, nil
}

type MemoryStats struct {
	Total       uint64
	Available   uint64
	Used        uint64
	UsedPercent float64
}

// GetMemoryStats retrieves info about the memory usage.
func GetMemoryStats() (*MemoryStats, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("error fetching memory stats: %v", err)
	}

	stats := &MemoryStats{
		Total:       v.Total / 1024 / 1024,
		Available:   v.Available / 1024 / 1024,
		Used:        v.Used / 1024 / 1024,
		UsedPercent: v.UsedPercent,
	}

	return stats, nil
}

// CpuInfo holds information about a CPU.
type CpuInfo struct {
	CPU       int32
	VendorID  string
	ModelName string
	Cores     int32
	Mhz       float64
	Usage     []float64
}

// GetCpuInfo retrieves and returns information about the CPUs.
func GetCpuInfo() ([]CpuInfo, error) {
	cpus, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("error getting CPU info: %v", err)
	}

	// Get CPU usage statistics
	usage, err := cpu.Percent(0, true)
	if err != nil {
		return nil, fmt.Errorf("error getting CPU usage: %v", err)
	}

	cpuInfos := make([]CpuInfo, len(cpus))
	for i, c := range cpus {
		cpuInfos[i] = CpuInfo{
			CPU:       c.CPU,
			VendorID:  c.VendorID,
			ModelName: c.ModelName,
			Cores:     c.Cores,
			Mhz:       c.Mhz,
			Usage:     usage,
		}
	}

	return cpuInfos, nil
}

type ShellInfo struct {
	Name    string
	Path    string
	Version string
}

// GetShellVersion determines the current shell and retrieves its version.
func GetShellVersion() (*ShellInfo, error) {
	// Get the SHELL environment variable
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return nil, fmt.Errorf("unable to determine shell: SHELL environment variable is not set")
	}

	// Get the shell name from the path
	shellName := shellPath[strings.LastIndex(shellPath, "/")+1:]

	// Create a map to handle shell version commands
	shellVersionCommands := map[string]string{
		"bash": "bash --version",
		"zsh":  "zsh --version",
		"tsh":  "tsh --version",
		"sh":   "sh --version", // or "sh -c 'echo $0'"
		// Add other shells and their version commands as needed
	}

	// Find the version command for the current shell
	versionCommand, exists := shellVersionCommands[shellName]
	if !exists {
		return nil, fmt.Errorf("shell version command not defined for shell: %s", shellName)
	}

	// Execute the version command
	cmd := exec.Command("sh", "-c", versionCommand)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error executing version command for %s: %v", shellName, err)
	}
	lines := strings.SplitN(out.String(), "\n", 2)
	shellVersion := strings.TrimSpace(lines[0])
	return &ShellInfo{
		Name:    shellName,
		Path:    shellPath,
		Version: shellVersion,
	}, nil
}

// GetOSInfo collects OS-specific release info
func GetOSInfo() (map[string]string, error) {
	info := make(map[string]string)

	switch runtime.GOOS {
	case "darwin":
		// macOS
		name, err := exec.Command("sw_vers", "-productName").Output()
		if err != nil {
			return nil, err
		}
		version, err := exec.Command("sw_vers", "-productVersion").Output()
		if err != nil {
			return nil, err
		}
		build, err := exec.Command("sw_vers", "-buildVersion").Output()
		if err != nil {
			return nil, err
		}
		info["Name"] = strings.TrimSpace(string(name))
		info["Version"] = strings.TrimSpace(string(version))
		info["Build"] = strings.TrimSpace(string(build))

	case "linux":
		// Linux (read from /etc/os-release)
		file, err := os.Open("/etc/os-release")
		if err != nil {
			return nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := parts[0]
			value := strings.Trim(parts[1], `"`)
			info[key] = value
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}

	case "windows":
		// Windows
		version := fmt.Sprintf("%s %s", os.Getenv("OS"), os.Getenv("PROCESSOR_ARCHITECTURE"))
		info["Name"] = "Windows"
		info["Version"] = strings.TrimSpace(version)

	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return info, nil
}

// uname -sm
func Uname() (string, string) {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	return osName, arch
}

type SystemInfo struct {
	OS        string
	Arch      string
	OSInfo    map[string]string
	ShellInfo *ShellInfo

	EnvVarNames string
	WorkDir     string
}

func CollectSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	var errs []error

	var err error

	// Collect OS type and architecture
	info.OS, info.Arch = Uname()

	// Get OS info
	info.OSInfo, err = GetOSInfo()
	if err != nil {
		errs = append(errs, fmt.Errorf("error getting OS info: %v", err))
	}

	// Get shell info
	info.ShellInfo, err = GetShellVersion()
	if err != nil {
		errs = append(errs, fmt.Errorf("error getting shell version: %v", err))
	}

	// Collect environment variables
	info.EnvVarNames = GetEnvVarNames()

	// Get working directory
	info.WorkDir, err = Getwd()
	if err != nil {
		errs = append(errs, fmt.Errorf("error getting working directory: %v", err))
	}

	if len(errs) > 0 {
		return info, fmt.Errorf("errors occurred while collecting system info: %v", errs)
	}

	return info, nil
}
