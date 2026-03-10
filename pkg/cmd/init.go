package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/nikitastrike/gios/pkg/config"
)

// UI Color Palette & Styles
var (
	hotPink  = lipgloss.Color("#FF06B7")
	cyan     = lipgloss.Color("#00FFFF")
	lime     = lipgloss.Color("#ADFF2F")
	darkGray = lipgloss.Color("#242424")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyan).
			MarginTop(1).
			MarginBottom(1).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(hotPink)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lime)

	focusedStyle = lipgloss.NewStyle().Foreground(cyan)
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	successStyle = lipgloss.NewStyle().Foreground(lime).Bold(true)
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
)

type initModel struct {
	step        int
	textInput   textinput.Model
	projectName string
	packageID   string
	version     string
	template    int // 1: CLI, 2: Daemon, 3: Tweak
	arch        int // 1: Legacy, 2: Modern
	deploy      int // 1: Wi-Fi, 2: USB
	ip          string
	quitting    bool
	done        bool
}

func initialInitModel() initModel {
	ti := textinput.New()
	ti.Placeholder = "Project Name"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20
	ti.PromptStyle = focusedStyle

	cwd, _ := os.Getwd()
	ti.SetValue(filepath.Base(cwd))

	return initModel{
		step:        0,
		textInput:   ti,
		projectName: filepath.Base(cwd),
		version:     "1.0.0",
		template:    1,
		arch:        2,
		deploy:      2,
		ip:          "127.0.0.1",
	}
}

func (m initModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m initModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			m.nextStep()
			if m.done {
				return m, tea.Quit
			}
			return m, nil
		}

		// Handle selection for step 1, 2, 3 (multiple choice)
		if m.step >= 1 && m.step <= 3 {
			switch msg.String() {
			case "1", "2", "3":
				if m.step == 1 { m.template = m.toInt(msg.String()) }
				if m.step == 2 { m.arch = m.toInt(msg.String()) }
				if m.step == 3 { m.deploy = m.toInt(msg.String()) }
				m.nextStep()
				return m, nil
			}
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *initModel) toInt(s string) int {
	if s == "1" { return 1 }
	if s == "2" { return 2 }
	if s == "3" { return 3 }
	return 1
}

func (m *initModel) nextStep() {
	switch m.step {
	case 0: // Name -> Template
		m.projectName = m.textInput.Value()
		m.packageID = "com.gios." + m.projectName
		m.step = 1
	case 1: // Template -> Arch
		m.step = 2
	case 2: // Arch -> Deploy
		m.step = 3
	case 3: // Deploy -> IP (if Wi-Fi)
		if m.deploy == 1 {
			m.textInput.SetValue("192.168.1.XX")
			m.textInput.Placeholder = "Device IP Address"
			m.textInput.Focus()
			m.step = 4
		} else {
			m.ip = "127.0.0.1"
			m.step = 5 // Confirm
		}
	case 4: // IP -> Confirm
		m.ip = m.textInput.Value()
		m.step = 5
	case 5: // Confirm -> Done
		m.done = true
	}
}

func (m initModel) View() string {
	if m.quitting {
		return "\nInitialization cancelled. Better luck next time!\n"
	}
	if m.done {
		return ""
	}

	header := titleStyle.Render("GIOS") + "\n"

	var body string
	switch m.step {
	case 0:
		body = headerStyle.Render("1. Give your project a name:") + "\n\n" + m.textInput.View()
	case 1:
		body = headerStyle.Render("2. Select Project Template:") + "\n\n" +
			m.choiceView(1, "CLI Tool", "Standalone binary for execution in terminal") +
			m.choiceView(2, "LaunchDaemon", "Persistent background service for iOS") +
			m.choiceView(3, "Cydia Tweak", "Objective-C runtime injection / hooking")
	case 2:
		body = headerStyle.Render("3. Select Target Architecture:") + "\n\n" +
			m.choiceView(1, "Legacy iOS (armv7)", "Supports iPhone 4/4s/5/iPad2 on iOS 6/7/8/9") +
			m.choiceView(2, "Modern iOS (arm64)", "Supports iPhone 5s+ on iOS 10 through 17")
	case 3:
		body = headerStyle.Render("4. Deployment Strategy:") + "\n\n" +
			m.choiceView(1, "Wi-Fi Access", "Requires device IP address (Root password 'alpine')") +
			m.choiceView(2, "USB Connection", "High-speed stable connection via iproxy")
	case 4:
		body = headerStyle.Render("5. Enter Device IP Address:") + "\n\n" + m.textInput.View()
	case 5:
		body = headerStyle.Render("🚀 Ready to blast off? Confirm your settings:") + "\n\n" +
			m.summaryView() + "\n" +
			lipgloss.NewStyle().Foreground(lime).Render("   [Enter] to Initialize Project")
	}

	help := "\n\n" + helpStyle.Render("Press [q/Esc] to quit • Up/Down arrows omitted (use [1,2,3] for selection)")
	return header + body + help
}

