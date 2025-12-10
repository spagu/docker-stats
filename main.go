// Package main provides a terminal-based Docker statistics monitor
// similar to 'top' or 'htop' for Linux systems.
//
// # Docker Stats Monitor
//
// A real-time terminal UI tool for monitoring Docker container statistics
// including CPU usage, memory consumption, network I/O, and disk usage.
//
// ## Features
//
//   - Real-time container statistics (CPU, Memory, Network, Disk)
//   - Terminal UI with keyboard navigation
//   - Sorting by different columns
//   - Auto-refresh with configurable interval
//   - Color-coded resource usage indicators
//
// ## Usage
//
//	./stats [flags]
//
// ## Flags
//
//	-interval duration    Refresh interval (default 2s)
//	-all                  Show all containers (including stopped)
//
// ## Keyboard Shortcuts
//
//	q, Ctrl+C    Quit
//	r            Force refresh
//	c            Sort by CPU
//	m            Sort by Memory
//	n            Sort by Name
//	↑/↓          Navigate containers
//
// ## Requirements
//
//   - Docker daemon running
//   - User must have permissions to access Docker socket
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tradik/cv-xslt/scripts/tools/stats/internal/docker"
	"github.com/tradik/cv-xslt/scripts/tools/stats/internal/ui"
)

// Version information
const (
	AppName    = "docker-stats"
	AppVersion = "1.0.0"
)

func main() {
	// Parse command line flags
	interval := flag.Duration("interval", 2*time.Second, "Refresh interval")
	showAll := flag.Bool("all", false, "Show all containers (including stopped)")
	simple := flag.Bool("simple", true, "Simple output mode (no TUI, like original bash script)")
	tui := flag.Bool("tui", false, "Use interactive TUI mode (requires full terminal)")
	once := flag.Bool("once", false, "Run once and exit (implies -simple)")
	version := flag.Bool("version", false, "Show version information")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	if *version {
		fmt.Printf("%s v%s\n", AppName, AppVersion)
		os.Exit(0)
	}

	// Create Docker client
	client, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to Docker: %v\n", err)
		fmt.Fprintln(os.Stderr, "Make sure Docker daemon is running and you have permissions to access it.")
		os.Exit(1)
	}
	defer client.Close() //nolint:errcheck // intentionally ignoring close error on exit

	// Simple mode or once mode (default), TUI only with -tui flag
	if (*simple && !*tui) || *once {
		runSimpleMode(client, *showAll, *once, *interval)
		return
	}

	// Create and run UI
	app := ui.NewApp(client, *interval, *showAll)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		app.Stop()
	}()

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`%s v%s - Docker Container Statistics Monitor

A real-time terminal UI tool for monitoring Docker container statistics.

USAGE:
    %s [OPTIONS]

OPTIONS:
    -interval duration    Refresh interval (default: 2s)
    -all                  Show all containers (including stopped)
    -version              Show version information
    -help                 Show this help message

KEYBOARD SHORTCUTS:
    q, Ctrl+C    Quit the application
    r            Force refresh statistics
    c            Sort by CPU usage
    m            Sort by Memory usage
    n            Sort by container Name
    ↑/↓          Navigate through containers
    Enter        Show container details

COLUMNS:
    NAME         Container name
    CPU%%         CPU usage percentage
    MEM USAGE    Memory usage (used / limit)
    MEM%%         Memory usage percentage
    NET I/O      Network input/output
    BLOCK I/O    Disk read/write
    PIDS         Number of processes
    IMAGE SIZE   Size of the container image

EXAMPLES:
    %s                    # Run with default settings
    %s -interval 5s       # Refresh every 5 seconds
    %s -all               # Show all containers

REQUIREMENTS:
    - Docker daemon must be running
    - User must have permissions to access Docker socket
      (typically member of 'docker' group or root)

`, AppName, AppVersion, AppName, AppName, AppName, AppName)
}

// Styles for the TUI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6")).
			Background(lipgloss.Color("0"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("7"))

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("8")).
			Foreground(lipgloss.Color("15"))

	greenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	redStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	cyanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	blueStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	magentaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	grayStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// Model for bubbletea
type statsModel struct {
	client     *docker.Client
	containers []docker.ContainerStats
	info       *docker.DockerInfo
	sortField  docker.SortField
	sortAsc    bool
	showAll    bool
	interval   time.Duration
	width      int
	height     int
	scroll     int
	selected   int
	err        error
	quitting   bool
}

type tickMsg time.Time
type containerMsg struct {
	containers []docker.ContainerStats
	info       *docker.DockerInfo
	err        error
}

