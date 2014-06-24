package main

// This implements an interface to external hooks
type HookRunner interface {
	Run(...string) error
}

// This implement hooks as external scripts
type hookRunnerCmdExec struct {
	cmd string
}

func NewScriptHook(cmd string) HookRunner {
	newhook := hookRunnerCmdExec{cmd: cmd}

	return newhook
}

func (hookRunnerCmdExec) Run(args ...string) (err error) {
	return
}

// This accepts hooks that are internal functions
//, primarily intended for testing
type hookRunnerFuncExec struct {
	f hookFunc
}

type hookFunc func(...string) error

func NewScriptHookr(hook hookFunc) HookRunner {
	newhook := hookRunnerFuncExec{f: hook}

	return newhook
}

func (hookRunnerFuncExec) Run(args ...string) (err error) {
	return
}