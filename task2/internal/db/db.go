package db

import (
	"fmt"
	"log/slog"
	"sync"
)

var store sync.Map

func Get(key any) (any, bool) {
	return store.Load(key)
}

func ProcessWrite(command, key string, value, oldValue *string) (bool, error) {
	if err := validateCommand(command); err != nil {
		slog.Error("Invalid command", "command", command)
		return false, err
	}

	switch command {
	case "CREATE":
		return handleCreate(key, value)
	case "DELETE":
		return handleDelete(key)
	case "UPDATE":
		return handleUpdate(key, value)
	case "CAS":
		return handleCAS(key, value, oldValue)
	default:
		slog.Error("Invalid command: expected CREATE/DELETE/UPDATE/CAS", "command", command)
		return false, fmt.Errorf("Invalid command: expected CREATE/DELETE/UPDATE/CAS, got %v", command)
	}
}

func validateCommand(command string) error {
	if len(command) == 0 {
		return fmt.Errorf("Empty command")
	}
	return nil
}

func handleCreate(key string, value *string) (bool, error) {
	if value == nil {
		slog.Error("Null value for CREATE", "key", key)
		return false, fmt.Errorf("Null value")
	}
	if _, exists := store.Load(key); exists {
		return false, fmt.Errorf("Key '%v' already exists", key)
	}
	store.Store(key, *value)
	return true, nil
}

func handleDelete(key string) (bool, error) {
	_, existed := store.LoadAndDelete(key)
	return existed, nil
}

func handleUpdate(key string, value *string) (bool, error) {
	if value == nil {
		slog.Error("Null value for UPDATE", "key", key)
		return false, fmt.Errorf("Null value")
	}
	if _, exists := store.Load(key); !exists {
		return false, fmt.Errorf("Key '%v' does not exist", key)
	}
	store.Store(key, *value)
	return true, nil
}

func handleCAS(key string, value, oldValue *string) (bool, error) {
	if value == nil {
		slog.Error("Null new value for CAS", "key", key)
		return false, fmt.Errorf("Null new value")
	}
	if oldValue == nil {
		slog.Error("Null old value for CAS", "key", key)
		return false, fmt.Errorf("Null old value")
	}
	return store.CompareAndSwap(key, *oldValue, *value), nil
}
