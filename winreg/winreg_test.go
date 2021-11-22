//go:build windows

package winreg

import (
	"errors"
	"io"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/knadh/koanf"
	"golang.org/x/sys/windows/registry"
)

const (
	success = "\u2713"
	failed  = "\u2717"

	testKey = "{26FB54D3-C8FF-4CD8-9D78-E1365170B217}"
)

func TestParseRegistry(t *testing.T) {
	t.Log("Testing Windows registry provider.")
	{
		createTestData(t)
		defer deleteTestData(t)

		k := koanf.New(".")

		testID := 0
		t.Logf("\tTest %d:\tRead().", testID)
		{
			if err := k.Load(Provider(Config{Key: CURRENT_USER, Path: "SOFTWARE\\" + testKey, DefaultValue: "Default"}), nil); err != nil {
				t.Fatalf("\t%s\tUnable to read registry: %v.", failed, err)
				return
			}
			t.Logf("\t%s\tRegistry values was read.", success)
		}

		testID++
		t.Logf("\tTest %d:\treaded values.", testID)
		{
			allKeys := map[string]bool{
				"SubKeyA.Binary":   false,
				"SubKeyA.Expand":   false,
				"SubKeyA.Int64":    false,
				"SubKeyA.IntVal":   false,
				"SubKeyA.StrList":  false,
				"SubKeyA.StrValue": false,
				"SubKeyA.Sub Key":  false,
				"SubKeyB.Default":  false,
				"off":              false,
				"on":               false,
			}

			for _, key := range k.Keys() {
				if _, ok := allKeys[key]; !ok {
					t.Fatalf("\t%s\treaded keys check failed, got unexpected key \"%s\".", failed, key)
				}

				allKeys[key] = true
			}
			for key, value := range allKeys {
				if !value {
					t.Fatalf("\t%s\treaded keys check failed, key \"%s\" wasn't read.", failed, key)
				}
			}
			t.Logf("\t%s\tAll values read successfully.", success)
		}

		testID++
		t.Logf("\tTest %d:\tSubKeyA.Binary.", testID)
		{
			aBinary := k.String("SubKeyA.Binary")
			if aBinary != "[1 2 3]" {
				t.Fatalf("\t%s\tSubKeyA.Binary is invalid, got %s, expect [1 2 3].", failed, aBinary)
			}
			t.Logf("\t%s\tSubKeyA.Binary is valid.", success)
		}

		testID++
		t.Logf("\tTest %d:\tSubKeyA.Expand.", testID)
		{
			path := os.Getenv("PATH")
			aExpand := k.String("SubKeyA.Expand")
			if aExpand != "Some "+path {
				t.Fatalf("\t%s\tSubKeyA.Expand is invalid, got \"%s\", expect \"Some %s\".", failed, aExpand, path)
			}
			t.Logf("\t%s\tSubKeyA.Expand is valid.", success)
		}

		testID++
		t.Logf("\tTest %d:\tSubKeyA.Int64.", testID)
		{
			aInt64 := k.Int64("SubKeyA.Int64")
			if aInt64 != 5000000000 {
				t.Fatalf("\t%s\tSubKeyA.Int64 is invalid, got %d, expect 5000000000.", failed, aInt64)
			}
			t.Logf("\t%s\tSubKeyA.Int64 is valid.", success)
		}

		testID++
		t.Logf("\tTest %d:\tSubKeyA.IntVal.", testID)
		{
			aInt := k.Int("SubKeyA.IntVal")
			if aInt != 4000000000 {
				t.Fatalf("\t%s\tSubKeyA.IntVal is invalid, got %d, expect 4000000000.", failed, aInt)
			}
			t.Logf("\t%s\tSubKeyA.IntVal is valid.", success)
		}

		testID++
		t.Logf("\tTest %d:\tSubKeyA.IntVal.", testID)
		{
			aStrList := k.Strings("SubKeyA.StrList")
			if len(aStrList) != 4 {
				t.Fatalf("\t%s\tSubKeyA.StrList has invalid length, got %d, expect 4.", failed, len(aStrList))
			}
			if aStrList[0] != "Black cat" {
				t.Fatalf("\t%s\tSubKeyA.StrList[0] has invalid length, got \"%s\", expect \"Black cat\".", failed, aStrList[0])
			}
			if aStrList[1] != "sit on the mat" {
				t.Fatalf("\t%s\tSubKeyA.StrList[1] has invalid length, got \"%s\", expect \"sit on the mat\".", failed, aStrList[1])
			}
			if aStrList[2] != "and eat" {
				t.Fatalf("\t%s\tSubKeyA.StrList[2] has invalid length, got \"%s\", expect \"and eat\".", failed, aStrList[2])
			}
			if aStrList[3] != "the fat rat" {
				t.Fatalf("\t%s\tSubKeyA.StrList[3] has invalid length, got \"%s\", expect \"the fat rat\".", failed, aStrList[3])
			}
			t.Logf("\t%s\tSubKeyA.StrList is valid.", success)
		}

		testID++
		t.Logf("\tTest %d:\tSubKeyA.StrValue.", testID)
		{
			aStr := k.String("SubKeyA.StrValue")
			if aStr != "The quick brown fox jumps over the lazy dog" {
				t.Fatalf("\t%s\tSubKeyA.StrValue is invalid, got \"%s\", expect \"The quick brown fox jumps over the lazy dog\".", failed, aStr)
			}
			t.Logf("\t%s\tSubKeyA.StrValue is valid.", success)
		}

		testID++
		t.Logf("\tTest %d:\tSubKeyB.Default.", testID)
		{
			aStr := k.String("SubKeyB.Default")
			if aStr != "default value" {
				t.Fatalf("\t%s\tSubKeyB.Default is invalid, got \"%s\", expect \"default value\".", failed, aStr)
			}
			t.Logf("\t%s\tSubKeyB.Default is valid.", success)
		}

		testID++
		t.Logf("\tTest %d:\toff.", testID)
		{
			off := k.Bool("off")
			if off {
				t.Fatalf("\t%s\toff is invalid, got %v, expect false.", failed, off)
			}
			t.Logf("\t%s\toff is valid.", success)
		}

		testID++
		t.Logf("\tTest %d:\ton.", testID)
		{
			on := k.Bool("on")
			if !on {
				t.Fatalf("\t%s\ton is invalid, got %v, expect true.", failed, on)
			}
			t.Logf("\t%s\ton is valid.", success)
		}
	}
}

