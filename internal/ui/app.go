// Package ui provides the terminal user interface for the Docker stats monitor
package ui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/tradik/cv-xslt/scripts/tools/stats/internal/docker"
)

// App represents the main application
type App struct {
	client   *docker.Client
	interval time.Duration
	showAll  bool

	app       *tview.Application
	table     *tview.Table
	infoBar   *tview.TextView
	statusBar *tview.TextView

	containers []docker.ContainerStats
	sortField  docker.SortField
	sortAsc    bool
	mu         sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

// NewApp creates a new application instance
func NewApp(client *docker.Client, interval time.Duration, showAll bool) *App {
	ctx, cancel := context.WithCancel(context.Background())

	return &App{
		client:    client,
		interval:  interval,
		showAll:   showAll,
		sortField: docker.SortByCPU,
		sortAsc:   false,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Run starts the application
func (a *App) Run() error {
	a.app = tview.NewApplication()

	// Create UI components
	a.createUI()

	// Start background refresh after app starts
	a.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		return false
	})

	// Initial data load and start refresh loop
	go func() {
		// Small delay to let the app initialize
		time.Sleep(100 * time.Millisecond)
		a.refresh()
		a.refreshLoop()
	}()

	return a.app.Run()
}

// Stop stops the application
func (a *App) Stop() {
	a.cancel()
	a.app.Stop()
}

// createUI creates the user interface components
func (a *App) createUI() {
	// Info bar at top
	a.infoBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	a.infoBar.SetBorder(true).SetTitle(" Docker Info ")

	// Main table
	a.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	a.table.SetBorder(true).SetTitle(" Containers ")

	// Status bar at bottom
	a.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	a.statusBar.SetText("[yellow]q[white]:Quit  [yellow]r[white]:Refresh  [yellow]c[white]:Sort CPU  [yellow]m[white]:Sort Mem  [yellow]n[white]:Sort Name  [yellow]↑↓[white]:Navigate")

	// Layout
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.infoBar, 5, 0, false).
		AddItem(a.table, 0, 1, true).
		AddItem(a.statusBar, 1, 0, false)

	// Set up key bindings
	a.app.SetInputCapture(a.handleInput)

	a.app.SetRoot(flex, true)
}

// handleInput handles keyboard input
func (a *App) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlC:
		a.Stop()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'q', 'Q':
			a.Stop()
			return nil
		case 'r', 'R':
			go a.refresh()
			return nil
		case 'c', 'C':
			a.setSortField(docker.SortByCPU)
			return nil
		case 'm', 'M':
			a.setSortField(docker.SortByMemory)
			return nil
		case 'n', 'N':
			a.setSortField(docker.SortByName)
			return nil
		}
	}
	return event
}

// setSortField sets the sort field and refreshes the display
func (a *App) setSortField(field docker.SortField) {
	a.mu.Lock()
	if a.sortField == field {
		a.sortAsc = !a.sortAsc
	} else {
		a.sortField = field
		a.sortAsc = false
	}
	docker.SortContainers(a.containers, a.sortField, a.sortAsc)
	a.mu.Unlock()
	a.updateTable()
}

// refreshLoop periodically refreshes the statistics
func (a *App) refreshLoop() {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.refresh()
		}
	}
}

// refresh fetches new statistics and updates the display
func (a *App) refresh() {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	// Get Docker info
	info, err := a.client.GetDockerInfo(ctx)
	if err == nil {
		a.updateInfoBar(info)
	}

	// Get container stats
	containers, err := a.client.GetContainerStats(ctx, a.showAll)
	if err != nil {
		a.app.QueueUpdateDraw(func() {
			a.statusBar.SetText(fmt.Sprintf("[red]Error: %v", err))
		})
		return
	}

	a.mu.Lock()
	a.containers = containers
	docker.SortContainers(a.containers, a.sortField, a.sortAsc)
	a.mu.Unlock()

	a.updateTable()
}

// updateInfoBar updates the Docker info bar
func (a *App) updateInfoBar(info *docker.DockerInfo) {
	a.app.QueueUpdateDraw(func() {
		text := fmt.Sprintf(
			"[green]Docker %s[white] | Containers: [yellow]%d[white] running, [blue]%d[white] total | Images: [cyan]%d[white] (%s) | CPUs: [magenta]%d[white] | Memory: [cyan]%s[white] | %s/%s",
			info.ServerVersion,
			info.ContainersRunning,
			info.ContainersTotal,
			info.ImagesTotal,
			docker.FormatBytesInt64(info.TotalImageSize),
			info.CPUs,
			docker.FormatBytesInt64(info.MemoryTotal),
			info.OSType,
			info.Architecture,
		)
		a.infoBar.SetText(text)
	})
}

