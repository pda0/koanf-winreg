//go:build windows

// Package vault implements a koanf.Provider for Windows registry
// and returns a nested config map to provide it to koanf.
package winreg

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

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

func (s *WinReg) ReadBytes() ([]byte, error) {
	return nil, errors.New("winreg provider does not support this method")
}

func (s *WinReg) Read() (map[string]interface{}, error) {
	if retval, err := s.readKey(s.path, 1); err != nil {
		return nil, fmt.Errorf("unable to read registry %s", err.Error())
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
	k, err := registry.OpenKey(s.key, path, s.access)
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
					return nil, fmt.Errorf("%s: %s, %s", s.getKeyName(path), value, err.Error())
				}
				retval[value] = binary.LittleEndian.Uint32(tmpBuffer)
			case registry.BINARY:
				if retval[value], _, err = k.GetBinaryValue(value); err != nil {
					return nil, fmt.Errorf("%s: %s, %s", s.getKeyName(path), value, err.Error())
				}
			}
		}
	}

	// Reading subkeys
	if (s.maxDepth == 0) || (level < s.maxDepth) {
		if subKeys, err := k.ReadSubKeyNames(0); err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s: %s", s.getKeyName(path), err.Error())
		} else {
			for _, subKey := range subKeys {
				if retval[subKey], err = s.readKey(path+"\\"+subKey, level+1); err != nil {
					return nil, fmt.Errorf("%s: %s", s.getKeyName(path), err.Error())
				}
			}
		}
	}

	return retval, nil
}

func (s *WinReg) Watch(cb func(event interface{}, err error)) error {
	//TODO:
	return errors.New("winreg provider does not support this method")
}

func (c *Config) getAccess() (retval uint32) {
	retval = registry.READ

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