func (m statsModel) Init() tea.Cmd {
	return tea.Batch(tickCmd(m.interval), fetchContainers(m.client, m.showAll))
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchContainers(client *docker.Client, showAll bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		containers, err := client.GetContainerStats(ctx, showAll)
		info, infoErr := client.GetDockerInfo(ctx)
		if infoErr != nil {
			info = nil
		}
		return containerMsg{containers: containers, info: info, err: err}
	}
}

func (m statsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "c":
			if m.sortField == docker.SortByCPU {
				m.sortAsc = !m.sortAsc
			} else {
				m.sortField = docker.SortByCPU
				m.sortAsc = false
			}
		case "m":
			if m.sortField == docker.SortByMemory {
				m.sortAsc = !m.sortAsc
			} else {
				m.sortField = docker.SortByMemory
				m.sortAsc = false
			}
		case "n":
			if m.sortField == docker.SortByName {
				m.sortAsc = !m.sortAsc
			} else {
				m.sortField = docker.SortByName
				m.sortAsc = true
			}
		case "d":
			if m.sortField == docker.SortByBlockIO {
				m.sortAsc = !m.sortAsc
			} else {
				m.sortField = docker.SortByBlockIO
				m.sortAsc = false
			}
		case "i":
			if m.sortField == docker.SortByImageSize {
				m.sortAsc = !m.sortAsc
			} else {
				m.sortField = docker.SortByImageSize
				m.sortAsc = false
			}
		case "up", "k":
			if m.selected > 0 {
				m.selected--
				if m.selected < m.scroll {
					m.scroll = m.selected
				}
			}
		case "down", "j":
			if m.selected < len(m.containers)-1 {
				m.selected++
				visibleRows := m.height - 10
				if visibleRows < 1 {
					visibleRows = 1
				}
				if m.selected >= m.scroll+visibleRows {
					m.scroll = m.selected - visibleRows + 1
				}
			}
		case "pgup":
			m.selected -= 10
			if m.selected < 0 {
				m.selected = 0
			}
			m.scroll = m.selected
		case "pgdown":
			m.selected += 10
			if m.selected >= len(m.containers) {
				m.selected = len(m.containers) - 1
			}
			visibleRows := m.height - 10
			if m.selected >= m.scroll+visibleRows {
				m.scroll = m.selected - visibleRows + 1
			}
		case "home":
			m.selected = 0
			m.scroll = 0
		case "end":
			m.selected = len(m.containers) - 1
			visibleRows := m.height - 10
			m.scroll = m.selected - visibleRows + 1
			if m.scroll < 0 {
				m.scroll = 0
			}
		case "r":
			return m, fetchContainers(m.client, m.showAll)
		}
		return m, nil

	case tickMsg:
		return m, tea.Batch(tickCmd(m.interval), fetchContainers(m.client, m.showAll))

	case containerMsg:
		m.containers = msg.containers
		m.info = msg.info
		m.err = msg.err
		docker.SortContainers(m.containers, m.sortField, m.sortAsc)
		// Keep selected in bounds
		if m.selected >= len(m.containers) {
			m.selected = len(m.containers) - 1
		}
		if m.selected < 0 {
			m.selected = 0
		}
		// Keep scroll in bounds
		visibleRows := m.height - 10
		if visibleRows < 1 {
			visibleRows = 1
		}
		maxScroll := len(m.containers) - visibleRows
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.scroll > maxScroll {
			m.scroll = maxScroll
		}
		if m.scroll < 0 {
			m.scroll = 0
		}
		return m, nil
	}

	return m, nil
}