func TestFailMaxDapthRegistry(t *testing.T) {
	t.Log("Testing depth limit of Windows registry provider.")
	{
		createTestData(t)
		defer deleteTestData(t)

		testID := 0
		t.Logf("\tTest %d:\tdepth 3 limit.", testID)
		{
			allKeys := map[string]bool{
				"SubKeyA.Binary":   false,
				"SubKeyA.Expand":   false,
				"SubKeyA.Int64":    false,
				"SubKeyA.IntVal":   false,
				"SubKeyA.StrList":  false,
				"SubKeyA.StrValue": false,
				"SubKeyA.Sub Key":  false,
				"SubKeyB":          false,
				"off":              false,
				"on":               false,
			}
			k := koanf.New(".")
			if err := k.Load(Provider(Config{Key: CURRENT_USER, Path: "SOFTWARE\\" + testKey, MaxDepth: 3}), nil); err != nil {
				t.Fatalf("\t%s\tUnable to read registry: %v.", failed, err)
				return
			}

			for _, key := range k.Keys() {
				if _, ok := allKeys[key]; !ok {
					t.Fatalf("\t%s\treaded keys check failed, got unexpected key \"%s\".", failed, key)
				}

				allKeys[key] = true
			}
			for key, value := range allKeys {
				if !value {
					t.Fatalf("\t%s\treaded keys check failed, key \"%s\" wasn't read.", failed, key)
				}
			}
			t.Logf("\t%s\tAll values read successfully.", success)
		}

		testID++
		t.Logf("\tTest %d:\tdepth 2 limit.", testID)
		{
			allKeys := map[string]bool{
				"SubKeyA.Binary":   false,
				"SubKeyA.Expand":   false,
				"SubKeyA.Int64":    false,
				"SubKeyA.IntVal":   false,
				"SubKeyA.StrList":  false,
				"SubKeyA.StrValue": false,
				"SubKeyB":          false,
				"off":              false,
				"on":               false,
			}
			k := koanf.New(".")
			if err := k.Load(Provider(Config{Key: CURRENT_USER, Path: "SOFTWARE\\" + testKey, MaxDepth: 2}), nil); err != nil {
				t.Fatalf("\t%s\tUnable to read registry: %v.", failed, err)
				return
			}

			for _, key := range k.Keys() {
				if _, ok := allKeys[key]; !ok {
					t.Fatalf("\t%s\treaded keys check failed, got unexpected key \"%s\".", failed, key)
				}

				allKeys[key] = true
			}
			for key, value := range allKeys {
				if !value {
					t.Fatalf("\t%s\treaded keys check failed, key \"%s\" wasn't read.", failed, key)
				}
			}
			t.Logf("\t%s\tAll values read successfully.", success)
		}

		testID++
		t.Logf("\tTest %d:\tdepth 1 limit.", testID)
		{
			allKeys := map[string]bool{
				"off": false,
				"on":  false,
			}
			k := koanf.New(".")
			if err := k.Load(Provider(Config{Key: CURRENT_USER, Path: "SOFTWARE\\" + testKey, MaxDepth: 1}), nil); err != nil {
				t.Fatalf("\t%s\tUnable to read registry: %v.", failed, err)
				return
			}

			for _, key := range k.Keys() {
				if _, ok := allKeys[key]; !ok {
					t.Fatalf("\t%s\treaded keys check failed, got unexpected key \"%s\".", failed, key)
				}

				allKeys[key] = true
			}
			for key, value := range allKeys {
				if !value {
					t.Fatalf("\t%s\treaded keys check failed, key \"%s\" wasn't read.", failed, key)
				}
			}
			t.Logf("\t%s\tAll values read successfully.", success)
		}

		testID++
		t.Logf("\tTest %d:\tdepth 0 (no) limit.", testID)
		{
			allKeys := map[string]bool{
				"SubKeyA.Binary":   false,
				"SubKeyA.Expand":   false,
				"SubKeyA.Int64":    false,
				"SubKeyA.IntVal":   false,
				"SubKeyA.StrList":  false,
				"SubKeyA.StrValue": false,
				"SubKeyA.Sub Key":  false,
				"SubKeyB":          false,
				"off":              false,
				"on":               false,
			}
			k := koanf.New(".")
			if err := k.Load(Provider(Config{Key: CURRENT_USER, Path: "SOFTWARE\\" + testKey, MaxDepth: 0}), nil); err != nil {
				t.Fatalf("\t%s\tUnable to read registry: %v.", failed, err)
				return
			}

			for _, key := range k.Keys() {
				if _, ok := allKeys[key]; !ok {
					t.Fatalf("\t%s\treaded keys check failed, got unexpected key \"%s\".", failed, key)
				}

				allKeys[key] = true
			}
			for key, value := range allKeys {
				if !value {
					t.Fatalf("\t%s\treaded keys check failed, key \"%s\" wasn't read.", failed, key)
				}
			}
			t.Logf("\t%s\tAll values read successfully.", success)
		}
	}
}

