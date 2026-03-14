package cli

import (
	"fmt"

	"github.com/mirtle/post-cli/internal/post"
)

type shortcutCommand struct {
	Type              string
	DefaultTTLMinutes int
	AllowFileContent  bool
	RequireFilePath   bool
}

var shortcutCommands = map[string]shortcutCommand{
	"md": {
		Type:              "md2html",
		DefaultTTLMinutes: 10080,
		AllowFileContent:  true,
	},
	"qr": {
		Type:              "qrcode",
		DefaultTTLMinutes: 10080,
		AllowFileContent:  true,
	},
	"file": {
		Type:              "file",
		DefaultTTLMinutes: 10080,
		RequireFilePath:   true,
	},
	"html": {
		Type:              "html",
		DefaultTTLMinutes: 10080,
		AllowFileContent:  true,
	},
	"text": {
		Type:              "text",
		DefaultTTLMinutes: 10080,
		AllowFileContent:  true,
	},
	"url": {
		Type:              "url",
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

	if options.Type != "" && options.Type != definition.Type {
		return options, fmt.Errorf("option --type/--convert is not supported with shortcut command '%s'", command)
	}

	options.Type = definition.Type
	if options.TTL == nil {
		defaultTTL := definition.DefaultTTLMinutes
		options.TTL = &defaultTTL
	}

	if definition.RequireFilePath {
		if options.ReadClipboard {
			return options, fmt.Errorf("option --read-clipboard is not supported with shortcut command '%s'", command)
		}
		options.WriteClipboard = true

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
	options.ReadClipboard = true
	options.WriteClipboard = true

	return options, nil
}
