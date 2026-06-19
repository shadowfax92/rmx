package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var Version = "dev"

var ErrCancelled = errors.New("")

var (
	clrCyan    = lipgloss.Color("6")
	clrHiGreen = lipgloss.Color("10")
	clrYellow  = lipgloss.Color("11")
	clrRed     = lipgloss.Color("9")
)

func helpHeader(s string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(clrCyan).Render(s)
}

func helpCmdCol(s string) string {
	return lipgloss.NewStyle().Foreground(clrHiGreen).Render(s)
}

func helpHint(s string) string {
	return lipgloss.NewStyle().Faint(true).Render(s)
}

func helpAliases(aliases []string) string {
	return lipgloss.NewStyle().Foreground(clrYellow).Render(fmt.Sprintf("(aliases: %s)", strings.Join(aliases, ", ")))
}

var groupOrder = []string{
	"Sessions:",
	"Output:",
	"Input:",
	"Other:",
}

func groupedHelp(cmd *cobra.Command) string {
	groups := map[string][]*cobra.Command{}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() && c.Name() != "help" {
			continue
		}
		g := c.Annotations["group"]
		if g == "" {
			g = "Other:"
		}
		groups[g] = append(groups[g], c)
	}

	var b strings.Builder
	for _, name := range groupOrder {
		cmds, ok := groups[name]
		if !ok {
			continue
		}
		b.WriteString("\n" + helpHeader(name) + "\n")
		for _, c := range cmds {
			line := "  " + helpCmdCol(fmt.Sprintf("%-10s", c.Name())) + " " + c.Short
			if len(c.Aliases) > 0 {
				line += " " + helpAliases(c.Aliases)
			}
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}

const usageTemplate = `{{helpHeader "Usage:"}}{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

{{helpHeader "Aliases:"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{helpHeader "Examples:"}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}
{{groupedHelp .}}{{end}}{{if .HasAvailableLocalFlags}}

{{helpHeader "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{helpHeader "Global Flags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

{{helpHint (printf "Use \"%s [command] --help\" for more information." .CommandPath)}}{{end}}
`

var rootCmd = &cobra.Command{
	Use:           "rmx",
	Short:         "fzf-powered rmux session wrapper",
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listCmd.RunE(cmd, args)
	},
}

func init() {
	cobra.AddTemplateFunc("helpHeader", helpHeader)
	cobra.AddTemplateFunc("helpCmdCol", helpCmdCol)
	cobra.AddTemplateFunc("helpAliases", helpAliases)
	cobra.AddTemplateFunc("helpHint", helpHint)
	cobra.AddTemplateFunc("groupedHelp", groupedHelp)

	rootCmd.SetUsageTemplate(usageTemplate)
}

// Execute runs the CLI root command and treats cancellation as a clean exit.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if errors.Is(err, ErrCancelled) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
