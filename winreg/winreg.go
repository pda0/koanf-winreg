//go:build windows

// Package winreg implements a koanf.Provider for Windows registry
// and returns a nested config map to provide it to koanf.
package winreg

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Determines which branch of the registry will be accessed:
// 32-bit or 64-bit.
const (
	RegAuto = iota
	Reg32Bit
	Reg64Bit
)

// Reflection of the registry package constants
// so you don't have to import it explicitly.
const (
	CLASSES_ROOT     = registry.CLASSES_ROOT
	CURRENT_USER     = registry.CURRENT_USER
	LOCAL_MACHINE    = registry.LOCAL_MACHINE
	USERS            = registry.USERS
	CURRENT_CONFIG   = registry.CURRENT_CONFIG
	PERFORMANCE_DATA = registry.PERFORMANCE_DATA
)

type Config struct {
	Key          registry.Key // Registry key
	Path         string       // A top path in selected key
	DefaultValue string       // The name of the value to which the default key value will be mapped
	MaxDepth     uint         // Maximum subkey reading depth
	Mode         int          // 32/64 bit registry branch, one of RegAuto/Reg32Bit/Reg64Bit constant
}

func (c *Config) getAccess() (retval uint32) {
	retval = 0

	switch c.Mode {
	case RegAuto:
		// do nothing
	case Reg32Bit:
		retval = retval | registry.WOW64_32KEY
	case Reg64Bit:
		retval = retval | registry.WOW64_64KEY
	default:
		panic("invalid winreg.Config.Mode value")
	}

	return
}

type WinReg struct {
	key          registry.Key
	path         string
	defaultValue string
	maxDepth     uint
	access       uint32
}

func Provider(cfg Config) *WinReg {
	return &WinReg{
		key:          cfg.Key,
		path:         cfg.Path,
		defaultValue: cfg.DefaultValue,
		maxDepth:     cfg.MaxDepth,
		access:       cfg.getAccess(),
	}
}

func (s *WinReg) getAccess(base uint32) uint32 {
	return base | s.access
}

func (s *WinReg) ReadBytes() ([]byte, error) {
	return nil, errors.New("winreg provider does not support this method")
}

func (s *WinReg) Read() (map[string]interface{}, error) {
	if retval, err := s.readKey(s.path, 1); err != nil {
		return nil, fmt.Errorf("unable to read registry, %s", err.Error())
	} else {
		return retval, nil
	}
}

func (s *WinReg) getKeyName(path string) string {
	switch s.key {
	case CLASSES_ROOT:
		return fmt.Sprintf("HKCR\\%s", path)
	case CURRENT_USER:
		return fmt.Sprintf("HKCU\\%s", path)
	case LOCAL_MACHINE:
		return fmt.Sprintf("HKLM\\%s", path)
	case USERS:
		return fmt.Sprintf("HKU\\%s", path)
	case CURRENT_CONFIG:
		return fmt.Sprintf("HKCC\\%s", path)
	case PERFORMANCE_DATA:
		return fmt.Sprintf("HKPD\\%s", path)
	default:
		return path
	}
}

func (s *WinReg) readKey(path string, level uint) (map[string]interface{}, error) {
	k, err := registry.OpenKey(s.key, path, s.getAccess(registry.READ))
	if err != nil {
		return nil, fmt.Errorf("%s: %s", s.getKeyName(path), err.Error())
	}
	defer k.Close()

	retval := make(map[string]interface{})
	// Reading key values
	if values, err := k.ReadValueNames(0); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: %s", s.getKeyName(path), err.Error())
	} else {
		var (
			koanfValue string
			tmpBuffer  []byte
			tmpStr     string
			typ        uint32
		)

		for _, value := range values {
			if _, typ, err = k.GetValue(value, nil); err != nil {
				return nil, fmt.Errorf("%s: %s, %s", s.getKeyName(path), value, err.Error())
			}
			switch typ {
			case registry.SZ:
				// Is it default key value
				if value == "" {
					if s.defaultValue == "" {
						continue
					}
					koanfValue = s.defaultValue
				} else {
					koanfValue = value
				}
				if retval[koanfValue], _, err = k.GetStringValue(value); err != nil {
					return nil, fmt.Errorf("%s: %s, %s", s.getKeyName(path), value, err.Error())
				}
			case registry.EXPAND_SZ:
				if tmpStr, _, err = k.GetStringValue(value); err != nil {
					return nil, fmt.Errorf("%s: %s, %s", s.getKeyName(path), value, err.Error())
				}
				if retval[value], err = registry.ExpandString(tmpStr); err != nil {
					return nil, fmt.Errorf("%s: %s, %s", s.getKeyName(path), value, err.Error())
				}
			case registry.MULTI_SZ:
				if retval[value], _, err = k.GetStringsValue(value); err != nil {
					return nil, fmt.Errorf("%s: %s, %s", s.getKeyName(path), value, err.Error())
				}
			case registry.DWORD, registry.QWORD:
				if retval[value], _, err = k.GetIntegerValue(value); err != nil {
					return nil, fmt.Errorf("%s: %s, %s", s.getKeyName(path), value, err.Error())
				}
			case registry.DWORD_BIG_ENDIAN:
				if len(tmpBuffer) == 0 {
					tmpBuffer = make([]byte, 4)
				}
				if _, _, err = k.GetValue(value, tmpBuffer); err != nil {
					return nil, fmt.Errorf("%s: %s, %v", s.getKeyName(path), value, err)
				}
				retval[value] = binary.LittleEndian.Uint32(tmpBuffer)
			case registry.BINARY:
				if retval[value], _, err = k.GetBinaryValue(value); err != nil {
					return nil, fmt.Errorf("%s: %s, %v", s.getKeyName(path), value, err)
				}
			}
		}
	}

	// Reading subkeys
	if (s.maxDepth == 0) || (level < s.maxDepth) {
		if subKeys, err := k.ReadSubKeyNames(0); err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s: %v", s.getKeyName(path), err)
		} else {
			for _, subKey := range subKeys {
				if retval[subKey], err = s.readKey(path+"\\"+subKey, level+1); err != nil {
					return nil, fmt.Errorf("%s: %v", s.getKeyName(path), err)
				}
			}
		}
	}

	return retval, nil
}