func (m statsModel) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 {
		return "Loading..."
	}

	var s string

	// Header
	header := titleStyle.Render(fmt.Sprintf(" 🐳 DOCKER STATS %s ", AppVersion))
	if m.info != nil {
		header += dimStyle.Render(" │ ") + magentaStyle.Render(fmt.Sprintf("Docker %s", m.info.ServerVersion))
		header += dimStyle.Render(" │ ") + greenStyle.Render(fmt.Sprintf("%d", m.info.ContainersRunning)) + fmt.Sprintf("/%d", m.info.ContainersTotal)
		header += dimStyle.Render(" │ ") + cyanStyle.Render(fmt.Sprintf("%d imgs", m.info.ImagesTotal))
	}
	header += dimStyle.Render(" │ ") + yellowStyle.Render(time.Now().Format("15:04:05"))
	s += header + "\n"

	// Sort info and keys
	sortName := "CPU"
	switch m.sortField {
	case docker.SortByMemory:
		sortName = "MEM"
	case docker.SortByName:
		sortName = "NAME"
	case docker.SortByBlockIO:
		sortName = "DISK"
	case docker.SortByImageSize:
		sortName = "IMG"
	}
	sortDir := "↓"
	if m.sortAsc {
		sortDir = "↑"
	}
	s += dimStyle.Render("Sort: ") + yellowStyle.Render(sortName) + " " + sortDir
	s += dimStyle.Render("  │  ") + cyanStyle.Render("[c]") + "pu " + cyanStyle.Render("[m]") + "em " + cyanStyle.Render("[n]") + "ame " + cyanStyle.Render("[d]") + "isk " + cyanStyle.Render("[i]") + "mg"
	s += dimStyle.Render("  │  ") + cyanStyle.Render("[↑↓]") + "scroll " + cyanStyle.Render("[r]") + "efresh " + redStyle.Render("[q]") + "uit\n\n"

	// Calculate dynamic column widths
	// Find longest container name
	maxNameLen := 9 // minimum width (8 + 1 for truncation)
	for _, c := range m.containers {
		nameLen := len(c.Name)
		if nameLen > maxNameLen {
			maxNameLen = nameLen
		}
	}

	// Calculate remaining width for other columns
	otherColsWidth := 8 + 8 + 6 + 5 + 8 + 6 + 9 + 9 + 9 + 9 + 9 + 8 + 11 // spaces between columns
	maxNameWidth := m.width - otherColsWidth
	if maxNameWidth < 9 {
		maxNameWidth = 9
	}
	if maxNameWidth > 30 {
		maxNameWidth = 30 // cap at 30 to prevent too wide
	}

	colName := maxNameLen
	if colName > maxNameWidth {
		colName = maxNameWidth
	}
	if colName < 9 {
		colName = 9
	}

	// Fixed column widths
	const (
		colState  = 8
		colCpuBar = 8
		colCpuPct = 6
		colCpuLim = 5
		colMemBar = 8
		colMemPct = 6
		colMemUse = 9
		colNet    = 9
		colDisk   = 9
		colImg    = 8
	)

	// Table header - build manually for exact alignment
	hdr := fmt.Sprintf("%-*s", colName, "CONTAINER")
	hdr += fmt.Sprintf(" %-*s", colState, "STATE")
	hdr += fmt.Sprintf(" %-*s", colCpuBar+1+colCpuPct, "CPU")
	hdr += fmt.Sprintf(" %-*s", colCpuLim, "LIMIT")
	hdr += fmt.Sprintf(" %-*s", colMemBar+1+colMemPct, "MEMORY")
	hdr += fmt.Sprintf(" %-*s", colMemUse, "MEM USE")
	hdr += fmt.Sprintf(" %-*s", colMemUse, "MEM LIMIT")
	hdr += fmt.Sprintf(" %-*s", colNet, "NET RX")
	hdr += fmt.Sprintf(" %-*s", colNet, "NET TX")
	hdr += fmt.Sprintf(" %-*s", colDisk, "DISK R")
	hdr += fmt.Sprintf(" %-*s", colDisk, "DISK W")
	hdr += fmt.Sprintf(" %-*s", colImg, "IMAGE")
	s += headerStyle.Render(hdr) + "\n"
	s += dimStyle.Render(repeatStr("─", m.width)) + "\n"

	// Calculate visible rows - maximize to use full terminal height
	visibleRows := m.height - 8 // Reduce header/footer overhead
	if visibleRows < 1 {
		visibleRows = 1
	}

	// Containers already sorted in Update
	endIdx := m.scroll + visibleRows
	if endIdx > len(m.containers) {
		endIdx = len(m.containers)
	}

	for i := m.scroll; i < endIdx; i++ {
		c := m.containers[i]

		// Name - truncate to fit column
		name := c.Name
		if len(name) > colName {
			name = name[:colName-1] + "…"
		}

		// State
		stateStyle := greenStyle
		if c.State != "running" {
			stateStyle = grayStyle
		}

		// CPU bar
		cpuBar := makeBar(c.CPUPercent, colCpuBar)
		cpuStyle := cyanStyle
		if c.CPUPercent >= 80 {
			cpuStyle = redStyle
		} else if c.CPUPercent >= 50 {
			cpuStyle = yellowStyle
		} else if c.CPUPercent >= 20 {
			cpuStyle = greenStyle
		}

		// Memory bar
		memBar := makeBar(c.MemPercent, colMemBar)
		memStyle := cyanStyle
		if c.MemPercent >= 90 {
			memStyle = redStyle
		} else if c.MemPercent >= 70 {
			memStyle = yellowStyle
		} else if c.MemPercent >= 40 {
			memStyle = greenStyle
		}

		// Format CPU limit
		cpuLim := "∞"
		if c.CPULimit > 0 {
			cpuLim = fmt.Sprintf("%.1f", c.CPULimit)
		}

		// Format memory usage and limit separately
		memUse := docker.FormatBytes(c.MemUsage)
		memLim := docker.FormatBytes(c.MemLimit)

		// Build row with consistent spacing - pad BEFORE color
		row := fmt.Sprintf("%-*s", colName, name)
		row += fmt.Sprintf(" %s", stateStyle.Render(fmt.Sprintf("%-*s", colState, c.State)))
		row += fmt.Sprintf(" %s %s", cpuBar, cpuStyle.Render(fmt.Sprintf("%*s", colCpuPct, fmt.Sprintf("%5.1f%%", c.CPUPercent))))
		row += fmt.Sprintf(" %s", dimStyle.Render(fmt.Sprintf("%-*s", colCpuLim, cpuLim)))
		row += fmt.Sprintf(" %s %s", memBar, memStyle.Render(fmt.Sprintf("%*s", colMemPct, fmt.Sprintf("%.1f%%", c.MemPercent))))
		row += fmt.Sprintf(" %s", dimStyle.Render(fmt.Sprintf("%-*s", colMemUse, memUse)))
		row += fmt.Sprintf(" %s", dimStyle.Render(fmt.Sprintf("%-*s", colMemUse, memLim)))
		row += fmt.Sprintf(" %s", cyanStyle.Render(fmt.Sprintf("%-*s", colNet, docker.FormatBytes(c.NetRx))))
		row += fmt.Sprintf(" %s", cyanStyle.Render(fmt.Sprintf("%-*s", colNet, docker.FormatBytes(c.NetTx))))
		row += fmt.Sprintf(" %s", blueStyle.Render(fmt.Sprintf("%-*s", colDisk, docker.FormatBytes(c.BlockRead))))
		row += fmt.Sprintf(" %s", blueStyle.Render(fmt.Sprintf("%-*s", colDisk, docker.FormatBytes(c.BlockWrite))))
		row += fmt.Sprintf(" %s", magentaStyle.Render(fmt.Sprintf("%-*s", colImg, docker.FormatBytesInt64(c.ImageSize))))

		if i == m.selected {
			s += selectedStyle.Render(row) + "\n"
		} else {
			s += row + "\n"
		}
	}

	// Scroll indicator
	if len(m.containers) > visibleRows {
		scrollInfo := fmt.Sprintf(" [%d-%d of %d] ", m.scroll+1, endIdx, len(m.containers))
		s += dimStyle.Render(repeatStr("─", m.width)) + "\n"
		s += dimStyle.Render(scrollInfo) + "\n"
	}

	s += dimStyle.Render(repeatStr("═", m.width)) + "\n"
	s += dimStyle.Render(fmt.Sprintf("  ⟳ Auto-refresh: %s", m.interval.String()))

	return s
}

func makeBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	var style lipgloss.Style
	switch {
	case percent >= 90:
		style = redStyle
	case percent >= 70:
		style = yellowStyle
	case percent >= 40:
		style = greenStyle
	default:
		style = cyanStyle
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += style.Render("█")
		} else {
			bar += dimStyle.Render("░")
		}
	}
	return bar
}

func repeatStr(s string, n int) string {
	if n <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-1] + "…"
	}
	return s
}

// runSimpleMode runs the bubbletea TUI
func runSimpleMode(client *docker.Client, showAll, once bool, interval time.Duration) {
	if once {
		// Simple one-shot output without TUI
		ctx := context.Background()
		containers, err := client.GetContainerStats(ctx, showAll)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		info, infoErr := client.GetDockerInfo(ctx)
		if infoErr != nil {
			info = nil
		}

		fmt.Printf("DOCKER STATS %s | %s", AppVersion, time.Now().Format("15:04:05"))
		if info != nil {
			fmt.Printf(" | Docker %s | %d/%d containers | %d images",
				info.ServerVersion, info.ContainersRunning, info.ContainersTotal, info.ImagesTotal)
		}
		fmt.Println()
		fmt.Printf("%-20s  %-8s  %6s  %6s  %-18s  %-18s  %5s\n",
			"CONTAINER", "STATE", "CPU%", "MEM%", "NET I/O", "BLOCK I/O", "PID")
		fmt.Println(repeatStr("-", 100))

		docker.SortContainers(containers, docker.SortByCPU, false)
		for _, c := range containers {
			name := c.Name
			if len(name) > 18 {
				name = name[:17] + "…"
			}
			fmt.Printf("%-20s  %-8s  %5.1f%%  %5.1f%%  %-18s  %-18s  %5d\n",
				name, c.State, c.CPUPercent, c.MemPercent,
				truncate(docker.FormatNetIO(c.NetRx, c.NetTx), 18),
				truncate(docker.FormatBlockIO(c.BlockRead, c.BlockWrite), 18),
				c.PIDs)
		}
		return
	}

	// Run bubbletea TUI
	m := statsModel{
		client:    client,
		showAll:   showAll,
		interval:  interval,
		sortField: docker.SortByCPU,
		sortAsc:   false,
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Clear terminal after TUI exits
	fmt.Print("\033[2J\033[H\033[0m")
	fmt.Println()
}
