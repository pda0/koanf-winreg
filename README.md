**winreg** is a [koanf](https://github.com/knadh/koanf) provider, which makes it possible to read data from the Windows registry.

### Installation

`go get -u github.com/pda0/koanf-winreg`

### Contents

- [Concepts](#concepts)
- [Reading config from Windows registry](#reading-config-from-windows-registry)
- [Watching registry key for changes](#watching-registry-key-for-changes)

### Concepts

`winreg.Provider` returns a `koanf.Provider` interface implementation that
reads Windows registry values into the koanf configuration tree.

You can use integer values, integers as booleans, strings, multiline values,
or even strings expanded with environment variables.

There are default key values in the Windows registry that have no name. Koanf
does not support such values, so if you want to read them you must specify
the name to which they will be mapped through the Config structure.

```go
winreg.Config{Key: <key>, Path: <path>, DefaultValue: "Default"}
```

Sometimes the subkey tree can be very large and undesirable to load into
memory. In this case, you can limit the maximum depth using the MatDepth
parameter. The counting corresponding only to the top-level key without
childrens starts at one. Zero means no depth limit.

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

### Watching registry key for changes

The `winreg.Provider` interface has a `Watch(cb)` method that asks a provider
to watch for changes and trigger the given callback that can live reload the
configuration.

Due to the nature of the Windows API, you cannot flexibly choose the depth
of change tracking. If MaxDepth is not set to 1 in the provider, changes
will be monitored to the full depth.

If the monitored top-level key is deleted, the function will stop
notifications, even if a key with the same name will create again. You must
call the Watch() method again.

```go
package main

import (
	"log"

	"github.com/knadh/koanf"
	"github.com/pda0/koanf-winreg/winreg"
	"golang.org/x/sys/windows/registry"
)

const testKey = "{26FB54D3-C8FF-4CD8-9D78-E1365170B217}"

// Global koanf instance. Use "." as the key path delimiter. This can be "/" or any character.
var k = koanf.New(".")

func main() {
	// Creating test key/data
	r, _, err := registry.CreateKey(registry.CURRENT_USER, "SOFTWARE\\"+testKey, registry.ALL_ACCESS)
	if err != nil {
		log.Fatalf("Unable to create test key: %v", err)
	}
	defer func() {
		r.Close()
		registry.DeleteKey(registry.CURRENT_USER, "SOFTWARE\\"+testKey)
	}()
	if err := r.SetDWordValue("IntVal", 100); err != nil {
		log.Fatalf("Unable to create test value: %v", err)
	}
	if err := r.SetStringValue("StrParam", "Hello world!"); err != nil {
		log.Fatalf("Unable to create test value: %v", err)
	}

	// Load registry key.
	p := winreg.Provider(winreg.Config{Key: winreg.CURRENT_USER, Path: "SOFTWARE\\" + testKey})
	if err := k.Load(p, nil); err != nil {
		log.Fatalf("error loading config: %v", err)
	}
	k.Print()

	// Watch the registry key and get a callback on change. The callback
	// can do whatever, like re-load the configuration.
	// Winreg provider always returns a nil `event`.
	err = p.Watch(func(event interface{}, err error) {
		if err != nil {
			log.Printf("watch error: %v", err)
			return
		}

		// Throw away the old config and load a fresh copy.
		log.Println("Config changed. Reloading ...")
		k = koanf.New(".")
		k.Load(p, nil)
		k.Print()
	})
	if err != nil {
		log.Fatalf("Unable to watch registry key: %v", err)
	}

	// Block forever (and manually make a change to registry)
	// to reload the config.
	log.Printf("Waiting forever. Try making a change to HKCU\\SOFTWARE\\%s key to live reload", testKey)
	<-make(chan bool)
}

```
