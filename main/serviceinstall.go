package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
)

const (
	// Contains the details of the service to be installed.
	// Please use this constant as follows as it depends on the RunCommand's version for some fields:
	// 	s := fmt.Sprintf(serviceDetails, runCommandVersion)
	serviceDetails = `
	[Unit]
	Description=Managed RunCommand Service

	[Service] 
	User=root
	Restart=always
	RestartSec=5
	WorkingDirectory=/var/lib/waagent/Micrososft.Cplat.Core.RunCommandHandlerLinux/%[1]s
	ExecStart=/var/lib/waagent/Micrososft.Cplat.Core.RunCommandHandlerLinux/%[1]s/run-command-handler

	[Install]
	WantedBy=multi-user.target`

	// The name of the systemd configuration file
	systemdServiceName = "managed-run-command.service"

	// The full path of the systemd configuration for the RunCommand service
	systemServiceFilePath = "/lib/systemd/system/" + systemdServiceName
)

// Installs RunCommand as a service on the client
func InstallRunCommandService(ctx *log.Context) (bool, error) {
	ctx.Log("message", "Trying to install run command as a service")
	_, err := createOrUpdateRunCommandService(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to install service")
	}

	return startService(ctx)
}

// Upgrades the RunCommand service with the latest available one (if any service exists)
func UpgradeRunCommandService(ctx *log.Context) (bool, error) {
	ctx.Log("message", "Trying to upgrade run command service")
	_, err := createOrUpdateRunCommandService(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to upgrade service")
	}

	ctx.Log("message", "Trying to reload service configuration after upgrade")
	_, err = exec.Command("systemctl", "start daemon-reload").Output()
	if err != nil {
		return false, errors.Wrap(err, "failed to reload service configuration")
	}

	ctx.Log("message", "Upgrade succeeded")
	return true, nil
}

// Stops and removes the installed service from the VM.
func UninstallRunCommandService(ctx *log.Context) (bool, error) {
	ctx.Log("message", "Trying to uninstall run command service")
	_, err := stopService(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to remove service")
	}

	err = os.Remove(systemServiceFilePath)
	if err != nil {
		return false, errors.Wrap(err, "failed to delete systemd configuration")
	}

	ctx.Log("message", "Run command service has been uninstalled")
	return true, nil
}

// Checks if the service is installed by checking for the presence of the systemd configuration file
func RunCommandServiceIsInstalled(ctx *log.Context) (bool, error) {
	ctx.Log("message", "Checking if runcommand has previously been installed")
	_, err := os.Stat(systemServiceFilePath)

	if errors.Is(err, os.ErrNotExist) {
		ctx.Log("message", "Service does not exists")
		return false, nil
	}

	if err != nil {
		return false, errors.Wrap(err, "failed to check if systemd configuration file exists")
	}

	ctx.Log("message", "Systemd file exists. Service has been installed before")
	return true, nil
}

// Updates the version of RunCommand to execute.
// It will update the 'WorkingDirectory' and 'ExecStart' paths of the systemd configuration.
// If this is the first time the method is getting invoked, then it will create the config file with the required details.
// Subsequent calls will update the version of RunCommand to use.
func createOrUpdateRunCommandService(ctx *log.Context) (bool, error) {
	// TODO: Get the latest runCommand version
	runCommandVersion := "2.42.0"
	systemdConfig := fmt.Sprintf(serviceDetails, runCommandVersion)

	ctx.Log("message", "Using run command version: "+runCommandVersion)
	err := os.WriteFile(systemServiceFilePath, []byte(systemdConfig), 0666)
	if err != nil {
		return false, errors.Wrap(err, "failed to write systemd configuration for runcommand version: "+runCommandVersion)
	}

	ctx.Log("message", "File to start run command as a service was successfully created")
	return true, nil
}

// Starts the RunCommand service by invoking 'systemctl start'
func startService(ctx *log.Context) (bool, error) {
	ctx.Log("message", "Trying to start run command service")
	_, err := exec.Command("systemctl", "start", "runService@"+systemdServiceName).Output()
	if err != nil {
		return false, errors.Wrap(err, "failed to start service")
	}

	ctx.Log("message", "Run command service has started")
	return true, nil
}

// Stops the RunCommand service by invoking 'systemctl stop'
func stopService(ctx *log.Context) (bool, error) {
	ctx.Log("message", "Trying to stop run command service")
	_, err := exec.Command("systemctl", "stop", systemdServiceName).Output()
	if err != nil {
		return false, errors.Wrap(err, "failed to stop service")
	}

	ctx.Log("message", "Run command service has been stopped")
	return true, nil
}
