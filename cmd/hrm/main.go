package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/clj/hrm-profile-tool/profile"
	"github.com/clj/hrm-profile-tool/render"
	"github.com/clj/hrm-profile-tool/utils/seekbufio"
	"github.com/clj/hrm-profile-tool/utils/text"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

type renderFn func(r io.ReadSeeker) (string, error)

var (
	textOutput     string
	profilePath    string
	svgOutput      string
	textVerbose    bool
	textLineNumber bool
	textInstNumber bool
	textRaw        bool
)

func parseInt(str string) int {
	base := 10
	if strings.HasPrefix(str, "0x") {
		base = 16
		str = str[2:]
	}
	i, err := strconv.ParseInt(str, base, strconv.IntSize)
	if err != nil {
		log.Fatal(err)
	}
	return int(i)
}

// Paths from: https://steamcommunity.com/app/375820/discussions/0/483368526585564846/
func profileFilePath() (string, error) {
	if profilePath != "" {
		return profilePath, nil
	}

	var profilePaths []string

	switch runtime.GOOS {
	case "windows":
		profilePaths = []string{`%APPDATA%\Human Resource Machine\profiles.bin`}
	case "darwin":
		profilePaths = []string{
			`~/Library/Application Support/Human Resource Machine/profiles.bin`,
			`~/Library/Containers/Tomorrow-Corporation.Human-Resource-Machine/Data/Library/Application Support/Human Resource Machine/profiles.bin`}
	case "linux":
		profilePaths = []string{`~/.local/share/Tomorrow\ Corporation/Human\ Resource\ Machine/profiles.bin`}
	default:
		return "", fmt.Errorf("unknown OS, cannot determine default profile path, please specify with --profile")
	}

	numExists := 0
	existsMap := make([]bool, len(profilePaths))
	var profilePath string
	for i, path := range profilePaths {
		var err error
		if path, err = homedir.Expand(path); err != nil {
			return "", err
		}
		if _, err := os.Stat(path); err == nil {
			numExists++
			existsMap[i] = true
			profilePath = path
		}
	}

	if numExists == 0 {
		return "", fmt.Errorf("no profiles found in default locations, use --profile to specify an alternative")

	}

	if numExists > 1 {
		availableProfiles := ""
		for i, exists := range existsMap {
			if exists {
				availableProfiles += fmt.Sprintf("    %s\n", profilePaths[i])
			}
		}
		return "", fmt.Errorf("multiple profiles exist, use --profile to specify one:\n" + availableProfiles)
	}

	return profilePath, nil
}

func openProfile() seekbufio.SeekableBufferedReader {
	profileFilePath, err := profileFilePath()
	if err != nil {
		log.Fatal(err)
	}
	reader, err := seekbufio.OpenSeekableBufferedReader(profileFilePath)
	if err != nil {
		log.Fatal(err)
	}
	return reader
}

func renderTab(args []string, outputFileName string, fn renderFn) {
	reader := openProfile()
	defer reader.Close()

	profileId := parseInt(args[0])
	if profileId != 1 {
		log.Fatal("Only profile slot 1 is supported currently")
	}
	floor := parseInt(args[1])
	tab := parseInt(args[2]) - 1
	floorIndex := profile.FloorToIndex(floor)
	tabStart := profile.TabStartAddr(profileId, floorIndex, tab)

	reader.Seek(tabStart, io.SeekStart)
	outputFile := os.Stdout
	if outputFileName != "" {
		var err error
		outputFile, err = os.Create(outputFileName)
		if err != nil {
			log.Fatal(err)
		}
	}
	str, err := fn(reader)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprint(outputFile, str)
}

func renderText(cmd *cobra.Command, args []string) {
	var options []render.RenderInstructionsTextOption
	if textVerbose || textLineNumber {
		options = append(options, render.ShowLineNumbers())
	}
	if textVerbose || textInstNumber {
		options = append(options, render.ShowInstructionNumbers())
	}
	if textVerbose || textRaw {
		options = append(options, render.ShowRawInstructions())
	}

	renderTab(args, textOutput, func(r io.ReadSeeker) (string, error) {
		tab_start, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return "", err
		}
		assembly, err := render.RenderInstructionsTextFromReader(r, options...)
		if err != nil {
			return "", err
		}
		comments_start := tab_start + profile.INSTRUCTIONS_SIZE
		r.Seek(comments_start, io.SeekStart)
		comments, err := render.RenderCommentsTextFromReader(r)
		if err != nil {
			return "", err
		}
		if comments != "" {
			assembly += "\n" + text.Wrap(comments, 80)
		}
		return assembly, nil
	})
}

func renderSVG(cmd *cobra.Command, args []string) {
	renderTab(args, svgOutput, render.RenderSVGFromReader)
}

func main() {
	var rootCmd = &cobra.Command{Use: "hrm"}

	var cmdRenderText = &cobra.Command{
		Use:   "text PROFILE PROGRAM TAB",
		Short: "Render Text",
		Long:  `Render a profile's program as text`,
		Args:  cobra.ExactArgs(3),
		Run:   renderText,
	}
	var cmdRenderSVG = &cobra.Command{
		Use:   "svg PROFILE PROGRAM TAB",
		Short: "Render SVG",
		Long:  `Render a single program as an SVG to stdout (or optionally directly to a file)`,
		Args:  cobra.ExactArgs(3),
		Run:   renderSVG,
	}

	rootCmd.Flags().StringVarP(&profilePath, "profile", "p", "", "`PATH` to a profiles.bin (otherwise search in default locations)")
	rootCmd.AddCommand(cmdRenderText)
	cmdRenderText.Flags().StringVarP(&textOutput, "output", "o", "", "`FILENAME` to write text assembly data to")
	cmdRenderText.Flags().BoolVarP(&textVerbose, "verbose", "v", false, "Show as much info as possible (same as -lir)")
	cmdRenderText.Flags().BoolVarP(&textLineNumber, "line-number", "l", false, "Show line numbers")
	cmdRenderText.Flags().BoolVarP(&textInstNumber, "inst-number", "i", false, "Show instruction numbers")
	cmdRenderText.Flags().BoolVarP(&textRaw, "raw", "r", false, "Show raw (hex) instructions")

	rootCmd.AddCommand(cmdRenderSVG)
	cmdRenderSVG.Flags().StringVarP(&svgOutput, "output", "o", "", "`FILENAME` to write SVG assembly data to")

	rootCmd.Execute()
}
