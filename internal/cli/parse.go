package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mirtle/post-cli/internal/post"
)

func parseNewOptions(args []string) (post.NewOptions, error) {
	expandedArgs, err := expandCombinedBooleanFlags(args)
	if err != nil {
		return post.NewOptions{}, err
	}

	options := post.NewOptions{
		Method: http.MethodPost,
	}

	for index := 0; index < len(expandedArgs); {
		arg := expandedArgs[index]
		switch arg {
		case "-s", "--slug":
			value, nextIndex, err := nextValue(args, index)
			if err != nil {
				return options, fmt.Errorf("option %s requires a value", arg)
			}
			options.Slug = value
			index = nextIndex
		case "-t", "--ttl":
			value, nextIndex, err := nextValue(args, index)
			if err != nil {
				return options, fmt.Errorf("option %s requires a number (minutes)", arg)
			}
			ttl, convertErr := strconv.Atoi(value)
			if convertErr != nil {
				return options, fmt.Errorf("option %s requires a number (minutes)", arg)
			}
			options.TTL = &ttl
			index = nextIndex
		case "-u", "--update":
			options.Method = http.MethodPut
			index++
		case "-y", "--no-confirm":
			options.SkipConfirm = true
			index++
		case "-r", "--read-clipboard":
			options.ReadClipboard = true
			index++
		case "-w", "--write-clipboard":
			options.WriteClipboard = true
			index++
		case "-x", "--export":
			options.Export = true
			index++
		case "-f", "--file":
			value, nextIndex, err := nextValue(args, index)
			if err != nil {
				return options, fmt.Errorf("option %s requires a file path", arg)
			}
			options.FilePath = value
			index = nextIndex
		case "-c", "--convert":
			value, nextIndex, err := nextValue(args, index)
			if err != nil {
				return options, fmt.Errorf("option %s requires a value: html|md2html|url|text|qrcode|file", arg)
			}
			if !isValidConvert(value) {
				return options, fmt.Errorf("invalid convert value '%s'. Must be one of: html, md2html, url, text, qrcode, file", value)
			}
			options.Convert = value
			index = nextIndex
		case "--":
			options.Args = append(options.Args, expandedArgs[index+1:]...)
			return options, nil
		default:
			if strings.HasPrefix(arg, "-") {
				return options, fmt.Errorf("unknown option '%s'. Try: post help", arg)
			}
			options.Args = append(options.Args, expandedArgs[index:]...)
			return options, nil
		}
	}

	return options, nil
}

func expandCombinedBooleanFlags(args []string) ([]string, error) {
	expanded := make([]string, 0, len(args))
	for _, arg := range args {
		if len(arg) <= 2 || !strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") {
			expanded = append(expanded, arg)
			continue
		}

		for _, shortFlag := range arg[1:] {
			if isBooleanShortFlag(shortFlag) {
				expanded = append(expanded, "-"+string(shortFlag))
				continue
			}
			if isValueShortFlag(shortFlag) {
				return nil, fmt.Errorf("option '-%c' requires a value and cannot be combined", shortFlag)
			}
			return nil, fmt.Errorf("unknown option '-%c'. Try: post help", shortFlag)
		}
	}
	return expanded, nil
}

func isBooleanShortFlag(shortFlag rune) bool {
	switch shortFlag {
	case 'u', 'y', 'x', 'r', 'w':
		return true
	default:
		return false
	}
}

func isValueShortFlag(shortFlag rune) bool {
	switch shortFlag {
	case 'f', 's', 't', 'c':
		return true
	default:
		return false
	}
}

func parsePathExportOptions(args []string, command string) (string, bool, error) {
	export := false
	index := 0
	for index < len(args) {
		arg := args[index]
		switch arg {
		case "-x", "--export":
			export = true
			index++
		case "--":
			index++
			if index < len(args) {
				return args[index], export, nil
			}
			return "", export, nil
		default:
			if strings.HasPrefix(arg, "-") {
				return "", false, fmt.Errorf("unknown option '%s'. Try: post help", arg)
			}
			return arg, export, nil
		}
	}

	if command == "rm" {
		return "", export, nil
	}
	return "", export, nil
}

func shouldPrependNew(args []string) bool {
	if len(args) == 0 {
		return true
	}

	switch args[0] {
	case "new", "md", "qr", "file", "html", "text", "url", "ls", "export", "rm", "help", "completion", "version", "--help", "-h", "--version", "-v":
		return false
	default:
		return true
	}
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func nextValue(args []string, index int) (string, int, error) {
	if index+1 >= len(args) {
		return "", index, io.EOF
	}
	return args[index+1], index + 2, nil
}

func isValidConvert(value string) bool {
	switch value {
	case "html", "md2html", "url", "text", "qrcode", "file":
		return true
	default:
		return false
	}
}