func TestFailParseRegistry(t *testing.T) {
	t.Log("Testing Windows registry provider (fail).")
	{
		deleteTestData(t)

		k := koanf.New(".")

		testID := 0
		t.Logf("\tTest %d:\tRead() (non-existent key).", testID)
		{
			var err error
			if err = k.Load(Provider(Config{Key: CURRENT_USER, Path: "SOFTWARE\\" + testKey}), nil); err == nil {
				t.Fatalf("\t%s\tNon-existent key was read.", failed)
			}
			if err.Error() != "unable to read registry, HKCU\\SOFTWARE\\"+testKey+": The system cannot find the file specified." {
				t.Fatalf("\t%s\tInvalid error value, got \"%v\", expect \"%s\".", failed, err, "unable to read registry, HKCU\\SOFTWARE\\"+testKey+": The system cannot find the file specified.")
			}
			t.Logf("\t%s\tReading a non-existent key returned an error.", success)
		}
	}
}

func TestWatch(t *testing.T) {
	t.Log("Testing provider's Watch method.")
	{
		const eventTimeout = 5
		var active int32
		sc := make(chan bool)
		ec := make(chan error)
		createTestData(t)
		defer deleteTestData(t)

		k := koanf.New(".")
		p := Provider(Config{Key: CURRENT_USER, Path: "SOFTWARE\\" + testKey})
		if err := k.Load(p, nil); err != nil {
			t.Fatalf("\t%s\tUnable to read registry key values: %v", failed, err)
		}

		atomic.StoreInt32(&(active), int32(1))
		err := p.Watch(func(event interface{}, err error) {
			if atomic.LoadInt32(&active) == 0 {
				// We're done now
				return
			}

			if err != nil {
				ec <- err
				return
			}
			t.Log("\t\tConfig changed. Reloading ...")
			k = koanf.New(".")
			if err := k.Load(p, nil); err != nil {
				ec <- err
				return
			}
			sc <- true
		})
		if err != nil {
			t.Fatalf("\t%s\tWatch() method failed: %v", failed, err)
		}

		testID := 0
		t.Logf("\tTest %d:\twaiting for value to be changed.", testID)
		func() {
			var (
				r   registry.Key
				err error
			)
			if r, err = registry.OpenKey(registry.CURRENT_USER, "SOFTWARE\\"+testKey+"\\SubKeyA", registry.ALL_ACCESS); err != nil {
				atomic.StoreInt32(&(active), int32(0))
				t.Fatalf("\t%s\tUnable to open registry key: %v", failed, err)
			}
			defer r.Close()

			if err := r.SetDWordValue("IntVal", 200); err != nil {
				atomic.StoreInt32(&(active), int32(0))
				t.Fatalf("\t%s\tUnable to change value \"IntVal\": %v", failed, err)
			}

			timeout := time.After(eventTimeout * time.Second)
			for cont := true; cont; {
				select {
				case <-sc:
					// Successfully reloaded
					val := k.Int("SubKeyA.IntVal")
					if val != 200 {
						atomic.StoreInt32(&(active), int32(0))
						t.Fatalf("\t%s\tSubKeyA.IntVal is invalid, got %d, expect 200.", failed, val)
					}
					cont = false
				case err = <-ec:
					// An error has been returned
					atomic.StoreInt32(&(active), int32(0))
					t.Fatalf("\t%s\tAn error occurred during the config reload: %v", failed, err)
				case <-timeout:
					// Timeout exceeded
					atomic.StoreInt32(&(active), int32(0))
					t.Fatalf("\t%s\tTimeout exceeded while waiting for config reload.", failed)
				}
			}
			t.Logf("\t%s\tThe reload was successful.", success)
		}()

		testID++
		t.Logf("\tTest %d:\twaiting for value to be deleted.", testID)
		func() {
			var (
				r   registry.Key
				err error
			)
			if r, err = registry.OpenKey(registry.CURRENT_USER, "SOFTWARE\\"+testKey+"\\SubKeyA", registry.ALL_ACCESS); err != nil {
				atomic.StoreInt32(&(active), int32(0))
				t.Fatalf("\t%s\tUnable to open registry key: %v", failed, err)
			}
			defer r.Close()

			if err := r.DeleteValue("IntVal"); err != nil {
				atomic.StoreInt32(&(active), int32(0))
				t.Fatalf("\t%s\tUnable to delete value \"IntVal\": %v", failed, err)
			}

			timeout := time.After(eventTimeout * time.Second)
			for cont := true; cont; {
				select {
				case <-sc:
					// Successfully reloaded
					val := k.Int("SubKeyA.IntVal")
					if val != 0 {
						atomic.StoreInt32(&(active), int32(0))
						t.Fatalf("\t%s\tSubKeyA.IntVal is invalid, got %d, expect 0.", failed, val)
					}
					cont = false
				case err = <-ec:
					// An error has been returned
					atomic.StoreInt32(&(active), int32(0))
					t.Fatalf("\t%s\tAn error occurred during the config reload: %v", failed, err)
				case <-timeout:
					// Timeout exceeded
					atomic.StoreInt32(&(active), int32(0))
					t.Fatalf("\t%s\tTimeout exceeded while waiting for config reload.", failed)
				}
			}
			t.Logf("\t%s\tThe reload was successful.", success)
		}()

		testID++
		t.Logf("\tTest %d:\twaiting for value to be created.", testID)
		func() {
			var (
				r   registry.Key
				err error
			)
			if r, err = registry.OpenKey(registry.CURRENT_USER, "SOFTWARE\\"+testKey+"\\SubKeyA", registry.ALL_ACCESS); err != nil {
				atomic.StoreInt32(&(active), int32(0))
				t.Fatalf("\t%s\tUnable to open registry key: %v", failed, err)
			}
			defer r.Close()

			if err := r.SetDWordValue("IntVal", 100); err != nil {
				atomic.StoreInt32(&(active), int32(0))
				t.Fatalf("\t%s\tUnable to create value \"IntVal\": %v", failed, err)
			}

			timeout := time.After(eventTimeout * time.Second)
			for cont := true; cont; {
				select {
				case <-sc:
					// Successfully reloaded
					val := k.Int("SubKeyA.IntVal")
					if val != 100 {
						atomic.StoreInt32(&(active), int32(0))
						t.Fatalf("\t%s\tSubKeyA.IntVal is invalid, got %d, expect 100.", failed, val)
					}
					cont = false
				case err = <-ec:
					// An error has been returned
					atomic.StoreInt32(&(active), int32(0))
					t.Fatalf("\t%s\tAn error occurred during the config reload: %v", failed, err)
				case <-timeout:
					// Timeout exceeded
					atomic.StoreInt32(&(active), int32(0))
					t.Fatalf("\t%s\tTimeout exceeded while waiting for config reload.", failed)
				}
			}
			t.Logf("\t%s\tThe reload was successful.", success)
		}()

		testID++
		t.Logf("\tTest %d:\twaiting for subkey to be deleted.", testID)
		func() {
			var (
				r   registry.Key
				err error
			)
			if r, err = registry.OpenKey(registry.CURRENT_USER, "SOFTWARE\\"+testKey+"\\SubKeyA", registry.ALL_ACCESS); err != nil {
				atomic.StoreInt32(&(active), int32(0))
				t.Fatalf("\t%s\tUnable to open registry key: %v", failed, err)
			}
			defer r.Close()

			if err := registry.DeleteKey(r, "Sub Key"); err != nil {
				atomic.StoreInt32(&(active), int32(0))
				t.Fatalf("\t%s\tUnable to delete \"Sub Key\" key: %v", failed, err)
			}

			timeout := time.After(eventTimeout * time.Second)
			for cont := true; cont; {
				select {
				case <-sc:
					// Successfully reloaded
					if _, ok := k.KeyMap()["SubKeyA.Sub Key"]; ok {
						atomic.StoreInt32(&(active), int32(0))
						t.Fatalf("\t%s\t\"SubKeyA.Sub Key\" key is not deleted.", failed)
					}
					cont = false
				case err = <-ec:
					// An error has been returned
					atomic.StoreInt32(&(active), int32(0))
					t.Fatalf("\t%s\tAn error occurred during the config reload: %v", failed, err)
				case <-timeout:
					// Timeout exceeded
					atomic.StoreInt32(&(active), int32(0))
					t.Fatalf("\t%s\tTimeout exceeded while waiting for config reload.", failed)
				}
			}
			t.Logf("\t%s\tThe reload was successful.", success)
		}()

		atomic.StoreInt32(&(active), int32(0))
	}
}

