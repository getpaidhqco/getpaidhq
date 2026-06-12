## gphq completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(gphq completion zsh)

To load completions for every new session, execute once:

#### Linux:

	gphq completion zsh > "${fpath[1]}/_gphq"

#### macOS:

	gphq completion zsh > $(brew --prefix)/share/zsh/site-functions/_gphq

You will need to start a new shell for this setup to take effect.


```
gphq completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
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

