package systemd

import (
	"fmt"
	"log"

	"github.com/godbus/dbus/v5"
)

var systemd dbus.BusObject = nil

func Init() error {
	// Connect to the system bus.
	conn, err := dbus.SystemBus()
	if err != nil {
		log.Fatalf("Failed to connect to system bus: %v", err)
		return err
	}
	if conn == nil {
		log.Fatalf("System bus is nil")
	}
	// Get a reference to the systemd D-Bus object.
	if systemd == nil {
		systemd = conn.Object("org.freedesktop.systemd1", dbus.ObjectPath("/org/freedesktop/systemd1"))
	}
	return nil
}

func ReloadUnit(unit string, mode string) error {
	// Call the ReloadUnit method on the systemd manager.
	if systemd == nil {
		err := Init()
		if err != nil {
			log.Fatal("Failed to initialize systemd")
		}
	}
	var jobPath dbus.ObjectPath
	err := systemd.Call("org.freedesktop.systemd1.Manager.ReloadUnit", 0, unit, mode).Store(&jobPath)
	if err != nil {
		return fmt.Errorf("failed to reload unit %s: %v", unit, err)
	}

	fmt.Printf("Reload job queued for %s; job path: %s\n", unit, jobPath)
	return nil
}
