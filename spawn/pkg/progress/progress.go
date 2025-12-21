package progress

import (
	"fmt"
	"runtime"
	"time"
)

// Progress tracks and displays spawn progress
type Progress struct {
	steps       []Step
	currentStep int
}

// Step represents a single step in the spawn process
type Step struct {
	Name      string
	Status    string // pending, running, complete, error
	StartTime time.Time
	EndTime   time.Time
}

// NewProgress creates a new progress tracker
func NewProgress() *Progress {
	return &Progress{
		steps: []Step{
			{Name: "Detecting AMI", Status: "pending"},
			{Name: "Setting up SSH key", Status: "pending"},
			{Name: "Setting up IAM role", Status: "pending"},
			{Name: "Creating security group", Status: "pending"},
			{Name: "Launching instance", Status: "pending"},
			{Name: "Installing spawnd agent", Status: "pending"},
			{Name: "Waiting for instance", Status: "pending"},
			{Name: "Getting public IP", Status: "pending"},
			{Name: "Waiting for SSH", Status: "pending"},
		},
		currentStep: 0,
	}
}

// Start marks a step as started
func (p *Progress) Start(stepName string) {
	for i := range p.steps {
		if p.steps[i].Name == stepName {
			p.steps[i].Status = "running"
			p.steps[i].StartTime = time.Now()
			p.currentStep = i
			p.display()
			return
		}
	}
}

// Complete marks a step as complete
func (p *Progress) Complete(stepName string) {
	for i := range p.steps {
		if p.steps[i].Name == stepName {
			p.steps[i].Status = "complete"
			p.steps[i].EndTime = time.Now()
			p.display()
			return
		}
	}
}

// Error marks a step as errored
func (p *Progress) Error(stepName string, err error) {
	for i := range p.steps {
		if p.steps[i].Name == stepName {
			p.steps[i].Status = "error"
			p.steps[i].EndTime = time.Now()
			p.display()
			fmt.Println()
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
	}
}

// Skip marks a step as skipped
func (p *Progress) Skip(stepName string) {
	for i := range p.steps {
		if p.steps[i].Name == stepName {
			p.steps[i].Status = "skipped"
			p.display()
			return
		}
	}
}

// display shows the current progress
func (p *Progress) display() {
	// Clear screen and move cursor to top
	if runtime.GOOS == "windows" {
		// Windows: just print progress without clearing
		fmt.Println()
	} else {
		// Unix: clear and redraw
		fmt.Print("\033[2J\033[H")
	}

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║  🚀 Spawning Instance...                               ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	for i, step := range p.steps {
		symbol := getSymbol(step.Status)
		duration := ""

		if step.Status == "complete" && !step.StartTime.IsZero() && !step.EndTime.IsZero() {
			elapsed := step.EndTime.Sub(step.StartTime)
			if elapsed < time.Second {
				duration = fmt.Sprintf(" (%.0fms)", elapsed.Seconds()*1000)
			} else {
				duration = fmt.Sprintf(" (%.1fs)", elapsed.Seconds())
			}
		} else if step.Status == "running" && !step.StartTime.IsZero() {
			elapsed := time.Since(step.StartTime)
			duration = fmt.Sprintf(" (%.1fs)", elapsed.Seconds())
		}

		// Highlight current step
		if i == p.currentStep && step.Status == "running" {
			fmt.Printf("  %s %s...%s\n", symbol, step.Name, duration)
		} else {
			fmt.Printf("  %s %s%s\n", symbol, step.Name, duration)
		}
	}

	fmt.Println()
}

// DisplaySuccess shows the final success message
func (p *Progress) DisplaySuccess(instanceID, publicIP, sshCommand string, config interface{}) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║  🎉 Instance Ready!                                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Instance Details:")
	fmt.Println()
	fmt.Printf("  Instance ID:  %s\n", instanceID)
	fmt.Printf("  Public IP:    %s\n", publicIP)
	fmt.Printf("  Status:       running\n")
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("🔌 Connect Now:")
	fmt.Println()
	fmt.Printf("  %s\n", sshCommand)
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Show monitoring info if applicable
	if launchConfig, ok := config.(LaunchConfigInterface); ok {
		if launchConfig.GetTTL() != "" || launchConfig.GetIdleTimeout() != "" {
			fmt.Println("💡 Automatic Monitoring:")
			fmt.Println()
			if ttl := launchConfig.GetTTL(); ttl != "" {
				fmt.Printf("   ⏰ Will terminate after: %s\n", ttl)
			}
			if idle := launchConfig.GetIdleTimeout(); idle != "" {
				fmt.Printf("   💤 Will terminate if idle: %s\n", idle)
			}
			fmt.Println()
			fmt.Println("   The spawnd agent is monitoring your instance.")
			fmt.Println("   You can close your laptop - it will handle everything!")
			fmt.Println()
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

// LaunchConfigInterface defines methods for getting launch config details
type LaunchConfigInterface interface {
	GetTTL() string
	GetIdleTimeout() string
}

func getSymbol(status string) string {
	switch status {
	case "pending":
		return "⏸️ "
	case "running":
		return "⏳"
	case "complete":
		return "✅"
	case "error":
		return "❌"
	case "skipped":
		return "⏭️ "
	default:
		return "  "
	}
}

// Spinner shows a simple spinner for long operations
type Spinner struct {
	message string
	frames  []string
	stop    chan bool
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		stop:    make(chan bool),
	}
}

// Start starts the spinner
func (s *Spinner) Start() {
	go func() {
		i := 0
		for {
			select {
			case <-s.stop:
				return
			default:
				fmt.Printf("\r  %s %s", s.frames[i], s.message)
				i = (i + 1) % len(s.frames)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop(success bool) {
	s.stop <- true
	if success {
		fmt.Printf("\r  ✅ %s\n", s.message)
	} else {
		fmt.Printf("\r  ❌ %s\n", s.message)
	}
}
