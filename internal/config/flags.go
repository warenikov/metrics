package config

import "os"

// flagArgs возвращает аргументы командной строки без имени программы.
// Используется для упрощения тестирования через подмену os.Args.
func flagArgs() []string {
	if len(os.Args) > 1 {
		return os.Args[1:]
	}
	return nil
}
