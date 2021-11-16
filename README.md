**winreg** is a [koanf](https://github.com/knadh/koanf) provider, which makes it possible to read data from the Windows registry.

### Installation

`go get -u github.com/pda0/koanf-winreg`

### Contents

- [Concepts](#concepts)
- [Reading config from Windows registry](#reading-config-from-windows-registry)

### Concepts

`winreg.Provider` returns a `koanf.Provider` interface implementation that reads Windows registry values into the koanf configuration tree.

You can use integer values, integers as booleans, strings, multiline values, or even strings expanded with environment variables.

There are default key values in the Windows registry that have no name. Koanf does not support such values, so if you want to read them you must specify the name to which they will be mapped through the Config structure.

```go
winreg.Config{Key: <key>, Path: <path>, DefaultValue: "Default"}
```

Sometimes the subkey tree can be very large and undesirable to load into memory. In this case, you can limit the maximum depth using the MatDepth parameter.The counting corresponding only to the top-level key without childrens starts at one. Zero means no depth limit.

```go
winreg.Config{Key: <key>, Path: <path>, MatDepth: 1}
```

### Reading config from Windows registry

```go
package main

import (
	"fmt"
	"log"

	"github.com/knadh/koanf"
	"github.com/pda0/koanf-winreg/winreg"
)

// Global koanf instance. Use "." as the key path delimiter. This can be "/" or any character.
var k = koanf.New(".")

func main() {
	// Load registry values.
	if err := k.Load(winreg.Provider(winreg.Config{Key: winreg.LOCAL_MACHINE, Path: "SOFTWARE\\Microsoft\\Windows NT", MaxDepth: 2}), nil); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	fmt.Println("Windows Internal Version: ", k.String("CurrentVersion.CurrentVersion"))
	fmt.Printf("Windows Public Version: %d.%d\n", k.Int("CurrentVersion.CurrentMajorVersionNumber"), k.Int("CurrentVersion.CurrentMinorVersionNumber"))
}

```
