package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/nikitastrike/gios/pkg/config"
	"github.com/nikitastrike/gios/pkg/sdk"
	"github.com/nikitastrike/gios/pkg/utils"
)

// Bubbles & Styles (sharing from init.go if possible, or keeping local for simplicity)
var (
	cyanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true)
	grayStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	successColor = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF11")).Bold(true)
)

type sdkListModel struct {
	choices  []sdk.SDKInfo
	cursor   int
	selected map[int]struct{}
	quitting bool
	height   int
}

func (m sdkListModel) Init() tea.Cmd {
	return nil
}

func (m sdkListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m sdkListModel) View() string {
	if m.quitting {
		return "\n  Selection cancelled.\n"
	}

	s := cyanStyle.Render("📦 SDK MANAGEMENT ENGINE") + "\n\n"
	s += "Select an iOS SDK to download/install:\n\n"

	maxVisible := m.height - 8
	if maxVisible < 5 {
		maxVisible = 5
	}
	start := 0
	end := len(m.choices)

	if len(m.choices) > maxVisible {
		start = m.cursor - maxVisible/2
		if start < 0 { start = 0 }
		end = start + maxVisible
		if end > len(m.choices) {
			end = len(m.choices)
			start = end - maxVisible
		}
	}

	for i := start; i < end; i++ {
		choice := m.choices[i]
		cursor := "  "
		if m.cursor == i {
			cursor = cyanStyle.Render("> ")
		}
		
		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}

		line := fmt.Sprintf("%s [%s] %-25s %s", cursor, checked, choice.Name, grayStyle.Render("("+choice.Platform+")"))
		if m.cursor == i {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Render(line) + "\n"
		} else {
			s += line + "\n"
		}
	}

	if start > 0 {
		s = strings.Replace(s, "Select an iOS SDK", "Select an iOS SDK (↑ more)", 1)
	}
	if end < len(m.choices) {
		s += grayStyle.Render("       ... (↓ more)\n")
	}

	s += "\n" + grayStyle.Render("Use [arrows] to navigate • [Space/Enter] to select • [q] to quit\n")
	return s
}

// Cobra Commands
var sdkCmd = &cobra.Command{
	Use:   "sdk",
	Short: "Manage iOS SDKs (list, add, remove)",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var listSDKCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available and downloaded iOS SDKs",
	Run: func(cmd *cobra.Command, args []string) {
		sdk.ListSDKs()
	},
}

var addSDKCmd = &cobra.Command{
	Use:   "add",
	Short: "Select and download an iOS SDK from the manifest",
	Run: func(cmd *cobra.Command, args []string) {
		sdks, err := sdk.FetchAvailableSDKs()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		m := sdkListModel{
			choices:  sdks,
			selected: make(map[int]struct{}),
		}

		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fm := finalModel.(sdkListModel)
		if fm.quitting || len(fm.selected) == 0 {
			return
		}

		for i := range fm.selected {
			selectedSDK := fm.choices[i]
			targetPath := filepath.Join(config.GiosDir, "sdks", selectedSDK.Name)

			fmt.Printf("\n%s[gios]%s Pre-flight check: %s...", cyanStyle.Render(""), "", selectedSDK.Name)
			
			if err := sdk.ValidateRemoteURL(selectedSDK.URL); err != nil {
				fmt.Printf(" %sFAILED%s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Render(""), "")
				fmt.Printf("[!] Error: %v\n", err)
				continue
			}
			fmt.Printf(" %sSUCCESS%s\n", successColor.Render(""), "")

			err := sdk.EnsureSDKFromURL(selectedSDK.Name, targetPath, selectedSDK.URL, selectedSDK.Hash)
			if err != nil {
				fmt.Printf("[!] Download Error: %v\n", err)
			} else {
				fmt.Printf("\n%s[Success]%s %s installed correctly!\n", successColor.Render(""), "", selectedSDK.Name)
			}
		}
	},
}

var removeSDKCmd = &cobra.Command{
	Use:   "remove",
	Short: "Uninstall a downloaded iOS SDK",
	Run: func(cmd *cobra.Command, args []string) {
		downloaded := sdk.GetDownloadedSDKs()
		if len(downloaded) == 0 {
			fmt.Println("[gios] No SDKs currently installed.")
			return
		}

		// Selection menu for removal
		fmt.Println(cyanStyle.Render("\n🔥 SDK PRUNING ENGINE"))
		for i, s := range downloaded {
			fmt.Printf("  [%d] %s\n", i+1, s)
		}
		
		choiceStr := utils.Prompt("\nEnter number to remove", "")
		var choice int
		fmt.Sscanf(choiceStr, "%d", &choice)
		
		if choice >= 1 && choice <= len(downloaded) {
			selected := downloaded[choice-1]
			fmt.Printf("[gios] Deleting %s...\n", selected)
			sdk.RemoveSDKByName(selected)
			fmt.Println(successColor.Render("[+] Success! SDK removed."))
		}
	},
}

func init() {
	sdkCmd.AddCommand(listSDKCmd)
	sdkCmd.AddCommand(addSDKCmd)
	sdkCmd.AddCommand(removeSDKCmd)
	rootCmd.AddCommand(sdkCmd)
}
