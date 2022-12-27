package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// docker         run image <cmd> <params>
// go run main.go run image <cmd> <params>

func main()  {
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("Invalid command!")
	}
}

func run() {
	fmt.Printf("Running main process %v as %d\n", os.Args[2:], os.Getpid())

	// Reinvoke the process inside the new namespace with a child process
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Cloneflags is only available in Linux
	// CLONE_NEWUTS: create the process in a new namespace
	// CLONE_NEWPID: isolates processes
    // CLONE_NEWNS: isolates mounts
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		// By default, the new namspace shared with the host. Use Unshareflags to not share
		Unshareflags: syscall.CLONE_NEWNS,
	}

	cmd.Run()
}

func child() {
	fmt.Printf("Running child process %v as %d\n", os.Args[2:], os.Getpid())

	configCgroups()

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	syscall.Sethostname([]byte("container"))
	// Change the root of the container
	syscall.Chroot("./ubuntu-fs")
	// Change directory to root after chroot
	syscall.Chdir("/")
	// Mount /proc inside container so that `ps` command works
	syscall.Mount("proc", "proc", "proc", 0, "")

	cmd.Run()

	// Unmount /proc when the process finishes
	syscall.Unmount("/proc", 0)
}

func configCgroups() {
	cgroups := "/sys/fs/cgroup/"
	container := filepath.Join(cgroups, "container")
	os.Mkdir(container, 0755)
	ioutil.WriteFile(filepath.Join(container, "pids.max"), []byte("10"), 0700)
	ioutil.WriteFile(filepath.Join(container, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700)
}