func TestWatchFail(t *testing.T) {
	t.Log("Testing fails of provider's Watch method.")
	{
		testID := 0
		t.Logf("\tTest %d:\tnon-existent key.", testID)
		{
			p := Provider(Config{Key: CURRENT_USER, Path: "SOFTWARE\\" + testKey})
			err := p.Watch(func(event interface{}, err error) {})
			if err == nil {
				t.Fatalf("\t%s\tWatch() method succeeded.", failed)
			}
			t.Logf("\t%s\tWatch() of a non-existent key returns an error.", success)
		}
	}
}

func createTestData(t *testing.T) {
	k, exists, err := registry.CreateKey(registry.CURRENT_USER, "SOFTWARE\\"+testKey, registry.ALL_ACCESS)
	if err != nil {
		t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
	}

	if exists {
		// Such a key already exists, left over from a past unsuccessful test
		k.Close()
		deleteTestData(t)
		k, exists, err = registry.CreateKey(registry.CURRENT_USER, "SOFTWARE\\"+testKey, registry.ALL_ACCESS)
		if err != nil {
			t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
		}
		if exists {
			t.Fatalf("\t%s\tUnable to prepare test key.", failed)
		}
	}
	defer k.Close()

	if err := k.SetDWordValue("on", 1); err != nil {
		t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
	}
	if err := k.SetDWordValue("off", 0); err != nil {
		t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
	}

	ka, _, err := registry.CreateKey(k, "SubKeyA", registry.ALL_ACCESS)
	if err != nil {
		t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
	}
	defer ka.Close()

	if err := ka.SetBinaryValue("Binary", []byte{1, 2, 3}); err != nil {
		t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
	}
	if err := ka.SetExpandStringValue("Expand", "Some %PATH%"); err != nil {
		t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
	}
	if err := ka.SetQWordValue("Int64", 5000000000); err != nil {
		t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
	}
	if err := ka.SetDWordValue("IntVal", 4000000000); err != nil {
		t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
	}
	if err := ka.SetStringsValue("StrList", []string{"Black cat", "sit on the mat", "and eat", "the fat rat"}); err != nil {
		t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
	}
	if err := ka.SetStringValue("StrValue", "The quick brown fox jumps over the lazy dog"); err != nil {
		t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
	}

	if ks, _, err := registry.CreateKey(ka, "Sub Key", registry.ALL_ACCESS); err != nil {
		t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
	} else {
		ks.Close()
	}

	kb, _, err := registry.CreateKey(k, "SubKeyB", registry.ALL_ACCESS)
	if err != nil {
		t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
	}
	defer kb.Close()

	if err := kb.SetStringValue("", "default value"); err != nil {
		t.Fatalf("\t%s\tUnable to create test key: %v", failed, err)
	}
}

func deleteSubKey(t *testing.T, k registry.Key, name string) {
	if ks, err := registry.OpenKey(k, name, registry.ALL_ACCESS); err == nil {
		defer ks.Close()

		if subKeys, err := ks.ReadSubKeyNames(0); err != nil && !errors.Is(err, io.EOF) {
			t.Fatalf("Unable to delete test key: %v", err)
		} else {
			for _, subKey := range subKeys {
				deleteSubKey(t, ks, subKey)
			}

			if err := registry.DeleteKey(k, name); err != nil {
				t.Fatalf("\t%s\tUnable to delete test key: %v", failed, err)
			}
		}
	}
}

func deleteTestData(t *testing.T) {
	if k, err := registry.OpenKey(registry.CURRENT_USER, "SOFTWARE", registry.ALL_ACCESS); err == nil {
		defer k.Close()

		deleteSubKey(t, k, testKey)
	}
}
