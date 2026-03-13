package cli

import (
	"fmt"

	"github.com/mirtle/post-cli/internal/post"
)

type shortcutCommand struct {
	Convert           string
	DefaultTTLMinutes int
	AllowFileContent  bool
	RequireFilePath   bool
}

var shortcutCommands = map[string]shortcutCommand{
	"md": {
		Convert:           "md2html",
		DefaultTTLMinutes: 10080,
		AllowFileContent:  true,
	},
	"qr": {
		Convert:           "qrcode",
		DefaultTTLMinutes: 10080,
		AllowFileContent:  true,
	},
	"file": {
		Convert:           "file",
		DefaultTTLMinutes: 10080,
		RequireFilePath:   true,
	},
	"html": {
		Convert:           "html",
		DefaultTTLMinutes: 10080,
		AllowFileContent:  true,
	},
	"text": {
		Convert:           "text",
		DefaultTTLMinutes: 10080,
		AllowFileContent:  true,
	},
	"url": {
		Convert:           "url",
		DefaultTTLMinutes: 10080,
		AllowFileContent:  true,
	},
}

func parseShortcutOptions(command string, args []string) (post.NewOptions, error) {
	definition, ok := shortcutCommands[command]
	if !ok {
		return post.NewOptions{}, fmt.Errorf("unknown command '%s'. Try: post help", command)
	}

	options, err := parseNewOptions(args)
	if err != nil {
		return options, err
	}

	if options.Convert != "" && options.Convert != definition.Convert {
		return options, fmt.Errorf("option --convert is not supported with shortcut command '%s'", command)
	}

	options.Convert = definition.Convert
	if options.TTL == nil {
		defaultTTL := definition.DefaultTTLMinutes
		options.TTL = &defaultTTL
	}

	if definition.RequireFilePath {
		if len(options.Args) > 1 {
			return options, fmt.Errorf("file command accepts a single file path")
		}
		if options.FilePath != "" && len(options.Args) == 1 {
			return options, fmt.Errorf("file command accepts either a positional file path or -f, not both")
		}
		if options.FilePath == "" && len(options.Args) == 1 {
			options.FilePath = options.Args[0]
			options.Args = nil
		}
		if options.FilePath == "" {
			return options, fmt.Errorf("file command requires a file path")
		}
		return options, nil
	}

	if !definition.AllowFileContent && options.FilePath != "" {
		return options, fmt.Errorf("option --file is not supported with shortcut command '%s'", command)
	}

	return options, nil
}