// updateTable updates the container table
func (a *App) updateTable() {
	a.app.QueueUpdateDraw(func() {
		a.table.Clear()

		// Header row
		headers := []string{"NAME", "STATUS", "CPU%", "MEM USAGE", "MEM%", "NET I/O", "BLOCK I/O", "PIDS", "IMAGE SIZE"}
		for col, header := range headers {
			cell := tview.NewTableCell(header).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft).
				SetSelectable(false).
				SetExpansion(1)
			if col == 0 {
				cell.SetExpansion(2)
			}
			a.table.SetCell(0, col, cell)
		}

		a.mu.RLock()
		defer a.mu.RUnlock()

		if len(a.containers) == 0 {
			cell := tview.NewTableCell("No containers found").
				SetTextColor(tcell.ColorGray).
				SetAlign(tview.AlignCenter).
				SetSelectable(false)
			a.table.SetCell(1, 0, cell)
			return
		}

		// Data rows
		for row, cont := range a.containers {
			// Name
			a.table.SetCell(row+1, 0, tview.NewTableCell(cont.Name).
				SetTextColor(tcell.ColorWhite).
				SetExpansion(2))

			// Status
			statusColor := tcell.ColorGreen
			if cont.State != "running" {
				statusColor = tcell.ColorGray
			}
			a.table.SetCell(row+1, 1, tview.NewTableCell(cont.State).
				SetTextColor(statusColor).
				SetExpansion(1))

			// CPU%
			cpuColor := getCPUColor(cont.CPUPercent)
			a.table.SetCell(row+1, 2, tview.NewTableCell(docker.FormatPercent(cont.CPUPercent)).
				SetTextColor(cpuColor).
				SetExpansion(1))

			// Memory Usage
			a.table.SetCell(row+1, 3, tview.NewTableCell(docker.FormatMemUsage(cont.MemUsage, cont.MemLimit)).
				SetTextColor(tcell.ColorWhite).
				SetExpansion(1))

			// Memory %
			memColor := getMemColor(cont.MemPercent)
			a.table.SetCell(row+1, 4, tview.NewTableCell(docker.FormatPercent(cont.MemPercent)).
				SetTextColor(memColor).
				SetExpansion(1))

			// Network I/O
			a.table.SetCell(row+1, 5, tview.NewTableCell(docker.FormatNetIO(cont.NetRx, cont.NetTx)).
				SetTextColor(tcell.ColorTeal).
				SetExpansion(1))

			// Block I/O
			a.table.SetCell(row+1, 6, tview.NewTableCell(docker.FormatBlockIO(cont.BlockRead, cont.BlockWrite)).
				SetTextColor(tcell.ColorBlue).
				SetExpansion(1))

			// PIDs
			a.table.SetCell(row+1, 7, tview.NewTableCell(fmt.Sprintf("%d", cont.PIDs)).
				SetTextColor(tcell.ColorWhite).
				SetExpansion(1))

			// Image Size
			a.table.SetCell(row+1, 8, tview.NewTableCell(docker.FormatBytesInt64(cont.ImageSize)).
				SetTextColor(tcell.ColorPurple).
				SetExpansion(1))
		}

		// Update title with count and last update time
		a.table.SetTitle(fmt.Sprintf(" Containers (%d) - Updated: %s ", len(a.containers), time.Now().Format("15:04:05")))
	})
}

// getCPUColor returns color based on CPU usage
func getCPUColor(percent float64) tcell.Color {
	switch {
	case percent >= 80:
		return tcell.ColorRed
	case percent >= 50:
		return tcell.ColorYellow
	case percent >= 20:
		return tcell.ColorGreen
	default:
		return tcell.ColorWhite
	}
}

// getMemColor returns color based on memory usage
func getMemColor(percent float64) tcell.Color {
	switch {
	case percent >= 90:
		return tcell.ColorRed
	case percent >= 70:
		return tcell.ColorYellow
	case percent >= 40:
		return tcell.ColorGreen
	default:
		return tcell.ColorWhite
	}
}
