## gphq completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(gphq completion bash)

To load completions for every new session, execute once:

#### Linux:

	gphq completion bash > /etc/bash_completion.d/gphq

#### macOS:

	gphq completion bash > $(brew --prefix)/etc/bash_completion.d/gphq

You will need to start a new shell for this setup to take effect.


```
gphq completion bash
```

### Options

```
  -h, --help              help for bash
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

