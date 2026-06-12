## gphq completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	gphq completion fish | source

To load completions for every new session, execute once:

	gphq completion fish > ~/.config/fish/completions/gphq.fish

You will need to start a new shell for this setup to take effect.


```
gphq completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq completion](gphq_completion.md)	 - Generate the autocompletion script for the specified shell