// Watch() watches the registry key and triggers a callback when it changes.
// Due to the nature of the Windows API, you cannot flexibly choose the depth
// of change tracking. If MaxDepth is not set to 1 in the provider, changes
// will be monitored to the full depth.
// If the monitored top-level key is deleted, the function will stop
// notifications, even if a key with the same name will create again. You must
// call the Watch() method again.
func (s *WinReg) Watch(cb func(event interface{}, err error)) error {
	const filter uint32 = REG_NOTIFY_CHANGE_NAME | REG_NOTIFY_CHANGE_LAST_SET

	k, err := registry.OpenKey(s.key, s.path, s.getAccess(registry.NOTIFY))
	if err != nil {
		return fmt.Errorf("failed to open registry key %s: %v", s.getKeyName(s.path), err)
	}

	// We need this complication because the function starts the goroutine,
	// but we cannot exit the function until the monitoring has actually started.
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		k.Close()
		return fmt.Errorf("watch failed: %v", err)
	}
	err = regNotifyChangeKeyValue(syscall.Handle(k), (s.maxDepth != 1), filter, event, true)
	if err != nil {
		k.Close()
		windows.Close(event)
		return fmt.Errorf("watch failed: %v", err)
	}

	go func() {
		var (
			waitResult uint32
			err        error
		)

		defer k.Close()
		defer windows.Close(event)
		for {
			waitResult, err = windows.WaitForSingleObject(event, windows.INFINITE)
			if err != nil {
				// The  windows.WaitForSingleObject() wrapper will assign
				// a non-nil value to err if the API function returns
				// WAIT_FAILED.
				cb(nil, fmt.Errorf("watch failed: %v", err))
				return
			}

			switch waitResult {
			case windows.WAIT_OBJECT_0:
				if err = windows.ResetEvent(event); err != nil {
					cb(nil, fmt.Errorf("watch failed: %v", err))
					return
				}
				// RegNotifyChangeKeyValue is a one-time function, according
				// to the documentation, we need to call it again to get the
				// next event.
				if err = regNotifyChangeKeyValue(syscall.Handle(k), (s.maxDepth != 1), filter, event, true); err != nil {
					cb(nil, fmt.Errorf("watch failed: %v", err))
					return
				}

				cb(nil, nil)
			case windows.WAIT_ABANDONED:
				// The program was terminated.
				return
			}
		}
	}()

	return nil
}

var (
	advapi32                    = syscall.NewLazyDLL("Advapi32.dll")
	procRegNotifyChangeKeyValue = advapi32.NewProc("RegNotifyChangeKeyValue")
)

const (
	REG_NOTIFY_CHANGE_NAME       = uint32(0x00000001)
	REG_NOTIFY_CHANGE_ATTRIBUTES = uint32(0x00000002)
	REG_NOTIFY_CHANGE_LAST_SET   = uint32(0x00000004)
	REG_NOTIFY_CHANGE_SECURITY   = uint32(0x00000008)
	REG_NOTIFY_THREAD_AGNOSTIC   = uint32(0x10000000)
)

func regNotifyChangeKeyValue(key syscall.Handle, watchSubtree bool, notifyFilter uint32, event windows.Handle, asynchronous bool) (regerrno error) {
	var _p0, _p1 uint32
	if watchSubtree {
		_p0 = 1
	}
	if asynchronous {
		_p1 = 1
	}
	r0, _, _ := syscall.Syscall6(procRegNotifyChangeKeyValue.Addr(), 5, uintptr(key), uintptr(_p0), uintptr(notifyFilter), uintptr(event), uintptr(_p1), 0)
	if r0 != 0 {
		regerrno = syscall.Errno(r0)
	}
	return
}
