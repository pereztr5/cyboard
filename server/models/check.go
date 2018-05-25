package models

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

type Check struct {
	Name     string `mapstructure:"check_name"`
	Filename string
	Script   *exec.Cmd
	Args     string
	Points   []int
	Disable  bool
}

func (c *Check) String() string {
	return fmt.Sprintf(`Check{name=%q, fullcmd="%s %s", pts=%v}`,
		c.Name, filepath.Base(c.Script.Path), c.Args, c.Points)
}