func (m initModel) choiceView(id int, label, desc string) string {
	selection := blurredStyle.Render(fmt.Sprintf("[%d]", id))
	text := label
	if (m.template == id && m.step == 1) || (m.arch == id && m.step == 2) || (m.deploy == id && m.step == 3) {
		selection = focusedStyle.Render(fmt.Sprintf("[%d]", id))
		text = focusedStyle.Bold(true).Render(label)
	}
	return fmt.Sprintf("   %s %-15s %s\n", selection, text, blurredStyle.Render("- "+desc))
}

func (m initModel) summaryView() string {
	return fmt.Sprintf("   %-15s %s\n   %-15s %s\n   %-15s %s\n   %-15s %s\n   %-15s %s\n",
		"Project Name:", m.projectName,
		"Template:", m.templateStr(),
		"Architecture:", m.archStr(),
		"Deployment:", m.deployStr(),
		"Target IP:", m.ip,
	)
}

func (m initModel) templateStr() string {
	switch m.template {
	case 1: return "CLI Tool"
	case 2: return "LaunchDaemon"
	case 3: return "Cydia Tweak"
	default: return "CLI"
	}
}

func (m initModel) archStr() string {
	switch m.arch {
	case 1: return "Legacy armv7 (Retro)"
	case 2: return "Modern arm64 (Standard)"
	default: return "Modern"
	}
}

func (m initModel) deployStr() string {
	switch m.deploy {
	case 1: return "Web/Wi-Fi"
	case 2: return "USB Tunnel"
	default: return "USB"
	}
}

// Cobra Integration
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive wizard to initialize a GIOS project",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(initialInitModel())
		finalModel, err := p.Run()
		if err != nil {
			fmt.Printf("Fatal TUI Error: %v\n", err)
			os.Exit(1)
		}

		m := finalModel.(initModel)
		if m.done {
			// Actually perform the initialization
			performInitialization(m)
		}
	},
}

func performInitialization(m initModel) {
	fmt.Println("\n" + successStyle.Render("✨ Processing Project Core..."))

	conf := config.Config{}
	conf.Name = m.projectName
	conf.PackageID = m.packageID
	conf.Version = m.version
    if m.version == "" { conf.Version = "1.0.0" }
    
	isModern := (m.arch == 2)
	if !isModern {
		conf.Arch = "armv7"
		conf.GoVersion = "go1.14.15"
		conf.SDKVersion = "9.3"
		conf.Entitlements = "ents.plist"
	} else {
		conf.Arch = "arm64"
		conf.GoVersion = "system"
		conf.SDKVersion = "system"
		conf.Entitlements = "none"
	}

	if m.deploy == 2 {
		conf.Deploy.USB = true
		conf.Deploy.IP = "127.0.0.1"
	} else {
		conf.Deploy.IP = m.ip
	}

	conf.Output = "out_" + conf.Name
	conf.Main = "main.go"
	if m.template == 2 { conf.Daemon = true }
	if m.template == 3 {
		if !strings.HasSuffix(conf.Output, ".dylib") {
			conf.Output = conf.Output + ".dylib"
		}
	}

	if isModern {
		conf.Deploy.Path = "/var/jb/var/root"
	} else {
		conf.Deploy.Path = "/var/root/" + conf.Output
	}

	// Go modules
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		modGoVer := "1.14"
		if isModern { modGoVer = "1.23" }
		modData := fmt.Sprintf("module %s\n\ngo %s\n", conf.Name, modGoVer)
		ioutil.WriteFile("go.mod", []byte(modData), 0644)
	}

	jsonData, _ := json.MarshalIndent(conf, "", "  ")
	ioutil.WriteFile("gios.json", jsonData, 0644)

	createProjectTemplatesDetailed(m.template, conf)

	fmt.Println("\n   Usage:")
	fmt.Println("   - " + lipgloss.NewStyle().Foreground(cyan).Render("gios build") + " : Compiles for target device")
	fmt.Println("   - " + lipgloss.NewStyle().Foreground(cyan).Render("gios run") + "   : Build, Send and Execute")
	fmt.Println("")
}

func createProjectTemplatesDetailed(tType int, conf config.Config) {
	mainTemplate := `package main

import "fmt"

func main() {
	fmt.Println("Hello from GIOS on iOS!")
}
`
	if tType == 2 { // Daemon
		mainTemplate = `package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("[gios] Daemon starting...")
	for {
		fmt.Printf("[gios] Heartbeat at %v\n", time.Now())
		time.Sleep(30 * time.Second)
	}
}
`
	} else if tType == 3 { // Tweak
		mainTemplate = `package main

/*
#include <stdio.h>
#include <objc/runtime.h>

// Constructor (Runs on library load)
__attribute__((constructor))
static void init() {
    printf("[gios] Tweak successfully injected!\n");
}
*/
import "C"

func main() {}
`
	}

	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		ioutil.WriteFile("main.go", []byte(mainTemplate), 0644)
	}

	if conf.Arch == "armv7" {
		ents := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>platform-application</key>
	<true/>
	<key>com.apple.private.security.no-container</key>
	<true/>
</dict>
</plist>`
		ioutil.WriteFile("ents.plist", []byte(ents), 0644)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
}
